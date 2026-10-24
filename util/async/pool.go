package async

import (
	"runtime"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/util/ptr"
	"github.com/curtisnewbie/miso/util/utillog"
	"github.com/panjf2000/ants"
)

var (
	_ AsyncPool = (*BoundedAsyncPool)(nil)
	_ AsyncPool = (*AntsAsyncPool)(nil)

	_ taskAcceptor = (*BoundedAsyncPool)(nil)
	_ taskAcceptor = (*AntsAsyncPool)(nil)
)

const (
	idleDur = 1 * time.Minute
)

// Async Pool Interface
type AsyncPool interface {
	Go(f func())
	Stop()
	StopAndWait()
	Run(f func() error) Future[struct{}]
}

// A long live, bounded pool of goroutines.
//
// Use [NewBoundedAsyncPool] to create a new pool.
//
// BoundedAsyncPool internally maintains a task queue with limited size and limited number of workers.
//
// By default, if the task queue is full and all workers are busy, the caller of *BoundedAsyncPool.Go is blocked indefinitively until the task can be processed.
// You can use [FallbackDropTask] or [FallbackCallerRun] to change this behaviour. A task refused by a
// fallback that discards work completes its [Future] with [ErrTaskDropped].
//
// Stopping blocks new queue entries; the tasks already accepted drain in the background, and
// [BoundedAsyncPool.StopAndWait] returns once they finish. A task submitted after stop runs inline
// when the pool blocks on a full queue, and goes through the configured fallback otherwise.
type BoundedAsyncPool struct {
	*asyncPoolCommon

	// stateMu guards stopped. Stop takes it for writing, every submission takes
	// it for reading, so wg.Add happens either before Stop starts or is skipped
	// by the stopped check. Dispatch happens outside the lock; the stop channel
	// case in goTask plus the inflight barrier keep the stop contract.
	stateMu sync.RWMutex
	stopped bool
	stopCh  chan struct{}

	// inflight counts producers that are dispatching a task. Workers wait for it
	// before draining on stop, so an accepted task cannot be enqueued after the
	// drain has finished.
	inflight sync.WaitGroup

	tasks   chan func()
	workers chan struct{}
	idleDur time.Duration
}

type asyncPoolCommon struct {
	doWhenPoolFull    func(task func()) bool
	blockWhenPoolFull bool
	wg                *sync.WaitGroup
}

type asyncPoolOptions struct {
	doWhenPoolFull    func(task func()) bool
	blockWhenPoolFull *bool
	wg                *sync.WaitGroup

	taskQueueSize *int // used by [NewAsyncPool] only.
}

func withAsyncPoolOptions(a *asyncPoolOptions) asyncPoolOption {
	return func(cp *asyncPoolOptions) {
		cp.blockWhenPoolFull = a.blockWhenPoolFull
		cp.doWhenPoolFull = a.doWhenPoolFull
		cp.taskQueueSize = a.taskQueueSize
		cp.wg = a.wg
	}
}

type asyncPoolOption func(a *asyncPoolOptions)

// Create a bounded pool of goroutines.
//
// The maxTasks determines the capacity of the task queues.
//
// The maxWorkers determines the max number of workers.
//
// By default, if the task queue is full and all workers are busy, the caller of *AsyncPool.Go is blocked indefinitively until the task can be processed.
// You can use [FallbackDropTask] or [FallbackCallerRun] to change this behaviour. A task refused by a
// fallback that discards work completes its [Future] with [ErrTaskDropped].
//
// Stopping blocks new queue entries; the tasks already accepted drain in the background, and
// [BoundedAsyncPool.StopAndWait] returns once they finish. A task submitted after stop runs inline
// when the pool blocks on a full queue, and goes through the configured fallback otherwise.
//
// Deprecated: Since v0.3.10, migrate to [NewAsyncPool] if possible.
func NewBoundedAsyncPool(maxTasks int, maxWorkers int, opts ...asyncPoolOption) *BoundedAsyncPool {
	if maxTasks < 0 {
		maxTasks = 0
	}
	if maxWorkers < 1 {
		maxWorkers = 0
	}
	ap := &BoundedAsyncPool{
		tasks:   make(chan func(), maxTasks),
		workers: make(chan struct{}, maxWorkers),
		stopCh:  make(chan struct{}),
		idleDur: idleDur,
		asyncPoolCommon: &asyncPoolCommon{
			doWhenPoolFull:    callerRunFallback,
			blockWhenPoolFull: true,
		},
	}

	if len(opts) > 0 {
		ops := &asyncPoolOptions{}
		for _, op := range opts {
			op(ops)
		}
		if ops.blockWhenPoolFull != nil {
			ap.blockWhenPoolFull = *ops.blockWhenPoolFull
		}
		if ops.doWhenPoolFull != nil {
			ap.doWhenPoolFull = ops.doWhenPoolFull
		}
		if ops.wg != nil {
			ap.wg = ops.wg
		} else {
			ap.wg = &sync.WaitGroup{}
		}
	} else {
		ap.wg = &sync.WaitGroup{}
	}
	return ap
}

