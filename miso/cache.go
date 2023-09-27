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
	cache map[string]tbucket[T]
	mu    sync.Mutex
	ttl   time.Duration
}

func (tc *ttlCache[T]) Get(key string, elseGet func() (T, bool)) (T, bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	now := time.Now()
	var v T

	buk, ok := tc.cache[key]
	if ok && buk.alive(now, tc.ttl) {
		return buk.val, true
	}

	evictable := ok // if ok, then v must be evictable

	v, ok = elseGet()
	if ok {
		tc.cache[key] = newTBucket(v)
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
//
// I.e., each k/v is evicted only at key lookup, there is no secret go-routine running to do the clean-up, the overhead for maintaining the cache is relatively small.
//
// The returned TTLCache can be used concurrently.
func NewTTLCache[T any](ttl time.Duration) TTLCache[T] {
	return &ttlCache[T]{
		cache: map[string]tbucket[T]{},
		ttl:   ttl,
	}
}
