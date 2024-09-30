package util

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
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
				Printlnf("panic recovered, %v\n%v", v, UnsafeByt2Str(debug.Stack()))
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

// AwaitFutures represent tasks that are submitted to the pool asynchronously whose results are awaited together.
//
// AwaitFutures should only be used once for the same group of tasks.
//
// Use miso.NewAwaitFutures() to create one.
type AwaitFutures[T any] struct {
	pool    *AsyncPool
	wg      sync.WaitGroup
	futures []Future[T]
}

// Submit task to AwaitFutures.
func (a *AwaitFutures[T]) SubmitAsync(task func() (T, error)) {
	a.wg.Add(1)
	delegate := func() (T, error) {
		defer a.wg.Done()
		return task()
	}
	if a.pool != nil {
		a.futures = append(a.futures, SubmitAsync[T](a.pool, delegate))
	} else {
		a.futures = append(a.futures, RunAsync[T](delegate))
	}
}

// Await results of all tasks.
func (a *AwaitFutures[T]) Await() []Future[T] {
	a.wg.Wait()
	return a.futures
}

// Await results of all tasks and return any error that is found in the task Futures.
func (a *AwaitFutures[T]) AwaitAnyErr() error {
	fut := a.Await()
	for _, f := range fut {
		_, err := f.Get()
		if err != nil {
			return err
		}
	}
	return nil
}

// Create new AwaitFutures for a group of tasks.
//
// *AsyncPool is optional, provide nil if not needed.
func NewAwaitFutures[T any](pool *AsyncPool) *AwaitFutures[T] {
	return &AwaitFutures[T]{
		pool:    pool,
		futures: make([]Future[T], 0, 2),
	}
}

// Create func that calls SubmitAsync(...) with the given pool.
func NewSubmitAsyncFunc[T any](pool *AsyncPool) func(task func() (T, error)) Future[T] {
	return func(task func() (T, error)) Future[T] {
		return SubmitAsync[T](pool, task)
	}
}

// A long live, bounded pool of goroutines.
//
// Use miso.NewAsyncPool to create a new pool.
//
// AsyncPool internally maintains a task queue with limited size and limited number of workers. If the task queue is full,
// the caller of *AsyncPool.Go is blocked indefinitively.
type AsyncPool struct {
	stopped      int32
	stopOnce     *sync.Once
	drainTasksWg *sync.WaitGroup
	tasks        chan func()
	workers      chan struct{}
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
		tasks:        make(chan func(), maxTasks),
		workers:      make(chan struct{}, maxWorkers),
		stopped:      0,
		stopOnce:     &sync.Once{},
		drainTasksWg: &sync.WaitGroup{},
	}
}

// Stop the pool.
//
// Once the pool is stopped, new tasks submitted are executed directly by the caller.
func (p *AsyncPool) Stop() {
	atomic.StoreInt32(&p.stopped, 1)
}

// Stop the pool and wait until existing workers drain all the remaining tasks.
//
// Once the pool is stopped, new tasks submitted are executed directly by the caller.
func (p *AsyncPool) StopAndWait() {
	atomic.StoreInt32(&p.stopped, 1)
	p.drainTasksWg.Wait()
}

func (p *AsyncPool) isStopped() bool {
	return atomic.LoadInt32(&p.stopped) == 1
}

// Submit task to the pool.
//
// If the task queue is full, the caller is blocked.
//
// If the pool is closed, caller will execute the submitted task directly.
func (p *AsyncPool) Go(f func()) {

	if p.isStopped() {
		p.stopOnce.Do(func() { close(p.tasks) })
		f() // caller runs the task
		return
	}

	p.drainTasksWg.Add(1)
	wrp := func() {
		defer p.drainTasksWg.Done()
		f()
	}

	select {
	case p.workers <- struct{}{}:
		go p.spawn(wrp)
	case p.tasks <- wrp:
		// channel select is completely random
		// extra select is to make sure that we have at least one worker running
		// or else tasks may be put straight into the task queue without any worker
		// conc package has the same problem as well :(, but the way we use the pool is different.
		select {
		case p.workers <- struct{}{}:
			go p.spawn(nil)
		default:
		}
		return
	}
}

// spawn a new worker.
func (p *AsyncPool) spawn(first func()) {
	defer func() { <-p.workers }()

	if first != nil {
		PanicSafeFunc(first)()
	}

	for f := range p.tasks {
		PanicSafeFunc(f)()
	}
}

func PanicSafeFunc(op func()) func() {
	return func() {
		defer func() {
			if v := recover(); v != nil {
				Printlnf("panic recovered, %v\n%v", v, UnsafeByt2Str(debug.Stack()))
			}
		}()
		op()
	}
}
