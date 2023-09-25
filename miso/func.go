package miso

// Predicate based on t
type Predicate[T any] func(t T) bool

// Convert t to v
type Converter[T any, V any] func(t T) (V, error)

// Consume t
type Consumer[T any] func(t T) error

// Transform t to another t
type Transform[T any] func(t T) T

// Supplier of T
type Supplier[T any] func() T

// Peek t
type Peek[T any] func(t T)
