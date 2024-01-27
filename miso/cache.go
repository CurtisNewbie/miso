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

// create new LocalCache with key of type string and value of type T.
func NewLocalCache[T any]() LocalCache[T] {
	return map[string]T{}
}

// get cached value identified by the key, if absent, call the supplier func instead, and cache and return the supplied value.
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

type cacheEvictStrategy[T any] interface {
	Evict(tc *ttlCache[T]) bool
	OnItemAdded(key string)
	OnItemRemoved(key string)
}

// Time-based Cache.
type TTLCache[T any] interface {
	Get(key string, elseGet func() (T, bool)) (T, bool)
	Put(key string, t T)
	Del(key string)
	Size() int
}

type tbucket[T any] struct {
	ctime time.Time
	val   T
}

func (t *tbucket[T]) alive(now time.Time, ttl time.Duration) bool {
	return now.Sub(t.ctime) < ttl
}

func newTBucket[T any](val T) tbucket[T] {
	return tbucket[T]{val: val, ctime: time.Now()}
}

type ttlCache[T any] struct {
	cache         map[string]tbucket[T]
	mu            sync.RWMutex
	ttl           time.Duration
	maxSize       int
	evictStrategy cacheEvictStrategy[T]
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
	evictable := ok // if ok, then v must be evictable

	v, ok = elseGet()
	if ok {

		// max size not exceeded yet
		if tc.maxSize < 1 || len(tc.cache) < tc.maxSize-1 {
			tc.cache[key] = newTBucket(v)
			tc.evictStrategy.OnItemAdded(key)
			return v, true
		}

		// max size exceeded, evict some items
		if tc.evictStrategy.Evict(tc) {
			tc.cache[key] = newTBucket(v)
			tc.evictStrategy.OnItemAdded(key)
		}

		return v, true
	}

	// elseGet() doesn't get the value, the evictable bucket is still there
	if evictable {
		delete(tc.cache, key)
	}

	return v, false
}

func (tc *ttlCache[T]) Put(key string, t T) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cache[key] = newTBucket(t)
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
		cache:   map[string]tbucket[T]{},
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

func (p *lruCacheEvictStrategy[T]) Evict(tc *ttlCache[T]) bool {
	pop := p.linkedItems.Back()
	if pop == nil || pop.Value == nil {
		return true
	}
	k := p.linkedItems.Remove(pop).(string)
	delete(tc.cache, k)
	return true
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
