package gocommon

import (
	"math/rand"
)

type Void struct{}
type Set[T comparable] map[T]Void

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
	for k := range s {
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
