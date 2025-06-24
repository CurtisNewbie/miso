package redis

import (
	"errors"
	"fmt"
	"time"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/go-redis/redis/v7"
)

const (
	rcacheScanLimit int64 = 100
)

// Configuration of RCache.
type RCacheConfig struct {
	//expire time for each entry
	Exp time.Duration

	// Disable use of distributed lock to synchronize access to the key in the cache.
	//
	// Most of the operations are atomic except Get(...) with supplier callback.
	// If your are loading the cache manually using Put(...), then you probably don't need synchronization at all.
	NoSync bool
}

// Redis Cache implementation.
//
// RCache internal isn't backed by an actual redis HSet. Cache name is simply the prefix for each key,
// and each key is stored independently.
//
//	Use NewRCache(...) to instantiate.
type RCache[T any] struct {
	ValueSerializer Serializer                   // serializer / deserializer
	getClient       util.Supplier[*redis.Client] // supplier of client (using func to make it lazy)
	exp             time.Duration                // ttl for each cache entry
	name            string                       // name of the cache
	sync            bool                         // synchronize operation
}

func (r *RCache[T]) Put(rail miso.Rail, key string, t T) error {
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

func (r *RCache[T]) Del(rail miso.Rail, key string) error {
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

func (r *RCache[T]) cacheKeyPattern() string {
	return "rcache:" + r.name + ":*"
}

func (r *RCache[T]) lockKey(key string) string {
	return "lock:" + r.cacheKey(key)
}

// Get from cache else run supplier
//
// Return miso.NoneErr if none is found
func (r *RCache[T]) Get(rail miso.Rail, key string, supplier func() (T, error)) (T, error) {

	// the actual operation
	op := func() (T, error) {

		cacheKey := r.cacheKey(key)
		var t T

		cmd := r.getClient().Get(cacheKey)
		if cmd.Err() == nil {
			return t, r.ValueSerializer.Deserialize(&t, cmd.Val()) // key found
		}

		if cmd.Err() != nil && !errors.Is(cmd.Err(), redis.Nil) { // cmd failed
			return t, fmt.Errorf("failed to get value from redis, unknown error, %w", cmd.Err())
		}

		// nothing to supply, give up
		if supplier == nil {
			return t, miso.NoneErr
		}

		// call supplier and cache the supplied value
		supplied, err := supplier()
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

func (r *RCache[T]) Exists(rail miso.Rail, key string) (bool, error) {
	op := func() (bool, error) {
		cacheKey := r.cacheKey(key)
		cmd := r.getClient().Exists(cacheKey)
		if cmd.Err() == nil {
			return cmd.Val() > 0, nil
		}
		if cmd.Err() != nil && !errors.Is(cmd.Err(), redis.Nil) { // cmd failed
			return false, fmt.Errorf("failed to get value from redis, unknown error, %w", cmd.Err())
		}
		return false, nil
	}

	if r.sync {
		return RLockRun(rail, r.lockKey(key), op)
	}

	return op()
}

func (r *RCache[T]) DelAll(rail miso.Rail) error {
	pat := r.cacheKeyPattern()
	cmd := r.getClient().Scan(0, pat, rcacheScanLimit)
	if cmd.Err() != nil {
		return fmt.Errorf("failed to scan redis with pattern '%v', %w", pat, cmd.Err())
	}

	iter := cmd.Iterator()
	for iter.Next() {
		if iter.Err() != nil {
			return fmt.Errorf("failed to iterate using scan, pattern: '%v', %w", pat, iter.Err())
		}
		key := iter.Val()
		dcmd := r.getClient().Del(key)
		if dcmd.Err() != nil {
			if !errors.Is(dcmd.Err(), redis.Nil) {
				return fmt.Errorf("failed to del key %v, %w", key, dcmd.Err())
			}
		} else {
			rail.Debugf("Deleted rcache key %v", key)
		}
	}
	return nil
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
