package core

type LocalCache[T any] map[string]T

func (lc LocalCache[T]) Get(key string, supplier func(string) (T, error)) (T, error) {
	if v, ok := lc[key]; ok {
		return v, nil
	}
	return supplier(key)
}
