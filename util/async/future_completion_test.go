package async

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/util/utillog"
)

func awaitSignal[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for signal")
		var zero T
		return zero
	}
}

type rejectTaskPool struct {
	goCalls atomic.Int32
}

func (p *rejectTaskPool) Go(func())    { p.goCalls.Add(1) }
func (p *rejectTaskPool) Stop()        {}
func (p *rejectTaskPool) StopAndWait() {}
func (p *rejectTaskPool) Run(f func() error) Future[struct{}] {
	return Submit(p, func() (struct{}, error) { return struct{}{}, f() })
}

// goTask reports that the pool refused the task.
func (p *rejectTaskPool) goTask(func()) bool { return false }

// runThenRejectTaskPool runs the task and still reports rejection, which must not
// complete the future twice.
type runThenRejectTaskPool struct {
	goTaskCalls atomic.Int32
}

func (p *runThenRejectTaskPool) Go(func())    {}
func (p *runThenRejectTaskPool) Stop()        {}
func (p *runThenRejectTaskPool) StopAndWait() {}
func (p *runThenRejectTaskPool) Run(f func() error) Future[struct{}] {
	return Submit(p, func() (struct{}, error) { return struct{}{}, f() })
}

func (p *runThenRejectTaskPool) goTask(f func()) bool {
	p.goTaskCalls.Add(1)
	f()
	return false
}

// recordingPool only implements AsyncPool, so Submit has to fall back to Go.
type recordingPool struct {
	tasks chan func()
}

func (p *recordingPool) Go(f func())  { p.tasks <- f }
func (p *recordingPool) Stop()        {}
func (p *recordingPool) StopAndWait() {}
func (p *recordingPool) Run(f func() error) Future[struct{}] {
	return Submit(p, func() (struct{}, error) { return struct{}{}, f() })
}

func TestSubmitCompletesFutureWithErrTaskDroppedWhenPoolRejects(t *testing.T) {
	p := &rejectTaskPool{}
	f := Submit(p, func() (int, error) { return 9, nil })

	v, err := f.Get()
	if !errors.Is(err, ErrTaskDropped) {
		t.Fatalf("expected ErrTaskDropped, got %v", err)
	}
	me, ok := err.(*errs.MisoErr)
	if !ok {
		t.Fatalf("expected a *errs.MisoErr, got %T", err)
	}
	if code := me.Code(); code != "ASYNC_TASK_DROPPED" {
		t.Fatalf("expected code ASYNC_TASK_DROPPED, got %q", code)
	}
	if v != 0 {
		t.Fatalf("expected zero value for a dropped task, got %v", v)
	}
	if n := p.goCalls.Load(); n != 0 {
		t.Fatalf("pool.Go should not be used when the pool reports a rejection, called %v times", n)
	}
}

func TestSubmitCompletesDroppedFutureOnlyOnce(t *testing.T) {
	p := &runThenRejectTaskPool{}
	f := Submit(p, func() (int, error) { return 9, nil })

	var calls atomic.Int32
	v, err := f.Then(func(int, error) { calls.Add(1) }).Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 9 {
		t.Fatalf("expected the task result to win the first completion, got %v", v)
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("completion hook should run exactly once, ran %v times", n)
	}
}

func TestSubmitFallsBackToGoForPoolsWithoutAcceptanceCapability(t *testing.T) {
	p := &recordingPool{tasks: make(chan func(), 1)}
	f := Submit(p, func() (int, error) { return 9, nil })

	task := awaitSignal(t, p.tasks)
	task()

	v, err := f.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 9 {
		t.Fatalf("expected 9, got %v", v)
	}
}

func TestFireWithNilHandleDoesNotPanic(t *testing.T) {
	orig := utillog.ErrorLog
	defer func() { utillog.ErrorLog = orig }()

	logs := make(chan string, 8)
	utillog.ErrorLog = func(pat string, args ...any) {
		select {
		case logs <- fmt.Sprintf(pat, args...):
		default:
		}
	}
	waitLog := func(substr string) {
		t.Helper()
		deadline := time.After(2 * time.Second)
		for {
			select {
			case msg := <-logs:
				if strings.Contains(msg, substr) {
					return
				}
			case <-deadline:
				t.Fatalf("no log containing %q", substr)
			}
		}
	}

	fireErr := errors.New("fire failed")

	// inline runner: the completion callback runs before Fire returns
	Fire(nil, func() error { return fireErr }, func(f func()) { f() })
	waitLog(fireErr.Error())

	// success must not report a failure
	Fire(nil, func() error { return nil }, func(f func()) { f() })
	select {
	case msg := <-logs:
		t.Fatalf("successful fire should not log a failure: %q", msg)
	default:
	}

	// goroutine runner: done closes after the completion callback has run
	done := make(chan struct{})
	Fire(nil, func() error { return fireErr }, func(f func()) {
		go func() {
			defer close(done)
			f()
		}()
	})
	awaitSignal(t, done)
}

