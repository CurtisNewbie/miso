package miso

import "strings"

const (
	BOOL_STR_TRUE  = "true"
	BOOL_STR_FALSE = "false"
)

// Optional value, useful for passing zero value struct
type Optional[T any] struct {
	Val       T
	IsPresent bool
}

// Empty Optional
func EmptyOpt[T any]() Optional[T] {
	return Optional[T]{IsPresent: false}
}

// Optional with value present
func OptWith[T any](t T) Optional[T] {
	return Optional[T]{Val: t, IsPresent: true}
}

func IsTrue(boolStr string) bool {
	return strings.ToLower(boolStr) == "true"
}

func IsBool(boolStr string) bool {
	ls := strings.ToLower(boolStr)
	return ls == "true" || ls == "false"
}
