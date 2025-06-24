package ptr

func StrVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func IntVal(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func FloatVal(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func BoolVal(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

func PtrVal[T any](p *T) T {
	if p == nil {
		var z T
		return z
	}
	return *p
}
