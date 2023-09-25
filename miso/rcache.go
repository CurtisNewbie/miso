package miso

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

type GetRCacheValue func(rail Rail, key string) (string, error)

// RCache, cache that is backed by Redis.
//
//	Use NewRCache(...) to instantiate.
type RCache struct {
	rclient  *redis.Client
	exp      time.Duration
	supplier GetRCacheValue
	name     string
}

// Put value to cache
func (r *RCache) Put(rail Rail, key string, val string) error {
	cacheKey := r.cacheKey(key)
	return RLockExec(rail, r.lockKey(key),
		func() error {
			err := r.rclient.Set(cacheKey, val, r.exp).Err()

			// value not found
			if err != nil && errors.Is(err, redis.Nil) {
				return NoneErr
			}

			return err
		},
	)
}

// Delete value from cache
func (r *RCache) Del(rail Rail, key string) error {
	cacheKey := r.cacheKey(key)
	return RLockExec(rail, r.lockKey(key),
		func() error {
			return r.rclient.Del(cacheKey).Err()
		},
	)
}

func (r *RCache) cacheKey(key string) string {
	return "rcache:" + r.name + ":" + key
}

func (r *RCache) lockKey(key string) string {
	return "lock:" + r.cacheKey(key)
}

// Get from cache else run supplier
//
// Return miso.NoneErr if none is found
func (r *RCache) Get(rail Rail, key string) (string, error) {

	// try not to always lock the whole operation, we only lock the write part
	cacheKey := r.cacheKey(key)
	cmd := r.rclient.Get(cacheKey)

	var err = cmd.Err()
	if err == nil {
		return cmd.Val(), nil
	}

	// command failed
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", err
	}

	if r.supplier == nil {
		return "", NoneErr
	}

	// attempts to load the missing value for the key
	return RLockRun(rail, r.lockKey(key), func() (string, error) {

		cmd := r.rclient.Get(cacheKey)
		if cmd.Err() == nil {
			return cmd.Val(), nil
		}

		// cmd failed
		if cmd.Err() != nil && !errors.Is(cmd.Err(), redis.Nil) {
			return "", cmd.Err()
		}

		// call supplier and cache the supplied value
		supplied, err := r.supplier(rail, key)
		if err != nil {
			return "", err
		}

		scmd := r.rclient.Set(cacheKey, supplied, r.exp)
		if scmd.Err() != nil {
			return "", scmd.Err()
		}
		return supplied, nil
	})
}

// Create new RCache
func NewRCache(name string, exp time.Duration, supplier GetRCacheValue) RCache {
	return RCache{rclient: GetRedis(), exp: exp, supplier: supplier, name: name}
}

// Lazy version of RCache, only initialized internally for the first method call.
//
//	Use NewLazyRCache(...) to instantiate.
type LazyRCache struct {
	_rcacheSupplier func() RCache
	_rcache         *RCache
	_initRCacheOnce sync.Once
}

// Obtain the wrapped *RCache object
func (r *LazyRCache) rcache() *RCache {
	r._initRCacheOnce.Do(func() {
		c := r._rcacheSupplier()
		r._rcache = &c
	})
	return r._rcache
}

// Put value to cache
func (r *LazyRCache) Put(rail Rail, key string, val string) error {
	return r.rcache().Put(rail, key, val)
}

// Get value from cache
func (r *LazyRCache) Get(rail Rail, key string) (val string, e error) {
	return r.rcache().Get(rail, key)
}

// Delete value from cache
func (r *LazyRCache) Del(rail Rail, key string) error {
	return r.rcache().Del(rail, key)
}

// Create new lazy RCache.
func NewLazyRCache(name string, exp time.Duration, supplier GetRCacheValue) LazyRCache {
	return LazyRCache{
		_rcacheSupplier: func() RCache { return NewRCache(name, exp, supplier) },
	}
}

// Lazy object RCache.
//
//	Use NewLazyORCache(...) to instantiate.
type LazyORCache[T any] struct {
	lazyRCache *LazyRCache
}

// convert string to T.
func fromCachedStr[T any](v string) (T, error) {
	var t T
	err := json.Unmarshal([]byte(v), &t)
	if err != nil {
		return t, fmt.Errorf("unable to unmarshal from string, %v", err)
	}
	return t, err
}

// convert from T to string.
func toCachedStr(t any) (string, error) {
	b, err := json.Marshal(&t)
	if err != nil {
		return "", fmt.Errorf("unable to marshal value to string, %v", err)
	}
	return string(b), nil
}

// Delete value from cache
func (r *LazyORCache[T]) Del(rail Rail, key string) error {
	return r.lazyRCache.Del(rail, key)
}

// Get from cache else run the supplier provided.
//
// Return T or error, returns miso.NoneErr if not found.
func (r *LazyORCache[T]) Get(rail Rail, key string) (T, error) {
	strVal, err := r.lazyRCache.Get(rail, key)

	var t T
	if err != nil {
		return t, err
	}
	return fromCachedStr[T](strVal)
}

type GetORCacheValue[T any] func(rail Rail, key string) (T, error)

// Create new lazy object RCache.
func NewLazyORCache[T any](name string, exp time.Duration, supplier GetORCacheValue[T]) LazyORCache[T] {
	var wrappedSupplier GetRCacheValue = nil

	if supplier != nil {
		wrappedSupplier = func(rail Rail, key string) (string, error) {
			t, err := supplier(rail, key)
			if err != nil {
				return "", err
			}
			return toCachedStr(t)
		}
	}

	lazyRCache := NewLazyRCache(name, exp, wrappedSupplier)
	return LazyORCache[T]{lazyRCache: &lazyRCache}
}
