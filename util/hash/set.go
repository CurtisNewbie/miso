package hash

import (
	"fmt"
)

// Hash Set.
//
// It's internally backed by a Map.
//
// To create a new Set, use [NewSet].
type Set[T comparable] struct {
	// Keys in Set
	Keys map[T]struct{}
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
	(s.Keys)[key] = struct{}{}
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
	(s.Keys)[key] = struct{}{}
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

func (s *Set[T]) ForEach(f func(v T) (stop bool)) {
	for k := range s.Keys {
		if f(k) {
			return
		}
	}
}

// Create new Set
func NewSet[T comparable](keys ...T) Set[T] {
	s := Set[T]{Keys: map[T]struct{}{}}
	for _, k := range keys {
		s.Add(k)
	}
	return s
}
