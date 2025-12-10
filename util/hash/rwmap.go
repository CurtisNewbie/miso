package hash

import (
	"sync"
)

// Create new RWMap
func NewRWMap[K comparable, V any]() *RWMap[K, V] {
	return &RWMap[K, V]{
		storage: make(map[K]V),
	}
}

// Map with sync.RWMutex embeded.
type RWMap[K comparable, V any] struct {
	mu      sync.RWMutex
	storage map[K]V
}

func (r *RWMap[K, V]) Get(k K) (V, bool) {
	return r.GetElse(k, nil)
}

func (r *RWMap[K, V]) Keys() []K {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return MapKeys(r.storage)
}

func (r *RWMap[K, V]) Put(k K, v V) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.storage[k] = v
}

func (r *RWMap[K, V]) PutIfAbsent(k K, f func() V) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.storage[k]
	if ok {
		return
	}
	r.storage[k] = f()
}

func (r *RWMap[K, V]) PutIfAbsentErr(k K, f func() (V, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.storage[k]
	if ok {
		return nil
	}
	v, err := f()
	if err != nil {
		return err
	}
	r.storage[k] = v
	return nil
}

func (r *RWMap[K, V]) Del(k K) (prev V, hasPrev bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.storage[k]
	if ok {
		delete(r.storage, k)
	}
	return v, ok
}

func (r *RWMap[K, V]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	clear(r.storage)
}

func (r *RWMap[K, V]) GetElse(k K, elseFunc func(k K) V) (V, bool) {
	r.mu.RLock()
	if v, ok := r.storage[k]; ok {
		defer r.mu.RUnlock()
		return v, true
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok := r.storage[k]; ok {
		return v, true
	}

	if elseFunc == nil {
		var v V
		return v, false
	}

	newItem := elseFunc(k)
	r.storage[k] = newItem
	return newItem, true
}

func (r *RWMap[K, V]) GetElseErr(k K, elseFunc func(k K) (V, error)) (V, error) {
	r.mu.RLock()
	if v, ok := r.storage[k]; ok {
		defer r.mu.RUnlock()
		return v, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok := r.storage[k]; ok {
		return v, nil
	}

	if elseFunc == nil {
		var v V
		return v, nil
	}

	newItem, err := elseFunc(k)
	if err != nil {
		return newItem, err
	}
	r.storage[k] = newItem
	return newItem, nil
}
