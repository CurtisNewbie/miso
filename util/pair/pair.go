package pair

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
