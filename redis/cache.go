package redis

import (
	"time"

	"github.com/bsm/redislock"
	"github.com/go-redis/redis"
)

type RCache struct {
	rclient *redis.Client
	rlocker *redislock.Client
	exp     time.Duration
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
