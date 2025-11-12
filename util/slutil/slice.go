package slutil

import (
	"math/rand"
	"sort"
)

// Select one from the slice that matches the condition.
func FirstMatch[T any](items []T, f func(T) bool) (T, bool) {
	for i := range items {
		t := items[i]
		if f(t) {
			return t, true
		}
	}
	var t T
	return t, false
}

// Select random one from the slice
func RandOne[T any](items []*T) *T {
	l := len(items)
	if l < 1 {
		return nil
	}
	return items[rand.Intn(l)]
}

// Filter duplicate values
func Distinct(l []string) []string {
	s := make(map[string]struct{}, len(l))
	for _, v := range l {
		s[v] = struct{}{}
	}
	var keys []string = make([]string, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	return keys
}

// Filter duplicate values, faster but values are sorted, and the slice values are filtered in place.
func FastDistinct(l []string) []string {
	sort.Strings(l)
	j := 0
	for i := 1; i < len(l); i++ {
		if l[j] == l[i] {
			continue
		}
		j++
		l[j] = l[i]
	}
	return l[:j+1]
}

// Filter slice values in place.
//
// Be cautious that both slices are backed by the same array.
func Filter[T any](l []T, f func(T) bool) []T {
	cp := l[:0]
	for i := range l {
		x := l[i]
		if f(x) {
			cp = append(cp, x)
		}
	}
	for i := len(cp); i < len(l); i++ {
		var tv T
		l[i] = tv
	}
	return cp
}

// Filter slice values in place.
//
// Be cautious that both slices are backed by the same array.
func FilterIdx[T any](l []T, f func(int, T) bool) []T {
	cp := l[:0]
	for i := range l {
		x := l[i]
		if f(i, x) {
			cp = append(cp, x)
		}
	}
	for i := len(cp); i < len(l); i++ {
		var tv T
		l[i] = tv
	}
	return cp
}

// Filter slice value.
//
// The original slice is not modified only copied.
func CopyFilter[T any](l []T, f func(T) bool) []T {
	ln := len(l)
	if ln > 10 {
		ln = ln / 2 // 50%?
	}
	cp := make([]T, 0, ln)
	for i := range l {
		x := l[i]
		if f(x) {
			cp = append(cp, x)
		}
	}
	return cp
}

// Map slice item to another.
func MapTo[T any, V any](ts []T, mapFunc func(t T) V) []V {
	if len(ts) < 1 {
		return []V{}
	}
	vs := make([]V, 0, len(ts))
	for i := range ts {
		vs = append(vs, mapFunc(ts[i]))
	}
	return vs
}

// Merge slice of items to a map.
func MergeMapSlice[K comparable, V any](vs []V, keyFunc func(v V) K) map[K][]V {
	if len(vs) < 1 {
		return make(map[K][]V)
	}

	m := make(map[K][]V, len(vs))
	for i := range vs {
		v := vs[i]
		k := keyFunc(v)
		if prev, ok := m[k]; ok {
			m[k] = append(prev, v)
		} else {
			m[k] = []V{v}
		}
	}
	return m
}

// Merge slice of items to a map.
func MergeMap[K comparable, V any](vs []V, keyFunc func(v V) K) map[K]V {
	if len(vs) < 1 {
		return make(map[K]V)
	}

	m := make(map[K]V, len(vs))
	for i := range vs {
		v := vs[i]
		k := keyFunc(v)
		m[k] = v
	}
	return m
}

// Merge slice of items to a map.
func MergeMapAs[T any, K comparable, V any](ts []T, keyFunc func(t T) K, valueFunc func(t T) V) map[K]V {
	if len(ts) < 1 {
		return make(map[K]V)
	}

	m := make(map[K]V, len(ts))
	for i := range ts {
		t := ts[i]
		k := keyFunc(t)
		m[k] = valueFunc(t)
	}
	return m
}

func Copy[T any](v []T) []T {
	cp := make([]T, len(v))
	copy(cp, v)
	return cp
}

func First[T any](v []T) (t T, ok bool) {
	if len(v) > 0 {
		t = v[0]
		ok = true
		return
	}
	ok = false
	return
}

func VarArgAny[T any](v []T, defVal func() T) (t T) {
	f, ok := First(v)
	if ok {
		return f
	}
	return defVal()
}

func Remove[T any](v []T, idx ...int) []T {
	cp := make([]T, 0, len(v)-len(idx))
	idSet := map[int]struct{}{}
	for _, v := range idx {
		idSet[v] = struct{}{}
	}
	for i := 0; i < len(v); i++ {
		if _, ok := idSet[i]; ok {
			continue
		}
		cp = append(cp, v[i])
	}
	return cp
}

func Prepend[T any](v []T, ts ...T) []T {
	if len(ts) == 0 {
		return v
	}

	total := len(v) + len(ts)
	if cap(v) >= total {
		v = v[:total]
		copy(v[len(ts):], v)
		copy(v, ts)
		return v
	}

	cp := make([]T, total)
	copy(cp, ts)
	copy(cp[len(ts):], v)
	return cp
}

func QuoteStrSlice(sl []string) []string {
	return MapTo(sl, func(s string) string { return "\"" + s + "\"" })
}

func SplitSubSlices[T any](sl []T, limit int, f func(sub []T) error) error {
	j := 0
	for i := 0; i < len(sl); i += limit {
		j += limit
		if j > len(sl) {
			j = len(sl)
		}

		err := f(sl[i:j])
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateSliceValue[T any](s []T, upd func(t T) T) {
	for i, v := range s {
		s[i] = upd(v)
	}
}

func MergeVarargs[T any](fst T, args ...T) []T {
	ar := make([]T, 0, 1+len(args))
	ar = append(ar, fst)
	return append(ar, args...)
}

func Concat[T any](a []T, b ...[]T) []T {
	var total int = len(a)
	for _, v := range b {
		total += len(v)
	}
	cp := make([]T, 0, total)
	cp = append(cp, a...)
	for _, v := range b {
		cp = append(cp, v...)
	}
	return cp
}
