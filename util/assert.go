package util

// Assert true else panic
func AssertTrue(condition bool, errMsg string) {
	if !condition {
		panic(errMsg)
	}
}

// Assert non nil, if nil then panic else return the pointer
func NonNil[T any](t *T, errMsg string) *T {
	if t == nil {
		panic(errMsg)
	}
	return t
}
