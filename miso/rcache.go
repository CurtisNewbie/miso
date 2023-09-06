package miso

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

// RCache
type RCache struct {
	rclient *redis.Client
	exp     time.Duration
}

// Lazy version of RCache
type LazyRCache struct {
	rcacheSupplier func() RCache
	_rcache        *RCache
	rwmu           sync.RWMutex
}

// Lazy object RCache
type LazyObjRCache[T any] struct {
	lazyRCache *LazyRCache
}

// Remove value from cache
func (r *LazyObjRCache[T]) Del(ec Rail, key string) error {
	return r.lazyRCache.Del(ec, key)
}

// Get from cache else run supplier
func (r *LazyObjRCache[T]) GetElse(ec Rail, key string, supplier func() (T, bool, error)) (val T, ok bool, e error) {
	strVal, err := r.lazyRCache.GetElse(ec, key, func() (string, error) {
		supplied, ok, err := supplier()
		if err != nil {
			return "", err
		}
		if !ok {
			return "", nil
		}
		b, err := json.Marshal(&supplied)
		if err != nil {
			return "", err
		}
		return string(b), nil
	})

	var t T
	if err != nil {
		return t, false, err
	}
	if strVal == "" {
		return t, false, nil
	}
	if err := json.Unmarshal([]byte(strVal), &t); err != nil {
		return t, false, err
	}
	return t, true, nil
}

func (r *LazyRCache) rcache() *RCache {
	r.rwmu.RLock()
	if r._rcache != nil {
		defer r.rwmu.RUnlock()
		return r._rcache
	}
	r.rwmu.RUnlock()

	r.rwmu.Lock()
	defer r.rwmu.Unlock()
	c := r.rcacheSupplier()
	r._rcache = &c
	return r._rcache
}

// Put value to cache
func (r *LazyRCache) Put(ec Rail, key string, val string) error {
	return r.rcache().Put(ec, key, val)
}

// Get from cache
func (r *LazyRCache) Get(ec Rail, key string) (val string, e error) {
	return r.rcache().Get(ec, key)
}

// Get from cache else run supplier
func (r *LazyRCache) GetElse(ec Rail, key string, supplier func() (string, error)) (val string, e error) {
	return r.rcache().GetElse(ec, key, supplier)
}

// Remove value from cache
func (r *LazyRCache) Del(ec Rail, key string) error {
	return r.rcache().Del(ec, key)
}

// Put value to cache
func (r *RCache) Put(ec Rail, key string, val string) error {
	_, e := RLockRun(ec, "rcache:"+key, func() (any, error) {
		scmd := r.rclient.Set(key, val, r.exp)
		if scmd.Err() != nil {
			return nil, scmd.Err()
		}
		return nil, nil
	})

	if e != nil {
		return e
	}
	return nil
}

// Remove value from cache
func (r *RCache) Del(ec Rail, key string) error {
	_, e := RLockRun(ec, "rcache:"+key, func() (any, error) {
		scmd := r.rclient.Del(key)
		if scmd.Err() != nil {
			return nil, scmd.Err()
		}
		ec.Infof("Removed '%v' from cache", key)
		return nil, nil
	})
	return e
}

// Get from cache
func (r *RCache) Get(ec Rail, key string) (val string, e error) {
	return r.GetElse(ec, key, nil)
}

// Get from cache else run supplier, if supplier provides empty str, then the value is returned directly without call SET in redis
func (r *RCache) GetElse(ec Rail, key string, supplier func() (string, error)) (val string, e error) {

	// for the query, we try not to lock the operation, we only lock the write part
	cmd := r.rclient.Get(key)
	if cmd.Err() != nil {
		if !errors.Is(cmd.Err(), redis.Nil) { // trying to GET key that is not present is a valid case
			e = cmd.Err()
			return
		}
	} else { // no error, return the value we retrieved
		val = cmd.Val()
		return
	}

	// both the key and the supplier are missing, there is nothing we can do
	if supplier == nil {
		return
	}

	// attempts to load the missing value for the key
	res, e := RLockRun(ec, "rcache:"+key, func() (any, error) {

		cmd := r.rclient.Get(key)

		// key not found
		if cmd.Err() != nil {

			if !errors.Is(cmd.Err(), redis.Nil) {
				return "", cmd.Err() // cmd failed
			}

			// the key is still missing, tries to run the value supplier for the key
			supplied, err := supplier()
			if err != nil {
				return "", err
			}
			if supplied == "" {
				return "", nil
			}

			scmd := r.rclient.Set(key, supplied, r.exp)
			if scmd.Err() != nil {
				return "", scmd.Err()
			}
			return supplied, nil
		}

		// key is present, return the value
		return cmd.Val(), nil
	})

	if e != nil {
		return
	}

	val = res.(string)
	return
}

// Create new RCache
func NewRCache(exp time.Duration) RCache {
	return RCache{rclient: GetRedis(), exp: exp}
}

// Create new lazy RCache
func NewLazyRCache(exp time.Duration) LazyRCache {
	return LazyRCache{rcacheSupplier: func() RCache { return NewRCache(exp) }}
}

// Create new lazy, object RCache
func NewLazyObjectRCache[T any](exp time.Duration) LazyObjRCache[T] {
	lr := NewLazyRCache(exp)
	return LazyObjRCache[T]{lazyRCache: &lr}
}
