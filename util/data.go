package util

import (
	"container/heap"
	"container/list"
	"fmt"
	"hash/maphash"
	"math/rand"
	"reflect"
	"slices"
	"sort"
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

// String-based Pair data structure
type StrPair struct {
	Left  string
	Right any
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

/*
Set data structure

It's internally backed by a Map.

To create a new Set, use NewSet() func.
*/
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
func NewSetPtr[T comparable](keys ...T) *Set[T] {
	s := NewSet[T](keys...)
	return &s
}

// Create new Set
func NewSet[T comparable](keys ...T) Set[T] {
	s := Set[T]{Keys: map[T]Void{}}
	for _, k := range keys {
		s.Add(k)
	}
	return s
}

// Create new Set from slice
func NewSetFromSlice[T comparable](ts []T) Set[T] {
	s := Set[T]{Keys: map[T]Void{}}
	s.AddAll(ts)
	return s
}

// Select one from the slice that matches the condition.
func SliceFilterFirst[T any](items []T, f func(T) bool) (T, bool) {
	for i := range items {
		t := items[i]
		if f(t) {
			return t, true
		}
	}
	return NewVar[T](), false
}

// Select random one from the slice
func SliceGetOne[T any](items []*T) *T {
	l := len(items)
	if l < 1 {
		return nil
	}
	return items[rand.Intn(l)]
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

// Copy keys of set
func SetToSlice[T comparable](s Set[T]) []T {
	return s.CopyKeys()
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

// Filter duplicate values
func Distinct(l []string) []string {
	s := NewSet[string]()
	for _, v := range l {
		s.Add(v)
	}
	return SetToSlice(s)
}

// Filter duplicate values, faster but values are sorted, and the slice values are filtered in place.
func FastDistinct(l []string) []string {
	sort.Strings(l)
	j := 0
	for i := 1; i < len(l); i++ {
		if l[j] == l[i] {
			continue
		}
		j++
		l[j] = l[i]
	}
	return l[:j+1]
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

// Filter slice values in place.
//
// Be cautious that both slices are backed by the same array.
func Filter[T any](l []T, f func(T) bool) []T {
	cp := l[:0]
	for i := range l {
		x := l[i]
		if f(x) {
			cp = append(cp, x)
		}
	}
	for i := len(cp); i < len(l); i++ {
		l[i] = NewVar[T]()
	}
	return cp
}

// Filter slice value.
//
// The original slice is not modified only copied.
func CopyFilter[T any](l []T, f func(T) bool) []T {
	cp := make([]T, 0, len(l))
	for i := range l {
		x := l[i]
		if f(x) {
			cp = append(cp, x)
		}
	}
	return cp
}

// Map slice item to another.
func MapTo[T any, V any](ts []T, mapFunc func(t T) V) []V {
	if len(ts) < 1 {
		return []V{}
	}

	vs := make([]V, 0, len(ts))
	for i := range ts {
		vs = append(vs, mapFunc(ts[i]))
	}
	return vs
}

// Merge slice of items to a map.
func MergeSlice[K comparable, V any](vs []V, keyFunc func(v V) K) map[K][]V {
	if len(vs) < 1 {
		return make(map[K][]V)
	}

	m := make(map[K][]V, len(vs))
	for i := range vs {
		v := vs[i]
		k := keyFunc(v)
		if prev, ok := m[k]; ok {
			m[k] = append(prev, v)
		} else {
			m[k] = []V{v}
		}
	}
	return m
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

func SliceFirst[T any](v []T) (t T, ok bool) {
	if len(v) > 0 {
		t = v[0]
		ok = true
		return
	}

	ok = false
	return
}

func SliceCopy[T any](v []T) []T {
	cp := make([]T, len(v))
	copy(cp, v)
	return cp
}

func MapCopy[T comparable, V any](v map[T]V) map[T]V {
	cp := make(map[T]V, len(v))
	for k, v := range v {
		cp[k] = v
	}
	return cp
}

func SliceRemove[T any](v []T, idx ...int) []T {
	cp := make([]T, 0, len(v)-len(idx))
	idSet := NewSet[int]()
	idSet.AddAll(idx)
	for i := 0; i < len(v); i++ {
		if idSet.Has(i) {
			continue
		}
		cp = append(cp, v[i])
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

func (r *StrRWMap[V]) Keys() []string {
	keys := make([]string, 0, 10)
	for _, st := range r.storage {
		keys = append(keys, st.Keys()...)
	}
	return keys
}

func QuoteStrSlice(sl []string) []string {
	return MapTo(sl, func(s string) string { return QuoteStr(s) })
}

func SplitSubSlices[T any](sl []T, limit int, f func(sub []T) error) error {
	j := 0
	for i := 0; i < len(sl); i += limit {
		j += limit
		if j > len(sl) {
			j = len(sl)
		}

		err := f(sl[i:j])
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateSliceValue[T any](s []T, upd func(t T) T) {
	for i, v := range s {
		s[i] = upd(v)
	}
}
