package core

// Predicate
type Predicate[T any] func(t T) bool

// Convert t to v
type Converter[T any, V any] func(t T) (V, error)

// Consume t
type Consumer[T any] func(t T) error

// Peek t
type Peek[T any] func(t T) T
