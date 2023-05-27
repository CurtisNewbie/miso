package common

// Assert true else panic
//
// Deprecated, we should never panic
func AssertTrue(condition bool, errMsg string) {
	if !condition {
		panic(errMsg)
	}
}

// Assert non nil, if nil then panic else return the pointer
//
// Deprecated, we should never panic
func NonNil[T any](t *T, errMsg string) *T {
	if t == nil {
		panic(errMsg)
	}
	return t
}
