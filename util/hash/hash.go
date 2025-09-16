package hash

import (
	"fmt"
	"hash/maphash"
	"sync"
)

// Hash Set.
//
// It's internally backed by a Map.
//
// To create a new Set, use [NewSet].
type Set[T comparable] struct {
	// keys in Set
	keys map[T]struct{}
}

// Test whether the key is in the set
func (s *Set[T]) Has(key T) bool {
	_, ok := (s.keys)[key]
	return ok
}

// Add key to set, return true if the key wasn't present previously
func (s *Set[T]) Add(key T) bool {
	if s.Has(key) {
		return false
	}
	(s.keys)[key] = struct{}{}
	return true
}

// Delete key.
func (s *Set[T]) Del(key T) {
	delete(s.keys, key)
}

// Add keys to set
func (s *Set[T]) AddAll(keys []T) {
	if keys == nil {
		return
	}
	for _, k := range keys {
		s.Add(k)
	}
}

// Add key to set (same as Add, but used for method chaining)
func (s *Set[T]) AddThen(key T) *Set[T] {
	(s.keys)[key] = struct{}{}
	return s
}

// Check if the Set is empty
func (s *Set[T]) IsEmpty() bool {
	return s.Size() < 1
}

// Get the size of the Set
func (s *Set[T]) Size() int {
	return len(s.keys)
}

// To string
func (s Set[T]) String() string {
	var ks []T = MapKeys(s.keys)
	lks := len(ks)
	st := "{ "
	for i, k := range ks {
		st += fmt.Sprintf("%v", k)
		if i < lks-1 {
			st += ", "
		}
	}
	st += " }"
	return st
}

// Copy keys in set
func (s *Set[T]) CopyKeys() []T {
	var keys []T = make([]T, 0, len(s.keys))
	for k := range s.keys {
		keys = append(keys, k)
	}
	return keys
}

// Create new Set
func NewSet[T comparable](keys ...T) Set[T] {
	s := Set[T]{keys: map[T]struct{}{}}
	for _, k := range keys {
		s.Add(k)
	}
	return s
}

// Copy values of map
func MapValues[K comparable, V any](m map[K]V) []V {
	var values []V = []V{}
	if m == nil {
		return values
	}
	for k := range m {
		values = append(values, (m)[k])
	}
	return values
}

// Get keys from map
func MapKeys[T comparable, V any](m map[T]V) []T {
	var keys []T = []T{}
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Get first from map
func MapFirst[K comparable, V any](m map[K]V) V {
	for k := range m {
		return (m)[k]
	}
	var v V
	return v
}

// Build a map with string type key and any type of value
func StrMap[T any, V any](l []T, keyMapper func(T) string, valueMapper func(T) V) map[string]V {
	m := map[string]V{}
	if l == nil {
		return m
	}
	for i := range l {
		li := l[i]
		m[keyMapper(li)] = valueMapper(li)
	}
	return m
}

// Build a map with string type key and slice value of any type
func StrSliceMap[T any, V any](l []T, keyMapper func(T) string, valueMapper func(T) V) map[string][]V {
	m := map[string][]V{}
	if l == nil {
		return m
	}
	for i := range l {
		li := l[i]
		m[keyMapper(li)] = append(m[keyMapper(li)], valueMapper(li))
	}
	return m
}

// Map with sync.RWMutex embeded.
type RWMap[K comparable, V any] struct {
	mu      sync.RWMutex
	storage map[K]V
}

// Create new RWMap
func NewRWMap[K comparable, V any]() *RWMap[K, V] {
	return &RWMap[K, V]{
		storage: make(map[K]V),
	}
}

func (r *RWMap[K, V]) Get(k K) (V, bool) {
	return r.GetElse(k, nil)
}

func (r *RWMap[K, V]) Keys() []K {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return MapKeys(r.storage)
}

func (r *RWMap[K, V]) Put(k K, v V) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.storage[k] = v
}

func (r *RWMap[K, V]) PutIfAbsent(k K, f func() V) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.storage[k]
	if ok {
		return
	}
	r.storage[k] = f()
}

func (r *RWMap[K, V]) PutIfAbsentErr(k K, f func() (V, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.storage[k]
	if ok {
		return nil
	}
	v, err := f()
	if err != nil {
		return err
	}
	r.storage[k] = v
	return nil
}

func (r *RWMap[K, V]) Del(k K) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.storage, k)
}

func (r *RWMap[K, V]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	clear(r.storage)
}

func (r *RWMap[K, V]) GetElse(k K, elseFunc func(k K) V) (V, bool) {
	r.mu.RLock()
	if v, ok := r.storage[k]; ok {
		defer r.mu.RUnlock()
		return v, true
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok := r.storage[k]; ok {
		return v, true
	}

	if elseFunc == nil {
		var v V
		return v, false
	}

	newItem := elseFunc(k)
	r.storage[k] = newItem
	return newItem, true
}

func (r *RWMap[K, V]) GetElseErr(k K, elseFunc func(k K) (V, error)) (V, error) {
	r.mu.RLock()
	if v, ok := r.storage[k]; ok {
		defer r.mu.RUnlock()
		return v, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok := r.storage[k]; ok {
		return v, nil
	}

	if elseFunc == nil {
		var v V
		return v, nil
	}

	newItem, err := elseFunc(k)
	if err != nil {
		return newItem, err
	}
	r.storage[k] = newItem
	return newItem, nil
}

const strRWMapShards = 32

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

func MapCopy[T comparable, V any](v map[T]V) map[T]V {
	cp := make(map[T]V, len(v))
	for k, v := range v {
		cp[k] = v
	}
	return cp
}
