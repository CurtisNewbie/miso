package async

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/util/pair"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/src"
	"github.com/curtisnewbie/miso/util/utillog"
)

var (
	_ Future[any] = (*future[any])(nil)
	_ Future[any] = (*completedFuture[any])(nil)
)

var (
	ErrGetTimeout = errors.New("future.TimedGet timeout")

	// ErrTaskDropped reports that a pool refused to run a task, so the task never
	// produced a result of its own.
	ErrTaskDropped = errs.NewErrfCode("ASYNC_TASK_DROPPED", "async task dropped, pool is full or stopped")
)

// taskAcceptor is an optional capability of an AsyncPool. It reports whether the
// pool accepted the task, so that a rejected task can complete its Future instead
// of leaving it pending forever.
type taskAcceptor interface {
	goTask(func()) bool
}

// Result of a asynchronous task.
type Future[T any] interface {

	// Get result without timeout.
	Get() (T, error)

	// Get result with timeout, returns ErrGetTimeout if timeout exceeded.
	TimedGet(timeout int) (T, error)

	// Then callback to be invoked when the Future is completed.
	//
	// The callback runs on the goroutine that completes the Future, in the order
	// the callbacks were registered. A derived Future is returned, it completes
	// with the same result once the callback returns, so callbacks can be chained.
	//
	// If the Future is already completed, the callback runs synchronously in the
	// caller and a completed Future is returned.
	Then(tf func(T, error)) Future[T]

	// Then callback to be invoked when the Future is completed.
	//
	// Multiple callbacks are supported, each runs once when the Future completes.
	ThenErr(tf func(error))
}

// Fire async task on new goroutine and forget about it.
func Fire(rail interface{ Errorf(string, ...any) }, task func() error, runner ...func(func())) {
	start := time.Now()
	caller := src.GetCallerFn()
	fut, wrp := buildFuture(rail, func() (struct{}, error) {
		return struct{}{}, task()
	})
	fut.ThenErr(func(err error) {
		if err != nil {
			if rail != nil {
				rail.Errorf("Async task failed (%v), took: %v, %v", caller, time.Since(start), err)
			} else {
				utillog.ErrorLog("Async task failed (%v), took: %v, %v", caller, time.Since(start), err)
			}
		} else if rrail, ok := rail.(interface{ Infof(string, ...any) }); ok {
			rrail.Infof("Async task completed (%v), took: %v", caller, time.Since(start))
		}
	})
	runner1, ok := slutil.First(runner)
	if ok {
		runner1(wrp)
	} else {
		go wrp()
	}
}

// Create Future, once the future is created, it starts running on a new goroutine.
func Run[T any](task func() (T, error)) Future[T] {
	fut, wrp := buildFuture(nil, task)
	go wrp()
	return fut
}

// Create Future, once the future is created, it starts running on a saperate goroutine from the pool.
//
// If the pool reports that it refused the task, the returned Future completes with [ErrTaskDropped].
func Submit[T any](pool AsyncPool, task func() (T, error)) Future[T] {
	fut, wrp := buildFuture(nil, task)
	if acc, ok := pool.(taskAcceptor); ok {
		if !acc.goTask(wrp) {
			var zero T
			fut.complete(zero, ErrTaskDropped.New())
		}
		return fut
	}
	pool.Go(wrp)
	return fut
}

// AwaitFutures represent tasks that are submitted to the pool asynchronously whose results are awaited together.
//
// A group is sealed on the first Await: Await returns a copy of the Futures, and
// submitting to a sealed group panics.
//
// SubmitAsync holds the group lock while it submits, so it must not be called
// from inside a task of the same group.
//
// Use [NewAwaitFutures] to create one.
type AwaitFutures[T any] struct {
	// mu guards the pool, the futures and the sealed flag
	mu      sync.Mutex
	pool    AsyncPool
	sealed  bool
	futures []Future[T]
}

// Submit task to AwaitFutures.
//
// Panics if the group is already sealed by a call to Await.
func (a *AwaitFutures[T]) SubmitAsync(task func() (T, error)) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.sealed {
		panic("AwaitFutures.SubmitAsync called after Await, the group is sealed")
	}

	if a.pool != nil {
		a.futures = append(a.futures, Submit[T](a.pool, task))
	} else {
		a.futures = append(a.futures, Run[T](task))
	}
}

