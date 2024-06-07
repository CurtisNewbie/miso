package util

import "strings"

const (
	BoolStrTrue  = "true"
	BoolStrFalse = "false"
)

// Optional value, useful for passing zero value struct
type Opt[T any] struct {
	Val       T
	IsPresent bool
}

// Empty Optional
func EmptyOpt[T any]() Opt[T] {
	return Opt[T]{IsPresent: false}
}

// Optional with value present
func OptWith[T any](t T) Opt[T] {
	return Opt[T]{Val: t, IsPresent: true}
}

func (o *Opt[T]) Get() (T, bool) {
	return o.Val, o.IsPresent
}

func (o *Opt[T]) IfPresent(call func(t T)) {
	if o.IsPresent {
		call(o.Val)
	}
}

func IsTrue(boolStr string) bool {
	return strings.ToLower(boolStr) == "true"
}

func IsBool(boolStr string) bool {
	ls := strings.ToLower(boolStr)
	return ls == "true" || ls == "false"
}
