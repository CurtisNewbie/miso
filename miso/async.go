package miso

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
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
func SubmitAsync[T any](pool *AsyncPool, task func() (T, error)) Future[T] {
	fut, wrp := buildFuture(task)
	pool.Go(wrp)
	return fut
}

// AwaitFutures represent multiple tasks that will be submitted to the pool asynchronously whose results will be awaited together.
//
// AwaitFutures should only be used once everytime it's needed.
//
// Use miso.NewAwaitFutures() to create one.
type AwaitFutures[T any] struct {
	pool    *AsyncPool
	wg      sync.WaitGroup
	futures []Future[T]
}

func (a *AwaitFutures[T]) SubmitAsync(task func() (T, error)) {
	a.wg.Add(1)
	a.futures = append(a.futures, SubmitAsync[T](a.pool, func() (T, error) {
		defer a.wg.Done()
		return task()
	}))
}

func (a *AwaitFutures[T]) Await() []Future[T] {
	a.wg.Wait()
	return a.futures
}

func NewAwaitFutures[T any](pool *AsyncPool) *AwaitFutures[T] {
	return &AwaitFutures[T]{
		pool:    pool,
		futures: make([]Future[T], 0, 2),
	}
}

// A long live, bounded pool of goroutines.
//
// Use miso.NewAsyncPool to create a new pool.
//
// AsyncPool internally maintains a task queue with limited size and limited number of workers. If the task queue is full,
// the caller of *AsyncPool.Go is blocked indefinitively.
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
		// channel select is completely random
		// extra select is to make sure that we have at least one worker running
		// or else tasks may be put straight into the task queue without any worker
		// conc package has the same problem as well :(, but the way we use the pool is different.
		select {
		case p.workers <- struct{}{}:
			go p.spawnEmpty()
		default:
		}
		return
	}
}

// spawn a new worker.
func (p *AsyncPool) spawn(first func()) {
	defer func() { <-p.workers }()

	if first != nil {
		first()
	}

	for f := range p.tasks {
		f()
	}
}

// spawn a new worker.
func (p *AsyncPool) spawnEmpty() {
	defer func() { <-p.workers }()

	for f := range p.tasks {
		f()
	}
}
