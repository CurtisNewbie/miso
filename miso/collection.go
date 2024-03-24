package miso

import (
	"fmt"
	"math/rand"
	"reflect"
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
func (s *Set[T]) String() string {
	var ks []T = MapKeys(&s.Keys)
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

// Create new Set
func NewSet[T comparable]() Set[T] {
	return Set[T]{Keys: map[T]Void{}}
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
func MapValues[K comparable, V any](m *map[K]V) []V {
	var values []V = []V{}
	for k := range *m {
		values = append(values, (*m)[k])
	}
	return values
}

// Copy keys of set
func SetToSlice[T comparable](s Set[T]) []T {
	return s.CopyKeys()
}

// Get keys from map
func MapKeys[T comparable, V any](m *map[T]V) []T {
	var keys []T = []T{}
	for k := range *m {
		keys = append(keys, k)
	}
	return keys
}

// Get first from map
func MapFirst[K comparable, V any](m *map[K]*V) *V {
	for k := range *m {
		return (*m)[k]
	}
	return nil
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

// Map with sync.RWMutex embeded.
type RWMap[V any] struct {
	sync.RWMutex
	storage map[string]V
	new     func(string) V
}

// Create new RWMap
func NewRWMap[V any](newFunc func(string) V) *RWMap[V] {
	return &RWMap[V]{
		storage: make(map[string]V),
		new:     newFunc,
	}
}

// Get V using k, if V exists return, else create a new one and store it.
func (r *RWMap[V]) Get(k string) V {
	r.RLock()
	if v, ok := r.storage[k]; ok {
		defer r.RUnlock()
		return v
	}
	r.RUnlock()

	r.Lock()
	defer r.Unlock()

	if v, ok := r.storage[k]; ok {
		return v
	}

	newItem := r.new(k)
	r.storage[k] = newItem
	return newItem
}

// Filter slice values in place.
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

// Filter slice value, the original slice is copied before filtering.
func CopyFilter[T any](l []T, f func(T) bool) []T {
	initLen := len(l)
	if initLen > 2 {
		initLen = initLen / 2
	}

	cp := make([]T, 0, initLen)
	for i := range l {
		x := l[i]
		if f(x) {
			cp = append(cp, x)
		}
	}
	return cp
}