// default fallback: run the task on the caller instead of discarding it.
func callerRunFallback(task func()) bool {
	utillog.DebugLog("Pool is full, caller runs task")
	task()
	return true
}

// Create new Goroutine to run the task when pool is full.
func FallbackNewGoroutine() asyncPoolOption {
	return func(ap *asyncPoolOptions) {
		ap.doWhenPoolFull = func(task func()) bool {
			utillog.DebugLog("Pool is full, run in new Goroutine")
			go task()
			return true
		}
		ap.blockWhenPoolFull = ptr.BoolPtr(false)
	}
}

// Drop task when pool is full.
//
// The task's Future completes with [ErrTaskDropped].
func FallbackDropTask() asyncPoolOption {
	return func(ap *asyncPoolOptions) {
		ap.doWhenPoolFull = func(task func()) bool {
			utillog.WarnLog("Pool is full, task dropped")
			return false
		}
		ap.blockWhenPoolFull = ptr.BoolPtr(false)
	}
}

// Fallback to caller runs when pool is full.
func FallbackCallerRun() asyncPoolOption {
	return func(ap *asyncPoolOptions) {
		ap.doWhenPoolFull = callerRunFallback
		ap.blockWhenPoolFull = ptr.BoolPtr(false)
	}
}

// With task queue
func WithTaskQueue(queueSize int) asyncPoolOption {
	return func(ap *asyncPoolOptions) {
		ap.taskQueueSize = &queueSize
	}
}

// Return runtime.GOMAXPROCS(0)
func MaxProcs() int {
	return runtime.GOMAXPROCS(0)
}

// Return multi * GOMAXPROCS or min whichever is greater, if multi * GOMAXPROCS > max, max is returned instead.
//
// If max is unlimited, use -1 as placeholder.
func CalcPoolSize(multi int, min int, max int) int {
	n := multi * MaxProcs()
	if n < min {
		return min
	}
	if max > 0 && n > max {
		return max
	}
	return n
}

// Create AsyncPool with number of workers equals to 4 * GOMAXPROCS.
//
// Deprecated: Since v0.3.10. use [NewAsyncPool] instead.
func NewCpuAsyncPool() AsyncPool {
	return NewAntsAsyncPool(CalcPoolSize(4, 8, -1))
}

// Create AsyncPool with number of workers equals to 8 * GOMAXPROCS and a task queue of size 100.
//
// Deprecated: Since v0.3.10. use [NewAsyncPool] instead.
func NewIOAsyncPool() AsyncPool {
	return NewBoundedAsyncPool(100, CalcPoolSize(8, 16, -1))
}

// Stop the pool.
//
// Stopping blocks new submissions and closes the stop signal. The tasks already
// accepted drain in the background; use [BoundedAsyncPool.StopAndWait] to wait for
// them. Tasks submitted after stop run inline when the pool blocks on a full queue,
// and go through the configured fallback otherwise.
func (p *BoundedAsyncPool) Stop() {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	if p.stopped {
		return
	}
	p.stopped = true
	close(p.stopCh)
}

// Stop the pool and return after the tasks it already accepted finished.
func (p *BoundedAsyncPool) StopAndWait() {
	p.Stop()
	p.wg.Wait()
}

// Run task on the pool.
//
// Returns false if the task was dropped, so that the task's Future can be
// completed with [ErrTaskDropped]. A dropped task is never run.
func (p *BoundedAsyncPool) goTask(f func()) bool {
	p.stateMu.RLock()

	if p.stopped {
		blockWhenPoolFull := p.blockWhenPoolFull
		p.stateMu.RUnlock()

		// post-stop submissions are outside the stop barrier: they are not
		// counted by wg, incrementing it after StopAndWait may race Wait at
		// counter zero.
		if blockWhenPoolFull {
			PanicSafeFunc(f)()
			return true
		}
		return p.doWhenPoolFull(PanicSafeFunc(f))
	}

	p.wg.Add(1)
	p.inflight.Add(1)
	stopCh := p.stopCh
	p.stateMu.RUnlock()

	// no lock is held from here on: dispatching may block on a full pool, and
	// the fallback may run arbitrary user code
	defer p.inflight.Done()

	wrp := func() {
		defer p.wg.Done()
		PanicSafeFunc(f)()
	}

	if p.blockWhenPoolFull {
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
		case <-stopCh:
			// stopped while dispatching, run the task inline so it stays counted
			wrp()
		}
		return true
	}

	select {
	case p.workers <- struct{}{}:
		go p.spawn(wrp)
		return true
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
		return true
	case <-stopCh:
		// stopped while dispatching: the fallback runs without any lock held
		ran := p.doWhenPoolFull(wrp)
		if !ran {
			p.wg.Done()
		}
		return ran
	default:
		// when workers are all busy and queue is full
		ran := p.doWhenPoolFull(wrp)
		if !ran {
			p.wg.Done()
		}
		return ran
	}
}

