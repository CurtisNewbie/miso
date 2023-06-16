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
	ch       chan FutureResult[T]
	callback func() FutureResult[T]
}

type FutureResult[T any] struct {
	Result T
	Err    error
}

// Get from Future
func (f future[T]) Get() (T, error) {
	res := <-f.ch
	return res.Result, res.Err
}

// Get from Future with timeout (in milliseconds)
func (f future[T]) TimedGet(timeout int) (T, error) {
	var res FutureResult[T]
	select {
	case res = <-f.ch:
		return res.Result, res.Err
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		var t T
		return t, ErrGetTimeout
	}
}

// Create Future, once the future is created, it starts running on a new goroutine
func RunAsync[T any](callback func() FutureResult[T]) Future[T] {
	fut := future[T]{callback: callback}
	fut.ch = make(chan FutureResult[T])
	go func(cha chan FutureResult[T]) {
		cha <- fut.callback()
	}(fut.ch)
	return fut
}
