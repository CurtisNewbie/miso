package miso

import (
	"errors"
	"fmt"
	"runtime/debug"
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
	ch chan func() (T, error)
}

// Get from Future indefinitively
func (f *future[T]) Get() (T, error) {
	getResult := <-f.ch
	return getResult()
}

// Get from Future with timeout (in milliseconds)
func (f *future[T]) TimedGet(timeout int) (T, error) {
	select {
	case obtainResult := <-f.ch:
		return obtainResult()
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		var t T
		return t, ErrGetTimeout
	}
}

func buildFuture[T any](task func() (T, error)) (Future[T], func()) {
	fut := future[T]{}
	fut.ch = make(chan func() (T, error), 1)
	wrp := func() {
		defer func() {
			if v := recover(); v != nil {
				Warnf("panic recovered, %v\n%v", v, string(debug.Stack()))
				var t T
				if err, ok := v.(error); ok {
					fut.ch <- func() (T, error) { return t, err }
				} else {
					fut.ch <- func() (T, error) { return t, fmt.Errorf("%v", v) }
				}
			}
		}()

		t, err := task()
		fut.ch <- func() (T, error) { return t, err }
	}
	return &fut, wrp
}

// Create Future, once the future is created, it starts running on a new goroutine.
func RunAsync[T any](task func() (T, error)) Future[T] {
	fut, wrp := buildFuture(task)
	go wrp()
	return fut
}

// Create Future, once the future is created, it starts running on a saperate goroutine from the pool.
func RunAsyncPool[T any](pool *AsyncPool, task func() (T, error)) Future[T] {
	fut, wrp := buildFuture(task)
	pool.Go(wrp)
	return fut
}

// Bounded pool of goroutines.
type AsyncPool struct {
	tasks   chan func()
	workers chan struct{}
}

// Create a bounded pool of goroutines.
//
// The maxTasks determines the capacity of the task queues. If the task queue is full,
// the caller of *AsyncPool.Go is blocked.
//
// The maxWorkers determines the max number of workers.
func NewAsyncPool(maxTasks int, maxWorkers int) *AsyncPool {
	if maxTasks < 0 {
		maxTasks = 0
	}
	if maxWorkers < 1 {
		maxWorkers = 1
	}
	return &AsyncPool{
		tasks:   make(chan func(), maxTasks),
		workers: make(chan struct{}, maxWorkers),
	}
}

// Submit task to the pool.
//
// If the task queue is full, the caller is blocked.
//
// If the pool is closed, return ErrAsyncPoolClosed.
func (p *AsyncPool) Go(f func()) {
	select {
	case p.workers <- struct{}{}:
		go func() { p.spawn(f) }()
	case p.tasks <- f:
		return
	}
}

// spawn a new worker.
func (p *AsyncPool) spawn(first func()) {
	Debug("Spawned Worker")
	defer func() { <-p.workers }()

	if first != nil {
		first()
	}

	for f := range p.tasks {
		f()
	}
	Debug("Worker exited")
}
