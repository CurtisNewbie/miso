package util

import (
	"container/heap"
	"container/list"
	"fmt"
	"hash/maphash"
	"reflect"
	"slices"
	"sync"
)

var (
	voidType = reflect.TypeOf(Void{})
)

// Empty Struct
type Void struct{}

func IsVoid(t reflect.Type) bool {
	return t == voidType
}

// Pair data structure
type Pair struct {
	Left  any
	Right any
}

// Generic pair data structure
type GnPair[T any, V any] struct {
	Left  T
	Right V
}

// String-based Pair data structure
type StrGnPair[T any] struct {
	Left  string
	Right T
}

// String-based Pair data structure
type StrPair struct {
	Left  string
	Right any
}

// Merge StrPair into a map
func MergeStrGnPairs[T any](p ...StrGnPair[T]) map[string][]T {
	merged := map[string][]T{}
	for _, v := range p {
		if s, ok := merged[v.Left]; ok {
			merged[v.Left] = append(s, v.Right)
		} else {
			merged[v.Left] = []T{v.Right}
		}
	}
	return merged
}

// Merge StrPair into a map
func MergeStrPairs(p ...StrPair) map[string][]any {
	merged := map[string][]any{}
	for _, v := range p {
		if s, ok := merged[v.Left]; ok {
			merged[v.Left] = append(s, v.Right)
		} else {
			merged[v.Left] = []any{v.Right}
		}
	}
	return merged
}

// Set data structure
//
// It's internally backed by a Map.
//
// To create a new Set, use #NewSet func.
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
type Set[T comparable] struct {
	// Keys in Set
	Keys map[T]Void
}

// Test whether the key is in the set
func (s *Set[T]) Has(key T) bool {
	_, ok := (s.Keys)[key]
	return ok
}

// Add key to set, return true if the key wasn't present previously
func (s *Set[T]) Add(key T) bool {
	if s.Has(key) {
		return false
	}
	(s.Keys)[key] = Void{}
	return true
}

// Delete key.
func (s *Set[T]) Del(key T) {
	delete(s.Keys, key)
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
	(s.Keys)[key] = Void{}
	return s
}

// Check if the Set is empty
func (s *Set[T]) IsEmpty() bool {
	return s.Size() < 1
}

// Get the size of the Set
func (s *Set[T]) Size() int {
	return len(s.Keys)
}

// To string
func (s Set[T]) String() string {
	var ks []T = MapKeys(s.Keys)
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
	var keys []T = make([]T, 0, len(s.Keys))
	for k := range s.Keys {
		keys = append(keys, k)
	}
	return keys
}

// Create ptr to a new Set
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
func NewSetPtr[T comparable](keys ...T) *Set[T] {
	s := NewSet[T](keys...)
	return &s
}

// Create new Set
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
func NewSet[T comparable](keys ...T) Set[T] {
	s := Set[T]{Keys: map[T]Void{}}
	for _, k := range keys {
		s.Add(k)
	}
	return s
}

// Create new Set from slice
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
func NewSetFromSlice[T comparable](ts []T) Set[T] {
	s := Set[T]{Keys: map[T]Void{}}
	s.AddAll(ts)
	return s
}

// Copy values of map
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
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

// Copy keys of set
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
func SetToSlice[T comparable](s Set[T]) []T {
	return s.CopyKeys()
}

