package hash

import "hash/maphash"

const (
	strRWMapShards = 32
)

type StrRWMap[V any] struct {
	seed    maphash.Seed
	storage []*RWMap[string, V]
}

// Create new sharded, concurrent access StrRWMap.
func NewStrRWMap[V any]() *StrRWMap[V] {
	st := make([]*RWMap[string, V], strRWMapShards)
	for i := range st {
		st[i] = NewRWMap[string, V]()
	}
	return &StrRWMap[V]{
		storage: st,
		seed:    maphash.MakeSeed(),
	}
}

func (r *StrRWMap[V]) shard(k string) *RWMap[string, V] {
	i := maphash.String(r.seed, k) % uint64(len(r.storage))
	return r.storage[i]
}

func (r *StrRWMap[V]) Get(k string) (V, bool) {
	return r.shard(k).GetElse(k, nil)
}

func (r *StrRWMap[V]) Put(k string, v V) {
	r.shard(k).Put(k, v)
}

func (r *StrRWMap[V]) PutIfAbsent(k string, f func() V) {
	r.shard(k).PutIfAbsent(k, f)
}

func (r *StrRWMap[V]) PutIfAbsentErr(k string, f func() (V, error)) {
	r.shard(k).PutIfAbsentErr(k, f)
}

func (r *StrRWMap[V]) Del(k string) {
	r.shard(k).Del(k)
}

func (r *StrRWMap[V]) GetElse(k string, elseFunc func(k string) V) (V, bool) {
	return r.shard(k).GetElse(k, elseFunc)
}

func (r *StrRWMap[V]) GetElseErr(k string, elseFunc func(k string) (V, error)) (V, error) {
	return r.shard(k).GetElseErr(k, elseFunc)
}

func (r *StrRWMap[V]) Keys() []string {
	keys := make([]string, 0, 10)
	for _, st := range r.storage {
		keys = append(keys, st.Keys()...)
	}
	return keys
}
