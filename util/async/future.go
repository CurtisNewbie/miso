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

	// Then callback to be invoked when the Future is completed.
	//
	// Then callback should only be set once for every Future.
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
			rail.Errorf("Async task failed (%v), took: %v, %v", caller, time.Since(start), err)
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
func Submit[T any](pool AsyncPool, task func() (T, error)) Future[T] {
	fut, wrp := buildFuture(nil, task)
	pool.Go(wrp)
	return fut
}

// AwaitFutures represent tasks that are submitted to the pool asynchronously whose results are awaited together.
//
// AwaitFutures should only be used once for the same group of tasks.
//
// Use [NewAwaitFutures] to create one.
type AwaitFutures[T any] struct {
	pool    AsyncPool
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
		a.futures = append(a.futures, Submit[T](a.pool, delegate))
	} else {
		a.futures = append(a.futures, Run[T](delegate))
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

func (f *future[T]) ThenErr(tf func(error)) {
	f.Then(func(t T, err error) {
		tf(err)
	})
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

func buildFuture[T any](rail interface{ Errorf(string, ...any) }, task func() (T, error)) (Future[T], func()) {
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

func (f *completedFuture[T]) Then(tf func(T, error)) {
	tf(f.res, f.err)
}

func (f *completedFuture[T]) ThenErr(tf func(error)) {
	f.Then(func(t T, err error) { tf(err) })
}
