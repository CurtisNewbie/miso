package pair

import "iter"

// Pair data structure
type Pair[T any, V any] struct {
	Left  T
	Right V
}

func New[T any, V any](t T, v V) Pair[T, V] {
	return Pair[T, V]{Left: t, Right: v}
}

// Merge StrPair into a map
func MergeStrPairs[T any](p ...Pair[string, T]) map[string][]T {
	merged := map[string][]T{}
	for _, v := range p {
		if s, ok := merged[v.Left]; ok {
			merged[v.Left] = append(s, v.Right)
		} else {
			merged[v.Left] = []T{v.Right}
		}
	}
	return merged
}

// Create iterator from Pairs.
func All[T, V any](pairs []Pair[T, V]) iter.Seq2[T, V] {
	return func(yield func(T, V) bool) {
		for _, p := range pairs {
			if !yield(p.Left, p.Right) {
				return
			}
		}
	}
}
