package miso

type LocalCache[T any] map[string]T

// create new LocalCache with key of type string and value of type T.
func NewLocalCache[T any]() LocalCache[T] {
	return map[string]T{}
}

// get cached value identified by the key, if absent, call the supplier func instead, and cache and return the supplied value.
func (lc LocalCache[T]) Get(key string, supplier func(string) (T, error)) (T, error) {
	if v, ok := lc[key]; ok {
		return v, nil
	}
	v, err := supplier(key)
	if err == nil {
		lc[key] = v
	}
	return v, err
}
