package common

import (
	"fmt"
	"math/rand"
)

// Empty Struct
type Void struct{}

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
func MergeStrPairs(p ...StrPair) map[string]any {
	merged := map[string]any{}
	for _, v := range p {
		merged[v.Left] = v.Right
	}
	return merged
}

/*
Set data structure

It's internally backed by a Map.

To create a new Set, use NewSet() func.

Methods:

	Has(key)  : bool
	Add(key)  : bool
	IsEmpty() : bool
	Size()    : int
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
	var ks []T = KeysOfMap(&s.Keys)
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
	var keys []T = []T{}
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
func RandomOne[T any](items []*T) *T {
	l := len(items)
	if l < 1 {
		return nil
	}
	return items[rand.Intn(l)]
}

// Copy values of map
func ValuesOfMap[K comparable, V any](m *map[K]V) []V {
	var values []V = []V{}
	for k := range *m {
		values = append(values, (*m)[k])
	}
	return values
}

// Copy keys of set
func KeysOfSet[T comparable](s Set[T]) []T {
	return s.CopyKeys()
}

// Get keys from map
func KeysOfMap[T comparable, V any](m *map[T]V) []T {
	var keys []T = []T{}
	for k := range *m {
		keys = append(keys, k)
	}
	return keys
}

// Get first from map
func GetFirstInMap[K comparable, V any](m *map[K]*V) *V {
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
	return KeysOfSet(s)
}
