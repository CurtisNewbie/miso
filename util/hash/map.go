package hash

// Copy values of map
func MapValues[K comparable, V any](m map[K]V) []V {
	var values []V = []V{}
	if m == nil {
		return values
	}
	for k := range m {
		values = append(values, (m)[k])
	}
	return values
}

// Get keys from map
func MapKeys[T comparable, V any](m map[T]V) []T {
	var keys []T = []T{}
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Get first from map
func MapFirst[K comparable, V any](m map[K]V) V {
	for k := range m {
		return (m)[k]
	}
	var v V
	return v
}

// Build a map with string type key and any type of value
func StrMap[T any, V any](l []T, keyMapper func(T) string, valueMapper func(T) V) map[string]V {
	m := map[string]V{}
	if l == nil {
		return m
	}
	for i := range l {
		li := l[i]
		m[keyMapper(li)] = valueMapper(li)
	}
	return m
}

// Build a map with string type key and slice value of any type
func StrSliceMap[T any, V any](l []T, keyMapper func(T) string, valueMapper func(T) V) map[string][]V {
	m := map[string][]V{}
	if l == nil {
		return m
	}
	for i := range l {
		li := l[i]
		m[keyMapper(li)] = append(m[keyMapper(li)], valueMapper(li))
	}
	return m
}

func MapCopy[T comparable, V any](v map[T]V) map[T]V {
	cp := make(map[T]V, len(v))
	for k, v := range v {
		cp[k] = v
	}
	return cp
}
