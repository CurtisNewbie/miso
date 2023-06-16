package common

import (
	"errors"
	"time"
)

var (
	ErrGetTimeout = errors.New("future.TimedGet timeout")
)

type Future[T any] interface {
	Get() (T, error)
	TimedGet(timeout int) (T, error)
}

type future[T any] struct {
	ch       chan func() (T, error)
	callback func() (T, error)
}

// Get from Future indefinitively
func (f future[T]) Get() (T, error) {
	getResult := <-f.ch
	return getResult()
}

// Get from Future with timeout (in milliseconds)
func (f future[T]) TimedGet(timeout int) (T, error) {
	select {
	case obtainResult := <-f.ch:
		return obtainResult()
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		var t T
		return t, ErrGetTimeout
	}
}

// Create Future, once the future is created, it starts running on a new goroutine
func RunAsync[T any](callback func() (T, error)) Future[T] {
	fut := future[T]{callback: callback}
	fut.ch = make(chan func() (T, error))
	go func(cha chan func() (T, error)) {
		t, err := fut.callback()
		f := func() (T, error) { return t, err }
		cha <- f
	}(fut.ch)
	return fut
}
