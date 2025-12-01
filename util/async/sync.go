package async

import "sync"

type Synced[T any, V any] struct {
	mu *sync.RWMutex
	t  T
}

func (s *Synced[T, V]) DoWrite(f func(v T) (V, error)) (V, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return f(s.t)
}

func (s *Synced[T, V]) DoRead(f func(v T) (V, error)) (V, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return f(s.t)
}

func NewSynced[T any, V any](t T) *Synced[T, V] {
	return &Synced[T, V]{
		mu: &sync.RWMutex{},
		t:  t,
	}
}
