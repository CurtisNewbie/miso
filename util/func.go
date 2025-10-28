package util

// Predicate based on t
//
// Deprecated since v0.3.7.
type Predicate[T any] func(t T) bool

// Convert t to v
//
// Deprecated since v0.3.7.
type Converter[T any, V any] func(t T) (V, error)

// Consume t
//
// Deprecated since v0.3.7.
type Consumer[T any] func(t T) error

// Transform t to another t
//
// Deprecated since v0.3.7.
type Transform[T any] func(t T) T

// Transform t to another t
//
// Deprecated since v0.3.7.
type TransformAsync[T any] func(t T) Future[T]

// Supplier of T
//
// Deprecated since v0.3.7.
type Supplier[T any] func() T

// Peek t
//
// Deprecated since v0.3.7.
type Peek[T any] func(t T)