// Await results of all tasks.
//
// The returned slice is a copy, so the caller cannot observe later submissions,
// and repeated calls return the same sealed set.
func (a *AwaitFutures[T]) Await() []Future[T] {
	a.mu.Lock()
	a.sealed = true
	futures := make([]Future[T], len(a.futures))
	copy(futures, a.futures)
	a.mu.Unlock()

	for _, f := range futures {
		// Only completion matters here. The value and the error are read by the
		// callers that need them: AwaitAnyErr, AwaitResultAnyErr, AwaitResultAll
		// and any caller iterating the returned Futures.
		_, _ = f.Get()
	}
	return futures
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

// Await results of all tasks and return any error that is found in the task Futures.
func (a *AwaitFutures[T]) AwaitResultAnyErr() ([]T, error) {
	fut := a.Await()
	res := make([]T, 0, len(fut))
	for _, f := range fut {
		v, err := f.Get()
		if err != nil {
			return nil, err
		}
		res = append(res, v)
	}
	return res, nil
}

// Await results of all tasks.
func (a *AwaitFutures[T]) AwaitResultAll() []pair.Pair[T, error] {
	fut := a.Await()
	res := make([]pair.Pair[T, error], 0, len(fut))
	for _, f := range fut {
		v, err := f.Get()
		res = append(res, pair.New(v, err))
	}
	return res
}

// Create new AwaitFutures for a group of tasks.
//
// *AsyncPool is optional, provide nil if not needed.
func NewAwaitFutures[T any](pool AsyncPool) *AwaitFutures[T] {
	return &AwaitFutures[T]{
		pool:    pool,
		futures: make([]Future[T], 0, 2),
	}
}

// Create func that calls SubmitAsync(...) with the given pool.
func NewSubmitAsyncFunc[T any](pool AsyncPool) func(task func() (T, error)) Future[T] {
	return func(task func() (T, error)) Future[T] {
		return Submit[T](pool, task)
	}
}

func NewCompletedFuture[T any](t T, err error) Future[T] {
	return &completedFuture[T]{res: t, err: err}
}

type future[T any] struct {
	// mu guards res, err, completed and the completion hooks
	mu        sync.Mutex
	res       T
	err       error
	completed bool
	hooks     []func(T, error)
	done      *SignalOnce
}

func newFuture[T any]() *future[T] {
	return &future[T]{done: NewSignalOnce()}
}

// Complete the future with the given result. A future completes exactly once,
// later calls are ignored.
//
// The result is published and waiters are woken while holding the lock, then the
// completion hooks run in registration order on the completing goroutine.
func (f *future[T]) complete(t T, err error) bool {
	f.mu.Lock()
	if f.completed {
		f.mu.Unlock()
		return false
	}
	f.res = t
	f.err = err
	f.completed = true

	// publish the result before waking waiters, while still holding the lock
	f.done.Notify()

	hooks := f.hooks
	f.hooks = nil
	f.mu.Unlock()

	for _, hook := range hooks {
		hook(t, err)
	}
	return true
}

// Create a completion hook that runs the callback and then completes the derived
// future with the parent's result unchanged.
func (f *future[T]) newHook(tf func(T, error), derived *future[T]) func(T, error) {
	return func(t T, err error) {
		func() {
			defer recoverPanic()
			tf(t, err)
		}()
		derived.complete(t, err)
	}
}

func (f *future[T]) Then(tf func(T, error)) Future[T] {
	if tf == nil {
		panic("Future.Then callback cannot be nil")
	}

	f.mu.Lock()
	if f.completed {
		t, err := f.res, f.err
		f.mu.Unlock()
		func() {
			defer recoverPanic()
			tf(t, err)
		}()
		return NewCompletedFuture(t, err)
	}
	derived := newFuture[T]()
	f.hooks = append(f.hooks, f.newHook(tf, derived))
	f.mu.Unlock()
	return derived
}

func (f *future[T]) ThenErr(tf func(error)) {
	f.Then(func(t T, err error) {
		tf(err)
	})
}

// Get from Future indefinitively
func (f *future[T]) Get() (T, error) {
	f.done.Wait()
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.res, f.err
}

// Get from Future with timeout (in milliseconds)
func (f *future[T]) TimedGet(timeout int) (T, error) {
	if f.done.TimedWait(time.Duration(timeout) * time.Millisecond) {
		var zero T
		return zero, ErrGetTimeout
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.res, f.err
}

func buildFuture[T any](rail interface{ Errorf(string, ...any) }, task func() (T, error)) (*future[T], func()) {
	fut := newFuture[T]()
	wrp := func() {
		var t T
		var err error

		defer func() {

			// task() panicked, change err
			if v := recover(); v != nil {
				if rail != nil {
					if verr, ok := v.(*errs.MisoErr); ok {
						rail.Errorf("Panic recovered, %v", verr)
					} else {
						rail.Errorf("Panic recovered, %v\n%v", v, string(debug.Stack()))
					}
				} else {
					utillog.ErrorLog("panic recovered, %v\n%v", v, string(debug.Stack()))
				}
				if verr, ok := v.(error); ok {
					err = verr
				} else {
					err = fmt.Errorf("%v", v)
				}
			}

			fut.complete(t, err)
		}()

		t, err = task()
	}
	return fut, wrp
}

type completedFuture[T any] struct {
	res T
	err error
}

func (f *completedFuture[T]) Get() (T, error) {
	return f.res, f.err
}

func (f *completedFuture[T]) TimedGet(timeout int) (T, error) {
	return f.res, f.err
}

func (f *completedFuture[T]) Then(tf func(T, error)) Future[T] {
	if tf == nil {
		panic("Future.Then callback cannot be nil")
	}
	func() {
		defer recoverPanic()
		tf(f.res, f.err)
	}()
	return f
}

func (f *completedFuture[T]) ThenErr(tf func(error)) {
	f.Then(func(t T, err error) { tf(err) })
}
