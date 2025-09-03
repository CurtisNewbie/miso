package redis

import (
	"fmt"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
)

// Redis Cache implementation.
//
// RCache internal isn't backed by an actual redis HSet. Cache name is simply the prefix for each key,
// and each key is stored independently.
//
//	Use [NewRCacheV2] to instantiate.
type rcacheV2[K any, T any] struct {
	c     *RCache[T]
	toKey func(k K) string
}

func (r *rcacheV2[K, T]) key(k K) string {
	return r.toKey(k)
}

func (r *rcacheV2[K, T]) Put(rail miso.Rail, k K, t T) error {
	return r.c.Put(rail, r.key(k), t)
}

func (r *rcacheV2[K, T]) RefreshTTL(rail miso.Rail, k K) error {
	return r.c.RefreshTTL(rail, r.key(k))
}

func (r *rcacheV2[K, T]) Del(rail miso.Rail, k K) error {
	return r.c.Del(rail, r.key(k))
}

func (r *rcacheV2[K, T]) GetVal(rail miso.Rail, k K) (T, error) {
	return r.GetValElse(rail, k, nil)
}

func (r *rcacheV2[K, T]) GetValElse(rail miso.Rail, k K, supplier func() (T, error)) (T, error) {
	return r.c.GetValElse(rail, r.key(k), supplier)
}

func (r *rcacheV2[K, T]) Get(rail miso.Rail, k K) (T, bool, error) {
	return r.c.Get(rail, r.key(k))
}

func (r *rcacheV2[K, T]) GetElse(rail miso.Rail, k K, supplier func() (util.Opt[T], error)) (T, bool, error) {
	return r.c.GetElse(rail, r.key(k), supplier)
}

func (r *rcacheV2[K, T]) Exists(rail miso.Rail, k K) (bool, error) {
	return r.c.Exists(rail, r.key(k))
}

func (r *rcacheV2[K, T]) DelAll(rail miso.Rail) error {
	return r.c.DelAll(rail)
}

// Create new RCache.
//
// K type must either be string or implements [fmt.Stringer], if not, it panics.
func NewRCacheV2[K any, T any](name string, conf RCacheConfig) *rcacheV2[K, T] {
	var k K
	var toKey func(K) string = nil

	if _, ok := any(k).(string); ok {
		toKey = func(k K) string {
			return any(k).(string)
		}
	}

	if toKey == nil {
		if _, ok := any(k).(fmt.Stringer); ok {
			toKey = func(k K) string {
				return any(k).(fmt.Stringer).String()
			}
		}
	}

	if toKey == nil {
		panic(miso.NewErrf("K type must either be string or fmt.Stringer"))
	}

	c := NewRCache[T](name, conf)
	return &rcacheV2[K, T]{
		c:     &c,
		toKey: toKey,
	}
}