// Get keys from map
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
func MapKeys[T comparable, V any](m map[T]V) []T {
	var keys []T = []T{}
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Get first from map
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
func MapFirst[K comparable, V any](m map[K]V) V {
	for k := range m {
		return (m)[k]
	}
	var v V
	return v
}

// Build a map with string type key and any type of value
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
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
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
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
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
type RWMap[K comparable, V any] struct {
	mu      sync.RWMutex
	storage map[K]V
}

// Create new RWMap
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
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

func NewStack[T any](cap int) *Stack[T] {
	if cap < 0 {
		cap = 0
	}
	return &Stack[T]{
		st: make([]T, 0, cap),
		p:  -1,
	}
}

type Stack[T any] struct {
	st []T
	p  int
}

func (s *Stack[T]) Push(v T) {
	s.st = append(s.st, v)
	s.p++
}

func (s *Stack[T]) Pop() (T, bool) {
	var v T
	if s.p < 0 {
		return v, false
	}
	v = s.st[s.p]

	s.st = slices.Delete(s.st, s.p, s.p+1)
	s.p--
	return v, true
}

func (s *Stack[T]) Peek() (T, bool) {
	var v T
	if s.p < 0 {
		return v, false
	}
	return s.st[s.p], true
}

func (s *Stack[T]) Empty() bool {
	return s.p < 0
}

func (s *Stack[T]) Len() int {
	return s.p
}

func (s Stack[T]) String() string {
	return fmt.Sprintf("%+v", s.st)
}

// Iterate Stack from top to bottom, iter func can return false to break the loop.
func (s *Stack[T]) ForEach(f func(v T) bool) {
	if s.p < 0 {
		return
	}
	for i := s.p; i >= 0; i-- {
		v := s.st[i]
		if !f(v) {
			return
		}
	}
}

func (s *Stack[T]) Slice() []T {
	return slices.Clone(s.st)
}

// Deprecated: Since v0.2.17, migrate to hash pkg.
func MapCopy[T comparable, V any](v map[T]V) map[T]V {
	cp := make(map[T]V, len(v))
	for k, v := range v {
		cp[k] = v
	}
	return cp
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{
		l: &list.List{},
	}
}

type Queue[T any] struct {
	l *list.List
}

func (q *Queue[T]) PopFront() T {
	f := q.l.Front()
	vf := f.Value.(T)
	q.l.Remove(f)
	return vf
}

func (q *Queue[T]) PopBack() T {
	f := q.l.Back()
	vf := f.Value.(T)
	q.l.Remove(f)
	return vf
}

func (q *Queue[T]) PushFront(t T) {
	q.l.PushFront(t)
}

func (q *Queue[T]) PushBack(t T) {
	q.l.PushBack(t)
}

func (q *Queue[T]) Len() int {
	return q.l.Len()
}

var _ heap.Interface = &sliceHeap[any]{}

type sliceHeap[T any] struct {
	Slice    *[]T
	LessFunc func(iv T, jv T) bool
}

func (l *sliceHeap[T]) Len() int {
	return len(*l.Slice)
}

func (l *sliceHeap[T]) Less(i, j int) bool {
	return l.LessFunc((*l.Slice)[i], (*l.Slice)[j])
}

func (l *sliceHeap[T]) Swap(i, j int) {
	(*l.Slice)[i], (*l.Slice)[j] = (*l.Slice)[j], (*l.Slice)[i]
}

func (l *sliceHeap[T]) Push(x any) {
	*l.Slice = append(*l.Slice, x.(T))
}

func (l *sliceHeap[T]) Pop() any {
	prevSlice := *l.Slice
	prevLen := len(prevSlice)
	x := prevSlice[prevLen-1]
	*l.Slice = prevSlice[0 : prevLen-1]
	return x
}

func (l *sliceHeap[T]) Peek() T {
	return (*l.Slice)[0]
}

type Heap[T any] struct {
	heap *sliceHeap[T]
}

func (h *Heap[T]) Len() int {
	return h.heap.Len()
}

func (h *Heap[T]) Push(t T) {
	heap.Push(h.heap, t)
}

func (h *Heap[T]) Pop() T {
	return heap.Pop(h.heap).(T)
}

func (h *Heap[T]) Peek() T {
	return h.heap.Peek()
}

func NewHeap[T any](cap int, lessFunc func(iv T, jv T) bool) *Heap[T] {
	if cap < 0 {
		cap = 0
	}
	sl := make([]T, 0, cap)
	h := &Heap[T]{
		heap: &sliceHeap[T]{
			Slice:    &sl,
			LessFunc: lessFunc,
		},
	}
	heap.Init(h.heap)
	return h
}

const strRWMapShards = 32

// Deprecated: Since v0.2.17, migrate to hash pkg.
type StrRWMap[V any] struct {
	seed    maphash.Seed
	storage []*RWMap[string, V]
}

// Create new sharded, concurrent access StrRWMap.
//
// Deprecated: Since v0.2.17, migrate to hash pkg.
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
	// Printlnf("k: %v, shard: %v", k, i)
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