func (p *BoundedAsyncPool) Run(f func() error) Future[struct{}] {
	return Submit(p, func() (struct{}, error) { return struct{}{}, f() })
}

// Submit task to the pool.
//
// If the pool is stopped, the task runs inline when the pool blocks on a full
// queue, and goes through the configured fallback otherwise. With
// [FallbackDropTask] it is dropped.
func (p *BoundedAsyncPool) Go(f func()) {
	p.goTask(f)
}

// spawn a new worker.
func (p *BoundedAsyncPool) spawn(first func()) {
	released := false
	defer func() {
		if !released {
			<-p.workers
		}
	}()
	utillog.DebugLog("AyncPool created worker")

	if first != nil {
		first()
	}

	idleTimer := time.NewTimer(p.idleDur)
	defer idleTimer.Stop()
	for {
		idleTimer.Reset(p.idleDur)
		select {
		case <-p.stopCh:
			// wait for producers that are still dispatching, so nothing can be
			// enqueued after the drain
			p.inflight.Wait()
			p.drainTasks()
			return
		case f := <-p.tasks:
			f()
		case <-idleTimer.C:
			utillog.DebugLog("AsyncPool.Worker has been idle for %v, release worker", p.idleDur)
			select {
			case <-p.stopCh:
				p.inflight.Wait()
			default:
				// idle exit that is not stopping must not wait for producers
			}
			p.drainTasks()

			// Invariant: a worker holds a workers token exactly while a live worker
			// goroutine reads p.tasks. Release the token before the final queue
			// check, so a producer that enqueues in this window finds the slot free
			// and spawns a replacement through the extra-spawn path.
			<-p.workers
			released = true

			if len(p.tasks) > 0 {
				select {
				case p.workers <- struct{}{}:
					released = false
					continue
				default:
					return
				}
			}
			return
		}
	}
}

// drainTasks runs the queued tasks without blocking.
func (p *BoundedAsyncPool) drainTasks() {
	for {
		select {
		case f := <-p.tasks:
			f()
		default:
			return
		}
	}
}

// A bounded pool of goroutines backed by ants.
//
// Stopping blocks new submissions and releases the ants pool; work already accepted
// drains in the background and [AntsAsyncPool.StopAndWait] returns once it finishes.
// A task submitted after stop runs inline when the pool blocks on a full queue, and
// goes through the configured fallback otherwise.
type AntsAsyncPool struct {
	*asyncPoolCommon

	// stateMu guards stopped, mirroring BoundedAsyncPool.
	stateMu sync.RWMutex
	stopped bool

	p *ants.Pool
}

func (p *AntsAsyncPool) Run(f func() error) Future[struct{}] {
	return Submit(p, func() (struct{}, error) { return struct{}{}, f() })
}

// Run task on the pool.
//
// Returns false if the task was dropped, so that the task's Future can be
// completed with [ErrTaskDropped]. A dropped task is never run.
func (a *AntsAsyncPool) goTask(f func()) bool {
	a.stateMu.RLock()

	if a.stopped {
		blockWhenPoolFull := a.blockWhenPoolFull
		a.stateMu.RUnlock()

		// post-stop submissions are outside the stop barrier: they are not
		// counted by wg, incrementing it after StopAndWait may race Wait at
		// counter zero.
		if blockWhenPoolFull {
			PanicSafeFunc(f)()
			return true
		}
		return a.doWhenPoolFull(PanicSafeFunc(f))
	}

	a.wg.Add(1)
	a.stateMu.RUnlock()

	wrp := func() {
		defer a.wg.Done()
		PanicSafeFunc(f)()
	}

	// no lock is held from here on: ants may block, and the fallback may run
	// arbitrary user code
	err := a.p.Submit(wrp)
	if err != nil {
		utillog.DebugLog("AntsAsyncPool is full or closed, calling fallback, %v", err)
		if !a.doWhenPoolFull(wrp) {
			a.wg.Done()
			return false
		}
	}
	return true
}

