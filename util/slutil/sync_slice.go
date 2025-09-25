package slutil

import "sync"

type syncSlice[T any] struct {
	sl *[]T
	mu *sync.RWMutex
}

func (s *syncSlice[T]) Append(t ...T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	*s.sl = append(*s.sl, t...)
}

func (s *syncSlice[T]) ForEachErr(f func(t T) (stop bool, err error)) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range *s.sl {
		st, er := f(v)
		if er != nil {
			return er
		}
		if st {
			return nil
		}
	}
	return nil
}

func (s *syncSlice[T]) ForEach(f func(t T) (stop bool)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range *s.sl {
		if f(v) {
			return
		}
	}
}

func NewSyncSlice[T any](initCap int) *syncSlice[T] {
	sl := make([]T, 0, initCap)
	v := &syncSlice[T]{
		sl: &sl,
		mu: &sync.RWMutex{},
	}

	return v
}
