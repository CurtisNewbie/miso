package miso

import (
	"container/list"
	"sync"
	"time"
)

// Simple local map-based cache.
//
// This should not be a long-live object.
type LocalCache[T any] map[string]T

// Get cached value identified by the key, if absent, call the supplier func instead, and cache and return the supplied value.
func (lc LocalCache[T]) Get(key string, supplier func(string) (T, error)) (T, error) {
	if v, ok := lc[key]; ok {
		return v, nil
	}
	v, err := supplier(key)
	if err == nil {
		lc[key] = v
	}
	return v, err
}

// Simple local map-based cache.
//
// This should not be a long-live object.
type LocalCacheV2[K comparable, T any] map[K]T

// Get cached value identified by the key, if absent, call the supplier func instead, and cache and return the supplied value.
func (lc LocalCacheV2[K, T]) Get(key K, supplier func() (T, error)) (T, error) {
	if v, ok := lc[key]; ok {
		return v, nil
	}
	v, err := supplier()
	if err == nil {
		lc[key] = v
	}
	return v, err
}

func (lc LocalCacheV2[K, T]) Set(key K, t T) {
	lc[key] = t
}

// Create new LocalCache with key of type string and value of type T.
//
// Migrate to [NewLocalCacheV2] if possible.
func NewLocalCache[T any]() LocalCache[T] {
	return map[string]T{}
}

// Create new LocalCache with key of type K and value of type T.
func NewLocalCacheV2[K comparable, T any]() LocalCacheV2[K, T] {
	return map[K]T{}
}

type evictedItem[T any] struct {
	Key    string
	Bucket TBucket[T]
}

type cacheEvictStrategy[T any] interface {
	Evict(tc *ttlCache[T]) []evictedItem[T]
	OnItemAdded(key string)
	OnItemRemoved(key string)
}

type TBucket[T any] struct {
	ctime time.Time
	val   T
}

func (t *TBucket[T]) alive(now time.Time, ttl time.Duration) bool {
	return now.Sub(t.ctime) < ttl
}

func NewTBucket[T any](val T) TBucket[T] {
	return TBucket[T]{val: val, ctime: time.Now()}
}

// Time-based Cache.
type TTLCache[T any] interface {
	TryGet(key string) (T, bool)
	Get(key string, elseGet func() (T, bool)) (T, bool)
	Put(key string, t T)
	Del(key string)
	Size() int
	Exists(key string) bool
	PutIfAbsent(key string, t T) bool

	// Register callback to be invoked when entry is evicted.
	//
	// Callback may be invoked with locks obtained, callback should not block currnet goroutine.
	// Callback should not call any method on TTLCache, deadlock is almost guaranteed.
	OnEvicted(f func(key string, t T))
	Keys() []string
}

type ttlCache[T any] struct {
	cache         map[string]TBucket[T]
	mu            sync.RWMutex
	ttl           time.Duration
	maxSize       int
	evictStrategy cacheEvictStrategy[T]
	onEvictCbk    func(key string, t T)
}

func (tc *ttlCache[T]) OnEvicted(f func(key string, t T)) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.onEvictCbk = f
}

func (tc *ttlCache[T]) Size() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return len(tc.cache)
}

func (tc *ttlCache[T]) Del(key string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if _, ok := tc.cache[key]; !ok {
		return
	}
	delete(tc.cache, key)
	tc.evictStrategy.OnItemRemoved(key)
}

func (tc *ttlCache[T]) TryGet(key string) (T, bool) {
	return tc.Get(key, nil)
}

func (tc *ttlCache[T]) addBuck(prev bool, key string, v T) {
	// replace original bucket, still using the same key
	if prev {
		tc.cache[key] = NewTBucket(v)
		return
	}

	// new bucket, max size not exceeded yet
	if tc.maxSize < 1 || len(tc.cache) < tc.maxSize {
		tc.cache[key] = NewTBucket(v)
		tc.evictStrategy.OnItemAdded(key)
		return
	}

	// max size exceeded, evict some items
	evicted := tc.evictStrategy.Evict(tc)
	tc.cache[key] = NewTBucket(v)
	tc.evictStrategy.OnItemAdded(key)
	if tc.onEvictCbk != nil {
		for i := range evicted {
			ei := evicted[i]
			defer tc.onEvictCbk(ei.Key, ei.Bucket.val)
		}
	}
}