func (a *AntsAsyncPool) Go(f func()) {
	a.goTask(f)
}

// Stop the pool.
//
// Stopping blocks new submissions and releases the ants pool; work already accepted
// drains in the background, and [AntsAsyncPool.StopAndWait] returns once it finishes.
// A task submitted after stop runs inline when the pool blocks on a full queue, and
// goes through the configured fallback otherwise.
func (a *AntsAsyncPool) Stop() {
	a.stateMu.Lock()
	if a.stopped {
		a.stateMu.Unlock()
		return
	}
	a.stopped = true
	a.stateMu.Unlock()
	a.p.Release()
}

// Stop the pool and return after the tasks it already accepted finished.
func (a *AntsAsyncPool) StopAndWait() {
	a.Stop()
	a.wg.Wait()
}

// Create a bounded pool of goroutines without extra task queue.
//
// The maxWorkers determines the max number of workers.
//
// [AntsAsyncPool] does not maintain task queue. By default, if all workers are busy, the caller of [AsyncPool.Go] is blocked
// indefinitively until the task can be processed.
//
// You can use [FallbackDropTask] or [FallbackCallerRun] to change this behaviour.
//
// [AntsAsyncPool] is good for cases where you want back pressure, i.e., stop producing tasks when all workers are busy.
//
// In cases where you want to have an extra queue of tasks that do not always block task producers, use [NewBoundedAsyncPool] intead.
//
// Deprecated: Since v0.3.10, migrate to [NewAsyncPool] if possible.
func NewAntsAsyncPool(maxWorkers int, opts ...asyncPoolOption) *AntsAsyncPool {
	ap := &AntsAsyncPool{
		asyncPoolCommon: &asyncPoolCommon{
			doWhenPoolFull:    callerRunFallback,
			blockWhenPoolFull: true,
		},
	}

	if len(opts) > 0 {
		ops := &asyncPoolOptions{}
		for _, op := range opts {
			op(ops)
		}
		if ops.blockWhenPoolFull != nil {
			ap.blockWhenPoolFull = *ops.blockWhenPoolFull
		}
		if ops.doWhenPoolFull != nil {
			ap.doWhenPoolFull = ops.doWhenPoolFull
		}
		if ops.wg != nil {
			ap.wg = ops.wg
		} else {
			ap.wg = &sync.WaitGroup{}
		}
	} else {
		ap.wg = &sync.WaitGroup{}
	}
	ap.p, _ = ants.NewPool(maxWorkers,
		ants.WithExpiryDuration(idleDur),
		ants.WithNonblocking(!ap.blockWhenPoolFull))

	return ap
}

// Create a bounded pool of goroutines.
//
// By default, the created [AsyncPool] does not maintain an extra task queue. If all workers are busy, the caller
// of [AsyncPool.Go] is blocked indefinitively until a worker is free.
//
// This is good for cases where you want back pressure, i.e., stop producing tasks when all workers are busy.
//
// In cases where you want an extra task queue, e.g., so that the task producers won't block so frequently when pool is
// exhausted, use [WithTaskQueue] to specify the task queue size. Decide whether you need a task queue based on your use case.
//
// You can also use [FallbackDropTask] or [FallbackCallerRun] to customize fallback behaviour. A task
// refused by a fallback that discards work completes its [Future] with [ErrTaskDropped].
//
// Stopping blocks new submissions; work already accepted drains in the background, and
// [AsyncPool.StopAndWait] returns once it finishes. A task submitted after stop runs inline when the
// pool blocks on a full queue, and goes through the configured fallback otherwise.
//
// Find proper worker pool size based on N * GOMAXPROCS, e.g., in Redis connection pool, N might be 10; in web server connection pool, N can be 258 and so on.
//
// When the tasks are CPU intensive, N should be relatively small, e.g., N=1 or N=2.
//
// See [CalcPoolSize].
func NewAsyncPool(maxWorkers int, opts ...asyncPoolOption) AsyncPool {
	ops := &asyncPoolOptions{}
	for _, op := range opts {
		op(ops)
	}
	var p AsyncPool
	if ops.taskQueueSize != nil && *ops.taskQueueSize > -1 {
		p = NewBoundedAsyncPool(*ops.taskQueueSize, maxWorkers, withAsyncPoolOptions(ops))
	} else {
		p = NewAntsAsyncPool(maxWorkers, withAsyncPoolOptions(ops))
	}
	return p
}
