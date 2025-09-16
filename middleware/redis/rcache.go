package redis

import (
	"errors"
	"strings"
	"time"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/redis/go-redis/v9"
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
		return errs.WrapErrf(err, "failed to serialze value")
	}
	op := func() error {
		return errs.WrapErr(r.getClient().Set(rail.Context(), cacheKey, val, r.exp).Err())
	}
	if r.sync {
		return RLockExec(rail, r.lockKey(key), op)
	}
	return op()
}

func (r *RCache[T]) RefreshTTL(rail miso.Rail, key string) error {
	cacheKey := r.cacheKey(key)
	op := func() error {
		return errs.WrapErr(r.getClient().Expire(rail.Context(), cacheKey, r.exp).Err())
	}
	if r.sync {
		return RLockExec(rail, r.lockKey(key), op)
	}
	return op()
}

func (r *RCache[T]) Del(rail miso.Rail, key string) error {
	cacheKey := r.cacheKey(key)
	op := func() error {
		return errs.WrapErr(r.getClient().Del(rail.Context(), cacheKey).Err())
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

func (r *RCache[T]) cacheKeyPrefix() string {
	return "rcache:" + r.name + ":"
}

func (r *RCache[T]) lockKey(key string) string {
	return "lock:" + r.cacheKey(key)
}

// Get from cache
func (r *RCache[T]) GetVal(rail miso.Rail, key string) (T, error) {
	return r.GetValElse(rail, key, nil)
}

// Get from cache else run supplier
func (r *RCache[T]) GetValElse(rail miso.Rail, key string, supplier func() (T, error)) (T, error) {
	v, _, err := r.GetElse(rail, key, func() (T, bool, error) {
		v, err := supplier()
		return v, true, err
	})
	return v, err
}

// Get from cache
func (r *RCache[T]) Get(rail miso.Rail, key string) (T, bool, error) {
	return r.GetElse(rail, key, nil)
}

// Get from cache else run supplier
func (r *RCache[T]) GetElse(rail miso.Rail, key string, supplier func() (T, bool, error)) (T, bool, error) {

	// the actual operation
	op := func() (T, error) {

		cacheKey := r.cacheKey(key)
		var t T

		cmd := r.getClient().Get(rail.Context(), cacheKey)
		if cmd.Err() == nil {
			return t, errs.WrapErr(r.ValueSerializer.Deserialize(&t, cmd.Val())) // key found
		}

		if cmd.Err() != nil && !errors.Is(cmd.Err(), redis.Nil) { // cmd failed
			return t, errs.WrapErrf(cmd.Err(), "failed to get value from redis")
		}

		// nothing to supply, give up
		if supplier == nil {
			return t, miso.NoneErr
		}

		// call supplier and cache the supplied value
		supplied, ok, err := supplier()
		if err != nil {
			return t, errs.WrapErr(err)
		}
		if !ok {
			return t, miso.NoneErr
		}

		// serialize supplied value
		v, err := r.ValueSerializer.Serialize(supplied)
		if err != nil {
			return t, errs.WrapErrf(err, "failed to serialize the supplied value")
		}

		// cache the serialized value
		scmd := r.getClient().Set(rail.Context(), cacheKey, v, r.exp)
		if scmd.Err() != nil {
			return t, errs.WrapErr(scmd.Err())
		}
		return supplied, nil
	}

	handleResult := func(t T, err error) (T, bool, error) {
		if err != nil {
			if errs.IsNoneErr(err) {
				return t, false, nil
			}
			return t, false, err
		}
		return t, true, nil
	}
	if r.sync {
		return handleResult(RLockRun(rail, r.lockKey(key), op))
	}

	return handleResult(op())
}

func (r *RCache[T]) Exists(rail miso.Rail, key string) (bool, error) {
	op := func() (bool, error) {
		cacheKey := r.cacheKey(key)
		cmd := r.getClient().Exists(rail.Context(), cacheKey)
		if cmd.Err() == nil {
			return cmd.Val() > 0, nil
		}
		if cmd.Err() != nil && !errors.Is(cmd.Err(), redis.Nil) { // cmd failed
			return false, errs.WrapErrf(cmd.Err(), "failed to get value from redis, unknown error")
		}
		return false, nil
	}

	if r.sync {
		return RLockRun(rail, r.lockKey(key), op)
	}

	return op()
}

func (r *RCache[T]) DelAll(rail miso.Rail) error {
	return r.doScanAll(rail, func(keys []string) error {
		return r.doBatchDel(rail, keys)
	})
}

func (r *RCache[T]) ScanAll(rail miso.Rail, f func(keys []string) error) error {
	prefix := r.cacheKeyPrefix()
	return r.doScanAll(rail, func(keys []string) error {
		slutil.UpdateSliceValue[string](keys, func(t string) string {
			t, _ = strings.CutPrefix(t, prefix)
			return t
		})
		return f(keys)
	})
}

func (r *RCache[T]) doScanAll(rail miso.Rail, f func(keys []string) error) error {

	pat := r.cacheKeyPattern()
	cmd := r.getClient().Scan(rail.Context(), 0, pat, rcacheScanLimit)
	if cmd.Err() != nil {
		return errs.WrapErrf(cmd.Err(), "failed to scan redis with pattern '%v'", pat)
	}

	iter := cmd.Iterator()
	const batchSize = 30
	buk := make([]string, 0, batchSize)
	for iter.Next(rail.Context()) {
		if iter.Err() != nil {
			return errs.WrapErrf(iter.Err(), "failed to iterate using scan, pattern: '%v'", pat)
		}
		key := iter.Val()
		buk = append(buk, key)
		if len(buk) == batchSize {
			err := f(buk)
			if err != nil {
				return err
			}
			buk = buk[:0]
		}
	}
	if len(buk) > 0 {
		return f(buk)
	}
	return nil
}

func (r *RCache[T]) doBatchDel(rail miso.Rail, keys []string) error {
	dcmd := r.getClient().Del(rail.Context(), keys...)
	if dcmd.Err() != nil {
		if !errors.Is(dcmd.Err(), redis.Nil) {
			return errs.WrapErrf(dcmd.Err(), "failed to del keys: %v", keys)
		}
	} else {
		if miso.IsDebugLevel() {
			rail.Debugf("Deleted %v rcache keys %v", len(keys), keys)
		}
	}
	return nil
}

// Create new RCache
//
// Use [NewRCacheV2] for complex key type.
func NewRCache[T any](name string, conf RCacheConfig) RCache[T] {
	return RCache[T]{
		getClient:       func() *redis.Client { return GetRedis() },
		exp:             conf.Exp,
		name:            name,
		sync:            !conf.NoSync,
		ValueSerializer: JsonSerializer{},
	}
}
