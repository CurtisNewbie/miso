package slutil

import "sync"

type SyncSlice[T any] struct {
	sl *[]T
	mu *sync.RWMutex
}

func (s *SyncSlice[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(*s.sl)
}

func (s *SyncSlice[T]) Append(t ...T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	*s.sl = append(*s.sl, t...)
}

func (s *SyncSlice[T]) ForEachErr(f func(t T) (stop bool, err error)) error {
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

func (s *SyncSlice[T]) ForEach(f func(t T) (stop bool)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range *s.sl {
		if f(v) {
			return
		}
	}
}

func (s *SyncSlice[T]) Copy() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := Copy(*s.sl)
	return cp
}

func NewSyncSlice[T any](initCap int) *SyncSlice[T] {
	sl := make([]T, 0, initCap)
	v := &SyncSlice[T]{
		sl: &sl,
		mu: &sync.RWMutex{},
	}

	return v
}
