package redis

import (
	"sync"
	"time"

	"github.com/bsm/redislock"
	"github.com/go-redis/redis"
)

// RCache
type RCache struct {
	rclient *redis.Client
	rlocker *redislock.Client
	exp     time.Duration
}

// Lazy version of RCache
type LazyRCache struct {
	rcacheSupplier func() RCache
	_rcache        *RCache
	rwmu           sync.RWMutex
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
func (r *LazyRCache) Put(key string, val string) error {
	return r.rcache().Put(key, val)
}

// Get from cache
func (r *LazyRCache) Get(key string) (val string, e error) {
	return r.rcache().Get(key)
}

// Get from cache else run supplier
func (r *LazyRCache) GetElse(key string, supplier func() string) (val string, e error) {
	return r.rcache().GetElse(key, supplier)
}

// Put value to cache
func (r *RCache) Put(key string, val string) error {
	_, e := RLockRun("rcache:"+key, func() (any, error) {
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

// Get from cache
func (r *RCache) Get(key string) (val string, e error) {
	return r.GetElse(key, nil)
}

// Get from cache else run supplier
func (r *RCache) GetElse(key string, supplier func() string) (val string, e error) {
	res, e := RLockRun("rcache:"+key, func() (any, error) {
		cmd := r.rclient.Get(key)

		// key not found
		if cmd == nil {
			if supplier == nil {
				return "", nil
			}

			s := supplier()
			scmd := r.rclient.Set(key, s, r.exp)
			if scmd.Err() != nil {
				return "", scmd.Err()
			}
			return s, nil
		}

		// cmd failed
		if cmd.Err() != nil {
			return "", cmd.Err()
		}

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
	return RCache{rclient: GetRedis(), rlocker: ObtainRLocker(), exp: exp}
}

// Create new lazy RCache
func NewLazyRCache(exp time.Duration) LazyRCache {
	return LazyRCache{rcacheSupplier: func() RCache { return NewRCache(exp) }}
}
