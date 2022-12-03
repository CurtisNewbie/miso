package common

import (
	"math/rand"
)

// Empty Struct
type Void struct{}

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

// Check if the Set is empty
func (s *Set[T]) IsEmpty() bool {
	return s.Size() < 1
}

// Get the size of the Set
func (s *Set[T]) Size() int {
	return len(s.Keys)
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

// Get values from map
func ValuesOfStMap[T any](m map[string]*T) []*T {
	var values []*T = []*T{}
	for k := range m {
		values = append(values, m[k])
	}
	return values
}

// Get values from map
func ValuesOfMap[T any](m map[any]*T) []*T {
	var values []*T = []*T{}
	for k := range m {
		values = append(values, m[k])
	}
	return values
}

// Get keys from set
func KeysOfSet[T comparable](s Set[T]) []T {
	var keys []T = []T{}
	for k := range s.Keys {
		keys = append(keys, k)
	}
	return keys
}

// Get keys from map
func KeysOfMap[T comparable](m map[T]any) []T {
	var keys []T = []T{}
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Get first from map
func GetFirstInMap[T any](m map[any]*T) *T {
	for k := range m {
		return m[k]
	}
	return nil
}
