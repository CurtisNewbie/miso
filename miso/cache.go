package miso

import (
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

// Time-based Cache.
type TTLCache[T any] interface {
	Get(key string, elseGet func() (T, bool)) (T, bool)
	Put(key string, t T)
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

// Simple concurrent safe, in-memory lru cache implementation.
type ttlCache[T any] struct {
	cache       map[string]tbucket[T]
	mu          sync.RWMutex
	ttl         time.Duration
	maxSize     int
	lastCleanup time.Time
}

func (tc *ttlCache[T]) cleanup(now time.Time, partitions int) {
	tc.lastCleanup = time.Now()

	// we divide the whole cache into N paritiions, we only do partial cleanup for the first persudo partition
	clen := len(tc.cache)
	partition_size := clen
	if clen > partitions {
		partition_size = clen / partitions
	}
	i := 0

	// iterate the cache to cleanup dead buckets, the ordering of keys accessed is not deterministic
	for k := range tc.cache {
		if i > partition_size {
			return
		}
		i += 1
		if buk := tc.cache[k]; !buk.alive(now, tc.ttl) {
			delete(tc.cache, k)
		}
	}
}

func (tc *ttlCache[T]) Get(key string, elseGet func() (T, bool)) (T, bool) {
	now := time.Now()
	var v T

	// obtain read lock, check if the key exists and is alive
	// only obtain write lock when we need to load the key
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

		maxSizeExceeded := func() bool { return tc.maxSize > 0 && len(tc.cache) > tc.maxSize }
		if !maxSizeExceeded() {
			tc.cache[key] = newTBucket(v)
			return v, true
		}

		// if we have already exceeded the max size, we attempt to do some cleanup
		// if the cache is already full, and we cannot spare any extra space at all, we have to avoid doing cleanup all the time
		// 10s cleanup gap is merely a guess.
		if tc.lastCleanup.Before(now.Add(-10 * time.Second)) {
			tc.cleanup(now, 10)

			// after the cleanup, the max size may still be exceeded, we must avoid blowing up the cache
			if !maxSizeExceeded() {
				tc.cache[key] = newTBucket(v)
			}
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
// TTLCache will first do a partial cleanup, if the max size is still violated after the cleanup, the value is returned directly without going into the cache.
//
// The returned TTLCache can be used concurrently.
func NewTTLCache[T any](ttl time.Duration, maxSize int) TTLCache[T] {
	if maxSize < 0 {
		maxSize = 0
	}
	return &ttlCache[T]{
		cache:       map[string]tbucket[T]{},
		ttl:         ttl,
		maxSize:     maxSize,
		lastCleanup: time.Now(),
	}
}