func (tc *ttlCache[T]) Get(key string, elseGet func() (T, bool)) (T, bool) {
	now := time.Now()
	var v T

	// obtain read lock, check if the key exists and is alive
	// only obtain write lock when we need to store the value
	tc.mu.RLock()
	buk, ok := tc.cache[key]
	if ok && buk.alive(now, tc.ttl) {
		defer tc.mu.RUnlock()
		return buk.val, true
	}
	tc.mu.RUnlock()

	// obtain write lock
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// check again, race condition is possible
	buk, ok = tc.cache[key]
	if ok && buk.alive(now, tc.ttl) {
		return buk.val, true
	}

	prev := ok

	// if exist, the bucket must be expired, invokes onEvict callback
	if prev && tc.onEvictCbk != nil {
		evicted := buk.val
		defer tc.onEvictCbk(key, evicted)
	}

	if elseGet != nil {
		v, ok = elseGet()
		if ok {
			tc.addBuck(prev, key, v)
			return v, true
		}
	}

	// elseGet() doesn't get the value, the evictable bucket is still there
	if prev {
		delete(tc.cache, key)
	}

	return v, false
}

func (tc *ttlCache[T]) Put(key string, t T) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	prev := false
	if _, ok := tc.cache[key]; ok {
		prev = true
	}
	tc.addBuck(prev, key, t)
}

func (tc *ttlCache[T]) Exists(key string) bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	buk, ok := tc.cache[key]
	if ok && buk.alive(time.Now(), tc.ttl) {
		return true
	}
	return false
}

func (tc *ttlCache[T]) PutIfAbsent(key string, t T) bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	buk, ok := tc.cache[key]
	if ok && buk.alive(time.Now(), tc.ttl) {
		return false
	}

	tc.addBuck(false, key, t)
	return true
}

func (tc *ttlCache[T]) Keys() []string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	keys := make([]string, 0, len(tc.cache))
	now := time.Now()
	for k, v := range tc.cache {
		if v.alive(now, tc.ttl) {
			keys = append(keys, k)
		}
	}
	return keys
}

// Create new TTLCache.
//
// Each k/v is associated with a timestamp. Each time a key lookup occurs, it checks whether the k/v is still valid by comparing the timestamp with time.Now().
//
// If the k/v is no longer 'alive', or the cache doesn't have the key, supplier func for the value is called, and the returned value is then cached.
// I.e., each k/v is evicted only at key lookup, there is no secret go-routine running to do the clean-up, the overhead for maintaining the cache is relatively small.
//
// For the max size, TTLCache will try it's best to maintain it, but it's quite possible that all values in the cache are 'alive'. Whenever the max size is violated,
// the cache will simply drop the 'least recently put' item.
//
// The returned TTLCache can be used concurrently.
func NewTTLCache[T any](ttl time.Duration, maxSize int) TTLCache[T] {
	if maxSize < 0 {
		maxSize = 0
	}
	return &ttlCache[T]{
		cache:   map[string]TBucket[T]{},
		ttl:     ttl,
		maxSize: maxSize,
		evictStrategy: &lruCacheEvictStrategy[T]{
			linkedItems: list.New(),
		},
	}
}

// Evict strategy that evicts cache item based on 'least recently put'.
//
// Always evict exactly one item with O(1) time complexity, but need to store all keys in separate linked list data structure.
type lruCacheEvictStrategy[T any] struct {
	linkedItems *list.List
}

func (p *lruCacheEvictStrategy[T]) Evict(tc *ttlCache[T]) []evictedItem[T] {
	pop := p.linkedItems.Back()
	if pop == nil || pop.Value == nil {
		return nil
	}
	k := p.linkedItems.Remove(pop).(string)
	v := tc.cache[k]
	delete(tc.cache, k)
	return []evictedItem[T]{{
		Key:    k,
		Bucket: v,
	}}
}

func (p *lruCacheEvictStrategy[T]) OnItemAdded(key string) {
	p.linkedItems.PushFront(key)
}

func (p *lruCacheEvictStrategy[T]) OnItemRemoved(key string) {
	for v := p.linkedItems.Front(); v != nil; v = v.Next() {
		if v.Value.(string) == key {
			p.linkedItems.Remove(v)
			return
		}
	}
}
