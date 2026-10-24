package async

import (
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestAntsPoolAfterStopRunsInlineAndStopsIdempotently(t *testing.T) {
	pool := NewAntsAsyncPool(2)
	pool.Stop()
	pool.Stop()

	ran := make(chan struct{}, 1)
	pool.Go(func() { ran <- struct{}{} })
	awaitSignal(t, ran)

	f := Submit(pool, func() (int, error) { return 7, nil })
	v, err := f.TimedGet(2000)
	if err != nil {
		t.Fatalf("submission after stop should run inline, got err %v", err)
	}
	if v != 7 {
		t.Fatalf("expected 7, got %v", v)
	}
	pool.StopAndWait()
}

func TestAntsPoolFailedSubmitGoesThroughFallback(t *testing.T) {
	pool := NewAntsAsyncPool(1, FallbackCallerRun())

	started := make(chan struct{})
	release := make(chan struct{})
	pool.Go(func() {
		close(started)
		<-release
	})
	awaitSignal(t, started)

	// the only worker is busy and ants is non-blocking, so the submit fails and
	// the fallback runs the task on the caller
	f := Submit(pool, func() (int, error) { return 4, nil })
	v, err := f.TimedGet(2000)
	if err != nil {
		t.Fatalf("fallback should run the task, got err %v", err)
	}
	if v != 4 {
		t.Fatalf("expected 4, got %v", v)
	}

	close(release)
	pool.StopAndWait()
}

func TestBoundedPoolStopAndSubmitStorm(t *testing.T) {
	pool := NewBoundedAsyncPool(16, 4, FallbackCallerRun())

	const n = 200
	futures := make([]Future[int], 0, n)
	var mu sync.Mutex
	var submitters sync.WaitGroup

	stop := make(chan struct{})
	go func() {
		<-stop
		pool.Stop()
	}()

	for i := 0; i < n; i++ {
		submitters.Add(1)
		go func(v int) {
			defer submitters.Done()
			f := Submit(pool, func() (int, error) { return v, nil })
			mu.Lock()
			futures = append(futures, f)
			mu.Unlock()
		}(i)
	}

	close(stop)
	submitters.Wait()
	pool.StopAndWait()

	mu.Lock()
	defer mu.Unlock()
	for i, f := range futures {
		v, err := f.TimedGet(5000)
		if errors.Is(err, ErrTaskDropped) {
			continue
		}
		if err != nil {
			t.Fatalf("future %v should resolve, got err %v", i, err)
		}
		if v < 0 || v >= n {
			t.Fatalf("future %v returned an unexpected value %v", i, v)
		}
	}
}

func TestBoundedPoolStopBarrierWaitsForQueuedTask(t *testing.T) {
	pool := NewBoundedAsyncPool(10, 1)

	started := make(chan struct{})
	releaseFirst := make(chan struct{})
	pool.Go(func() {
		close(started)
		<-releaseFirst
	})
	awaitSignal(t, started)

	taskStarted := make(chan struct{})
	releaseTask := make(chan struct{})
	pool.Go(func() {
		close(taskStarted)
		<-releaseTask
	})

	pool.Stop()

	stopped := make(chan struct{})
	go func() {
		pool.StopAndWait()
		close(stopped)
	}()

	close(releaseFirst)
	// the worker drained the queued task and is now running it
	awaitSignal(t, taskStarted)

	select {
	case <-stopped:
		t.Fatal("StopAndWait returned while an accepted task was still running")
	default:
	}

	close(releaseTask)
	awaitSignal(t, stopped)
}

func TestBoundedPoolDrainsQueuedTasksOnStop(t *testing.T) {
	pool := NewBoundedAsyncPool(10, 1)

	started := make(chan struct{})
	release := make(chan struct{})
	pool.Go(func() {
		close(started)
		<-release
	})
	awaitSignal(t, started)

	const n = 5
	var mu sync.Mutex
	ran := make([]int, 0, n)
	for i := 0; i < n; i++ {
		v := i
		pool.Go(func() {
			mu.Lock()
			ran = append(ran, v)
			mu.Unlock()
		})
	}

	pool.Stop()
	close(release)

	stopped := make(chan struct{})
	go func() {
		pool.StopAndWait()
		close(stopped)
	}()
	awaitSignal(t, stopped)

	mu.Lock()
	defer mu.Unlock()
	if len(ran) != n {
		t.Fatalf("expected %v queued tasks to run, got %v", n, len(ran))
	}
}

func TestBoundedPoolAfterStopRunsInlineAndStopsIdempotently(t *testing.T) {
	pool := NewBoundedAsyncPool(2, 2)
	pool.Stop()
	pool.Stop()

	ran := make(chan struct{}, 1)
	pool.Go(func() { ran <- struct{}{} })
	awaitSignal(t, ran)

	f := Submit(pool, func() (int, error) { return 5, nil })
	v, err := f.TimedGet(2000)
	if err != nil {
		t.Fatalf("submission after stop should run inline, got err %v", err)
	}
	if v != 5 {
		t.Fatalf("expected 5, got %v", v)
	}
	pool.StopAndWait()
}

func TestBoundedPoolCallerRunFallbackTaskCanStopOwnPool(t *testing.T) {
	// no worker, no queue: every submission goes through the caller-run fallback
	pool := NewBoundedAsyncPool(0, 0, FallbackCallerRun())

	done := make(chan struct{})
	go func() {
		defer close(done)
		f := Submit(pool, func() (int, error) {
			// must not deadlock: the fallback has to run without holding the
			// pool's state lock
			pool.Stop()
			return 1, nil
		})
		if v, err := f.TimedGet(2000); err != nil || v != 1 {
			t.Errorf("expected (1, nil), got (%v, %v)", v, err)
		}
	}()
	awaitSignal(t, done)

	barrier := make(chan struct{})
	go func() {
		pool.StopAndWait()
		close(barrier)
	}()
	awaitSignal(t, barrier)
}

func TestBoundedPoolStoppedDropFallbackCompletesFutureWithErrTaskDropped(t *testing.T) {
	pool := NewBoundedAsyncPool(1, 1, FallbackDropTask())
	pool.Stop()

	f := Submit(pool, func() (int, error) { return 9, nil })
	v, err := f.TimedGet(2000)
	if !errors.Is(err, ErrTaskDropped) {
		t.Fatalf("expected ErrTaskDropped, got (%v, %v)", v, err)
	}
	if v != 0 {
		t.Fatalf("expected zero value for a dropped task, got %v", v)
	}
}

func TestAntsPoolStopBarrierWaitsForRunningTask(t *testing.T) {
	pool := NewAntsAsyncPool(1)

	started := make(chan struct{})
	release := make(chan struct{})
	pool.Go(func() {
		close(started)
		<-release
	})
	awaitSignal(t, started)

	stopped := make(chan struct{})
	go func() {
		pool.StopAndWait()
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Fatal("StopAndWait returned while a task was still running")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	awaitSignal(t, stopped)
}

func TestBoundedPoolIdleExitDoesNotStrandTasks(t *testing.T) {
	pool := NewBoundedAsyncPool(1, 1)
	// Production idleDur is one minute, which would never expire inside a test, so
	// the package-private field is shortened to force idle exits. The assertions
	// below are about task resolution and stop returning, never goroutine counts.
	pool.idleDur = time.Millisecond

	const n = 100
	futures := make([]Future[int], 0, n)
	for i := 0; i < n; i++ {
		v := i
		futures = append(futures, Submit(pool, func() (int, error) { return v, nil }))
		// give the single worker a chance to hit its idle timer between submits
		runtime.Gosched()
	}

	for i, f := range futures {
		if _, err := f.TimedGet(2000); err != nil {
			t.Fatalf("task %v was stranded by an idle exit: %v", i, err)
		}
	}

	stopped := make(chan struct{})
	go func() {
		pool.StopAndWait()
		close(stopped)
	}()
	awaitSignal(t, stopped)
}

func TestBoundedPoolDropFallbackCompletesFutureWithErrTaskDropped(t *testing.T) {
	// one worker, one queue slot
	pool := NewBoundedAsyncPool(1, 1, FallbackDropTask())

	started := make(chan struct{})
	release := make(chan struct{})
	held := Submit(pool, func() (int, error) {
		close(started)
		<-release
		return 1, nil
	})
	awaitSignal(t, started)

	// fills the only queue slot, the held task keeps the only worker busy
	queuedDone := make(chan struct{})
	queued := Submit(pool, func() (int, error) {
		close(queuedDone)
		return 2, nil
	})

	dropped := Submit(pool, func() (int, error) { return 9, nil })
	v, err := dropped.TimedGet(2000)
	if !errors.Is(err, ErrTaskDropped) {
		t.Fatalf("expected ErrTaskDropped, got (%v, %v)", v, err)
	}
	if v != 0 {
		t.Fatalf("expected zero value for a dropped task, got %v", v)
	}

	close(release)
	if _, err := held.Get(); err != nil {
		t.Fatalf("held task should not fail: %v", err)
	}
	// the worker took the queued task, so the queue has room again
	awaitSignal(t, queuedDone)
	if _, err := queued.Get(); err != nil {
		t.Fatalf("queued task should not fail: %v", err)
	}

	ran := Submit(pool, func() (int, error) { return 3, nil })
	v, err = ran.TimedGet(2000)
	if err != nil {
		t.Fatalf("pool should accept work after capacity frees, got err %v", err)
	}
	if v != 3 {
		t.Fatalf("expected 3, got %v", v)
	}
	pool.StopAndWait()
}
