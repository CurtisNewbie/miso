package miso

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

// Configuration of RCache.
type RCacheConfig struct {
	Exp    time.Duration // exp of each entry
	NoSync bool          // doesn't use distributed lock to synchronize access to cache
}

// Redis Cache implementation.
//
//	Use NewRCache(...) to instantiate.
type RCache[T any] struct {
	ValueSerializer Serializer              // serializer / deserializer
	getClient       Supplier[*redis.Client] // supplier of client (using func to make it lazy)
	exp             time.Duration           // ttl for each cache entry
	name            string                  // name of the cache
	sync            bool                    // synchronize operation
}

func (r *RCache[T]) Put(rail Rail, key string, t T) error {
	cacheKey := r.cacheKey(key)
	val, err := r.ValueSerializer.Serialize(t)
	if err != nil {
		return fmt.Errorf("failed to serialze value, %w", err)
	}
	op := func() error {
		return r.getClient().Set(cacheKey, val, r.exp).Err()
	}
	if r.sync {
		return RLockExec(rail, r.lockKey(key), op)
	}
	return op()
}

func (r *RCache[T]) Del(rail Rail, key string) error {
	cacheKey := r.cacheKey(key)
	op := func() error {
		return r.getClient().Del(cacheKey).Err()
	}
	if r.sync {
		return RLockExec(rail, r.lockKey(key), op)
	}
	return op()
}

func (r *RCache[T]) cacheKey(key string) string {
	return "rcache:" + r.name + ":" + key
}

func (r *RCache[T]) lockKey(key string) string {
	return "lock:" + r.cacheKey(key)
}

// Get from cache else run supplier
//
// Return miso.NoneErr if none is found
func (r *RCache[T]) Get(rail Rail, key string, supplier func(rail Rail, key string) (T, error)) (T, error) {

	cacheKey := r.cacheKey(key)
	cmd := r.getClient().Get(cacheKey)

	var t T
	var err = cmd.Err()

	if err == nil {
		err = r.ValueSerializer.Deserialize(&t, cmd.Val())
		if err != nil {
			return t, fmt.Errorf("failed to deserialize value from cache, %v, %w", cmd.Val(), err)
		}
		return t, nil
	} else if !errors.Is(err, redis.Nil) { // command failed
		return t, fmt.Errorf("failed to get value from redis, unknown error, %w", err)
	}

	// nothing to supply, give up
	if supplier == nil {
		return t, NoneErr
	}

	// attempts to load the missing value for the key
	op := func() (T, error) {

		cmd := r.getClient().Get(cacheKey)
		if cmd.Err() == nil {
			err = r.ValueSerializer.Deserialize(&t, cmd.Val())
			return t, err
		}

		if cmd.Err() != nil && !errors.Is(cmd.Err(), redis.Nil) { // cmd failed
			return t, fmt.Errorf("failed to get value from redis, unknown error, %w", cmd.Err())
		}

		// call supplier and cache the supplied value
		supplied, err := supplier(rail, key)
		if err != nil {
			return t, err
		}

		// serialize supplied value
		v, err := r.ValueSerializer.Serialize(supplied)
		if err != nil {
			return t, fmt.Errorf("failed to serialize the supplied value, %w", err)
		}

		// cache the serialized value
		scmd := r.getClient().Set(cacheKey, v, r.exp)
		if scmd.Err() != nil {
			return t, scmd.Err()
		}
		return supplied, nil
	}

	if r.sync {
		return RLockRun(rail, r.lockKey(key), op)
	}

	return op()
}

// Create new RCache
func NewRCache[T any](name string, conf RCacheConfig) RCache[T] {
	return RCache[T]{
		getClient:       func() *redis.Client { return GetRedis() },
		exp:             conf.Exp,
		name:            name,
		sync:            !conf.NoSync,
		ValueSerializer: JsonSerializer{},
	}
}
