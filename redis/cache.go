package redis

import (
	"sync"
	"time"

	"github.com/curtisnewbie/gocommon/common"
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
func (r *LazyRCache) Put(ec common.ExecContext, key string, val string) error {
	return r.rcache().Put(ec, key, val)
}

// Get from cache
func (r *LazyRCache) Get(ec common.ExecContext, key string) (val string, e error) {
	return r.rcache().Get(ec, key)
}

// Get from cache else run supplier
func (r *LazyRCache) GetElse(ec common.ExecContext, key string, supplier func() string) (val string, e error) {
	return r.rcache().GetElse(ec, key, supplier)
}

// Put value to cache
func (r *RCache) Put(ec common.ExecContext, key string, val string) error {
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

// Get from cache
func (r *RCache) Get(ec common.ExecContext, key string) (val string, e error) {
	return r.GetElse(ec, key, nil)
}

// Get from cache else run supplier, if supplier provides empty str, then the value is returned directly without call SET in redis
func (r *RCache) GetElse(ec common.ExecContext, key string, supplier func() string) (val string, e error) {

	// for the query, we try not to lock the operation, we only lock the write part
	cmd := r.rclient.Get(key)
	if cmd != nil {
		if e = cmd.Err(); e != nil {
			return
		}

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
		if cmd == nil {
			supplied := supplier()
			if supplied == "" {
				return "", nil
			}

			scmd := r.rclient.Set(key, supplied, r.exp)
			if scmd.Err() != nil {
				return "", scmd.Err()
			}
			return supplied, nil
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
	return RCache{rclient: GetRedis(), exp: exp}
}

// Create new lazy RCache
func NewLazyRCache(exp time.Duration) LazyRCache {
	return LazyRCache{rcacheSupplier: func() RCache { return NewRCache(exp) }}
}
