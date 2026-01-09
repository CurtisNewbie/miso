package hash

import (
	"database/sql/driver"
	"encoding/json"
	"iter"
	"sync"
)

// Hash Set.
//
// It's internally backed by a Map.
//
// To create a new Set, use [NewSet].
type SyncSet[T comparable] struct {
	set *Set[T]
	mu  *sync.RWMutex
}

// Test whether the key is in the set
func (s *SyncSet[T]) Has(key T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.set.Has(key)
}

// Add key to set, return true if the key wasn't present previously
func (s *SyncSet[T]) Add(key T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Add(key)
}

// Delete key.
func (s *SyncSet[T]) Del(key T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.set.Del(key)
}

// Add keys to set
func (s *SyncSet[T]) AddAll(keys []T) {
	if keys == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.set.AddAll(keys)
}

// Get the size of the Set
func (s *SyncSet[T]) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.set.Size()
}

// To string
func (s *SyncSet[T]) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.set.String()
}

// To string
func (s *SyncSet[T]) GoString() string {
	return s.String()
}

// Copy keys in set
func (s *SyncSet[T]) CopyKeys() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.set.CopyKeys()
}

// Implements encoding/json Marshaler
func (s *SyncSet[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.CopyKeys())
}

// Implements encoding/json Unmarshaler.
func (s *SyncSet[T]) UnmarshalJSON(b []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.set.UnmarshalJSON(b)
}

// Implements driver.Valuer in database/sql.
func (s *SyncSet[T]) Value() (driver.Value, error) {
	v, err := s.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return string(v), nil
}

// Implements sql.Scanner in database/sql.
func (s *SyncSet[T]) Scan(value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Scan(value)
}

func (s *SyncSet[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.set.Clear()
}

func (s *SyncSet[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, k := range s.CopyKeys() {
			if !yield(k) {
				return
			}
		}
	}
}

// Create new SyncSet
func NewSyncSet[T comparable](keys ...T) *SyncSet[T] {
	set := NewSet(keys...)
	return &SyncSet[T]{
		set: &set,
		mu:  &sync.RWMutex{},
	}
}
