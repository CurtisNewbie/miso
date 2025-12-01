package hash

import (
	"database/sql/driver"
	"fmt"
	"iter"
	"reflect"

	"encoding/json"

	"github.com/curtisnewbie/miso/util/errs"
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

// To string
func (s Set[T]) GoString() string {
	return s.String()
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

func (s *Set[T]) ForEachErr(f func(v T) (stop bool, err error)) error {
	for k := range s.Keys {
		if st, err := f(k); st || err != nil {
			return err
		}
	}
	return nil
}

// Implements encoding/json Marshaler
func (s Set[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.CopyKeys())
}

// Implements encoding/json Unmarshaler.
func (s *Set[T]) UnmarshalJSON(b []byte) error {
	s.Keys = map[T]struct{}{}
	if len(b) < 1 || string(b) == "null" {
		return nil
	}
	var l []T
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}
	s.AddAll(l)
	return nil
}

// Implements driver.Valuer in database/sql.
func (s Set[T]) Value() (driver.Value, error) {
	v, err := s.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return string(v), nil
}

// Implements sql.Scanner in database/sql.
func (s *Set[T]) Scan(value interface{}) error {
	if value == nil {
		s.Keys = map[T]struct{}{}
		return nil
	}
	switch v := value.(type) {
	case string:
		if v == "" {
			s.Keys = map[T]struct{}{}
			return nil
		}
		return s.UnmarshalJSON([]byte(v))
	case []byte:
		return s.UnmarshalJSON(v)
	default:
		return errs.NewErrf("invalid field type '%v' for Set, unable to convert, %#v", reflect.TypeOf(value), v)
	}
}

func (s *Set[T]) Clear() {
	clear(s.Keys)
}

// Find keys that are in s but not in b.
func (s *Set[T]) NotInSet(b Set[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for k := range s.Keys {
			if !b.Has(k) {
				if !yield(k) {
					return
				}
			}
		}
	}
}

// Find keys that are in s and b.
func (s *Set[T]) InSet(b Set[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for k := range s.Keys {
			if b.Has(k) {
				if !yield(k) {
					return
				}
			}
		}
	}
}

// Create new Set
func NewSet[T comparable](keys ...T) Set[T] {
	s := Set[T]{Keys: map[T]struct{}{}}
	s.AddAll(keys)
	return s
}

// Create new Set with capacity.
func NewSetWithCap[T comparable](cap int, keys ...T) Set[T] {
	if cap < 1 {
		cap = 1
	}
	s := Set[T]{Keys: make(map[T]struct{}, cap)}
	s.AddAll(keys)
	return s
}
