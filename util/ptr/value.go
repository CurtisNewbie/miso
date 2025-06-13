package ptr

func StrVal(p *string) string {
	return PtrVal[string](p)
}

func IntVal(p *int) int {
	return PtrVal[int](p)
}

func PtrVal[T any](p *T) T {
	if p == nil {
		var z T
		return z
	}
	return *p
}
