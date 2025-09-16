package slutil

var (
	FilterEmptyStr = FilterEmptyStrFunc
)

// Update slice values.
//
// See [FilterFunc], [FilterEmptyStrFunc], [DistinctFunc].
func Update[T any](sl []T, fs ...func([]T) []T) []T {
	for _, f := range fs {
		sl = f(sl)
	}
	return sl
}

// Transform value to another type then perform optional value updates.
//
// See [FilterFunc], [MapFunc], [FilterEmptyStrFunc], [DistinctFunc].
func Transform[T any, V any](sl []T, f func([]T) []V, fs ...func([]V) []V) []V {
	v := f(sl)
	for _, f := range fs {
		v = f(v)
	}
	return v
}

// Update slice values then transform value to another type.
//
// See [FilterFunc], [MapFunc], [FilterEmptyStrFunc], [DistinctFunc].
func UpdateTransform[T any, V any](sl []T, f func([]T) []V, fs ...func([]T) []T) []V {
	for _, f := range fs {
		sl = f(sl)
	}
	v := f(sl)
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

func FilterEmptyStrFunc() func([]string) []string {
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
