package util

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

var (
	_ Future[any] = (*future[any])(nil)

	PanicLog func(pat string, args ...any) = Printlnf
)

var (
	ErrGetTimeout = errors.New("future.TimedGet timeout")
)

// Result of a asynchronous task.
type Future[T any] interface {

	// Get result without timeout.
	Get() (T, error)

	// Get result with timeout, returns ErrGetTimeout if timeout exceeded.
	TimedGet(timeout int) (T, error)

	// Then callback to be invoked when the Future is completed.
	//
	// Then callback should only be set once for every Future.
	Then(tf func(T, error))
}

type future[T any] struct {
	res  T
	err  error
	done *SignalOnce

	// mutex mainly used to sync between .Then() and the task func
	//
	// the future's res, err doesn't really need mutex
	thenMu *sync.Mutex
	then   func(T, error)
}

func (f *future[T]) Then(tf func(T, error)) {
	if tf == nil {
		panic("Future.Then callback cannot be nil")
	}

	f.thenMu.Lock()
	f.then = func(t T, err error) {
		defer recoverPanic()
		tf(t, err)
	}

	if f.done.Closed() {
		doThen := f.then
		f.thenMu.Unlock()
		doThen(f.Get())
	} else {
		f.thenMu.Unlock()
	}
}

// Get from Future indefinitively
func (f *future[T]) Get() (T, error) {
	if err := f.wait(0); err != nil {
		return f.res, err
	}
	return f.res, f.err
}

func (f *future[T]) wait(timeout int) error {
	if f.done.TimedWait(time.Duration(timeout) * time.Millisecond) {
		return ErrGetTimeout
	}
	return nil
}

// Get from Future with timeout (in milliseconds)
func (f *future[T]) TimedGet(timeout int) (T, error) {
	if err := f.wait(timeout); err != nil {
		return f.res, err
	}
	return f.res, f.err
}

func buildFuture[T any](task func() (T, error)) (Future[T], func()) {
	sig := NewSignalOnce()
	fut := future[T]{
		thenMu: &sync.Mutex{},
		done:   sig,
	}
	wrp := func() {
		var t T
		var err error

		defer func() {

			// task() panicked, change err
			if v := recover(); v != nil {
				PanicLog("panic recovered, %v\n%v", v, UnsafeByt2Str(debug.Stack()))
				if verr, ok := v.(error); ok {
					err = verr
				} else {
					err = fmt.Errorf("%v", v)
				}
			}

			fut.thenMu.Lock()
			fut.res = t
			fut.err = err
			sig.Notify()

			if fut.then != nil {
				doThen := fut.then
				fut.thenMu.Unlock()
				doThen(fut.res, fut.err)
			} else {
				fut.thenMu.Unlock()
			}
		}()

		t, err = task()
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

// Create AsyncPool with number of workers equals to number of cpu cores and a task queue of size 500.
func NewCpuAsyncPool() *AsyncPool {
	c := runtime.NumCPU()
	if c < 2 {
		c = 2 // at least 2
	}
	return NewAsyncPool(500, c)
}

// Create AsyncPool with number of workers equals to two times of the number of cpu cores and a task queue of size 500.
func NewIOAsyncPool() *AsyncPool {
	return NewAsyncPool(500, runtime.NumCPU()*2)
}

// Stop the pool.
//
// Once the pool is stopped, new tasks submitted are executed directly by the caller.
func (p *AsyncPool) Stop() {
	if atomic.CompareAndSwapInt32(&p.stopped, 0, 1) {
		p.stopOnce.Do(func() { close(p.tasks) })
	}
}

// Stop the pool and wait until existing workers drain all the remaining tasks.
//
// Once the pool is stopped, new tasks submitted are executed directly by the caller.
func (p *AsyncPool) StopAndWait() {
	p.Stop()
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
				PanicLog("panic recovered, %v\n%v", v, UnsafeByt2Str(debug.Stack()))
			}
		}()
		op()
	}
}

func recoverPanic() {
	if v := recover(); v != nil {
		PanicLog("panic recovered, %v\n%v", v, UnsafeByt2Str(debug.Stack()))
	}
}

type SignalOnce struct {
	c      context.Context
	cancel func()
}

func (s *SignalOnce) Closed() bool {
	return s.c.Err() != nil
}

func (s *SignalOnce) Wait() {
	s.TimedWait(0)
}

func (s *SignalOnce) TimedWait(timeout time.Duration) (isTimeout bool) {
	isTimeout = false
	if timeout < 1 {
		<-s.c.Done()
	} else {
		select {
		case <-s.c.Done():
		case <-time.After(timeout):
			isTimeout = true
			return
		}
	}
	return
}

func (s *SignalOnce) Notify() {
	s.cancel()
}

func NewSignalOnce() *SignalOnce {
	c, f := context.WithCancel(context.Background())
	return &SignalOnce{
		c:      c,
		cancel: f,
	}
}

type BatchTask[T any] struct {
	parallel int
	pipe     chan T
	wg       *sync.WaitGroup
	consumer func(T)
}

// Wait until all generated tasks are completed and close pipeline channel.
func (b *BatchTask[T]) Wait() {
	b.wg.Wait()
	close(b.pipe)
}

// Close underlying pipeline channel without waiting.
func (b *BatchTask[T]) Close() {
	close(b.pipe)
}

func (b *BatchTask[T]) preHeat() {
	for i := 0; i < b.parallel; i++ {
		go func() {
			for t := range b.pipe {
				b.consumer(t)
			}
		}()
	}
}

// Generate task.
func (b *BatchTask[T]) Generate(task T) {
	b.wg.Add(1)
	b.pipe <- task
}

// Create a batch of concurrent task for one time use.
func NewBatchTask[T any](parallel int, bufferSize int, consumer func(T)) *BatchTask[T] {
	bt := &BatchTask[T]{
		parallel: parallel,
		pipe:     make(chan T, bufferSize),
		wg:       &sync.WaitGroup{},
	}
	bt.consumer = func(t T) {
		defer bt.wg.Done()
		consumer(t)
	}
	bt.preHeat()
	return bt
}
