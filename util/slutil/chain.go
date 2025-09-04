package slutil

func Update[T any](sl []T, fs ...func([]T) []T) []T {
	for _, f := range fs {
		sl = f(sl)
	}
	return sl
}

func Transform[T any, V any](sl []T, f func([]T) []V, fs ...func([]V) []V) []V {
	v := f(sl)
	for _, f := range fs {
		v = f(v)
	}
	return v
}

// Filter slice value.
//
// The original slice is not modified only copied.
func FilterFunc[T any](f func(T) bool) func([]T) []T {
	return func(l []T) []T {
		return CopyFilter(l, f)
	}
}

// Map slice item to another.
func MapFunc[T any, V any](mapFunc func(t T) V) func([]T) []V {
	return func(ts []T) []V {
		if len(ts) < 1 {
			return []V{}
		}
		vs := make([]V, 0, len(ts))
		for i := range ts {
			vs = append(vs, mapFunc(ts[i]))
		}
		return vs
	}
}

func FilterEmptyStr() func([]string) []string {
	return func(s []string) []string {
		return Filter(s, func(s string) bool {
			return s != ""
		})
	}
}

// Filter duplicate values
func DistinctFunc() func([]string) []string {
	return func(l []string) []string {
		return Distinct(l)
	}
}