func TestThenOnCompletedFutureFiresBeforeReturning(t *testing.T) {
	completedErr := errors.New("done already")
	f := NewCompletedFuture(11, completedErr)

	fired := false
	derived := f.Then(func(v int, err error) {
		fired = true
		if v != 11 {
			t.Errorf("expected 11, got %v", v)
		}
		if !errors.Is(err, completedErr) {
			t.Errorf("expected completed error, got %v", err)
		}
	})
	if !fired {
		t.Fatal("callback should fire before Then returns")
	}
	v, err := derived.Get()
	if v != 11 {
		t.Fatalf("derived future should carry 11, got %v", v)
	}
	if !errors.Is(err, completedErr) {
		t.Fatalf("derived future should carry the completed error, got %v", err)
	}

	// same for a future that completed on its own before Then was called
	ran := Run(func() (int, error) { return 11, completedErr })
	if _, err := ran.Get(); !errors.Is(err, completedErr) {
		t.Fatalf("unexpected error from parent: %v", err)
	}
	fired = false
	derived = ran.Then(func(int, error) { fired = true })
	if !fired {
		t.Fatal("callback should fire before Then returns on a completed future")
	}
	if v, err := derived.Get(); v != 11 || !errors.Is(err, completedErr) {
		t.Fatalf("derived future should carry the parent result, got (%v, %v)", v, err)
	}
}

func TestThenNilCallbackPanics(t *testing.T) {
	assertPanics := func(name string, op func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s: expected panic", name)
			}
		}()
		op()
	}

	assertPanics("completedFuture", func() { NewCompletedFuture(1, nil).Then(nil) })
	assertPanics("future", func() {
		started := make(chan struct{})
		release := make(chan struct{})
		f := Run(func() (int, error) {
			close(started)
			<-release
			return 1, nil
		})
		<-started
		close(release)
		f.Then(nil)
	})
}

func TestTwoThenCallsOnOneFutureBothFire(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	f := Run(func() (int, error) {
		close(started)
		<-release
		return 3, nil
	})
	<-started

	fired := make(chan string, 2)
	f.Then(func(int, error) { fired <- "first" })
	f.Then(func(int, error) { fired <- "second" })

	close(release)

	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		seen[awaitSignal(t, fired)] = true
	}
	if !seen["first"] || !seen["second"] {
		t.Fatalf("both Then callbacks should fire, got %v", seen)
	}
}

func TestPanickingThenCallbackDoesNotStopNextCallback(t *testing.T) {
	parentErr := errors.New("boom")
	started := make(chan struct{})
	release := make(chan struct{})
	f := Run(func() (int, error) {
		close(started)
		<-release
		return 5, parentErr
	})
	<-started

	next := make(chan error, 1)
	var gotValue int
	var gotErr error
	f.Then(func(int, error) {
		panic("callback exploded")
	}).Then(func(v int, err error) {
		gotValue, gotErr = v, err
		next <- err
	})

	close(release)
	awaitSignal(t, next)

	if gotValue != 5 {
		t.Fatalf("expected value 5 after panicking callback, got %v", gotValue)
	}
	if !errors.Is(gotErr, parentErr) {
		t.Fatalf("expected parent error after panicking callback, got %v", gotErr)
	}
}

func TestDerivedFutureResolvesWithParentResult(t *testing.T) {
	parentErr := errors.New("parent failed")
	f := Run(func() (int, error) {
		return 42, parentErr
	})

	var callbackValue int
	var callbackErr error
	derived := f.Then(func(v int, err error) {
		callbackValue, callbackErr = v, err
	})

	v, err := derived.Get()
	if v != 42 {
		t.Fatalf("expected derived value 42, got %v", v)
	}
	if !errors.Is(err, parentErr) {
		t.Fatalf("expected parent error, got %v", err)
	}
	if callbackValue != 42 {
		t.Fatalf("expected callback value 42, got %v", callbackValue)
	}
	if !errors.Is(callbackErr, parentErr) {
		t.Fatalf("expected parent error in callback, got %v", callbackErr)
	}
}

func TestChainedThenCallbacksRunInRegistrationOrder(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	f := Run(func() (int, error) {
		close(started)
		<-release
		return 1, nil
	})
	<-started

	order := make(chan string, 2)
	f.Then(func(int, error) {
		order <- "first"
	}).Then(func(int, error) {
		order <- "second"
	})

	close(release)

	for _, want := range []string{"first", "second"} {
		if got := awaitSignal(t, order); got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	}
}

func TestTimedGetTimeoutReturnsZeroValueAndErrGetTimeout(t *testing.T) {
	// The task finishes after the timed get has given up, and nothing in the
	// test orders that write before the reader, which is what the race detector
	// looks at.
	f := Run(func() (int, error) {
		time.Sleep(20 * time.Millisecond)
		return 7, nil
	})

	v, err := f.TimedGet(1)
	if !errors.Is(err, ErrGetTimeout) {
		t.Fatalf("expected ErrGetTimeout, got err=%v", err)
	}
	if v != 0 {
		t.Fatalf("expected zero value on timeout, got %v", v)
	}

	v, err = f.Get()
	if err != nil {
		t.Fatalf("unexpected error after completion: %v", err)
	}
	if v != 7 {
		t.Fatalf("expected 7, got %v", v)
	}
}
