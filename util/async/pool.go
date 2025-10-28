package async

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/curtisnewbie/miso/util/utillog"
	"github.com/panjf2000/ants"
)

var (
	_ AsyncPoolItf = (*AsyncPool)(nil)
	_ AsyncPoolItf = (*AntsAsyncPool)(nil)
)

// Async Pool Interface
type AsyncPoolItf interface {
	Go(f func())
	Stop()
	StopAndWait()
	Run(f func() error) Future[struct{}]
}

// A long live, bounded pool of goroutines.
//
// Use miso.NewAsyncPool to create a new pool.
//
// AsyncPool internally maintains a task queue with limited size and limited number of workers.
//
// By default, if the task queue is full and all workers are busy, the caller of *AsyncPool.Go is blocked indefinitively until the task can be processed.
// You can use [DropTaskWhenPoolFull] or [CallerRunTaskWhenPoolFull] to change this behaviour.
type AsyncPool struct {
	*asyncPoolCommon
	stopped      int32
	stopOnce     *sync.Once
	drainTasksWg *sync.WaitGroup
	tasks        chan func()
	workers      chan struct{}
	idleDur      time.Duration
}

type asyncPoolCommon struct {
	doWhenPoolFull    func(task func())
	blockWhenPoolFull bool
}

func (a *asyncPoolCommon) unwrapPoolCommon() *asyncPoolCommon {
	utillog.DebugLog("Unwrapped *asyncPoolCommon")
	return a
}

type asyncPoolOption func(a AsyncPoolItf)

// Create a bounded pool of goroutines.
//
// The maxTasks determines the capacity of the task queues.
//
// The maxWorkers determines the max number of workers.
//
// By default, if the task queue is full and all workers are busy, the caller of *AsyncPool.Go is blocked indefinitively until the task can be processed.
// You can use [DropTaskWhenPoolFull] or [CallerRunTaskWhenPoolFull] to change this behaviour.
func NewAsyncPool(maxTasks int, maxWorkers int, opts ...asyncPoolOption) *AsyncPool {
	if maxTasks < 0 {
		maxTasks = 0
	}
	if maxWorkers < 1 {
		maxWorkers = 0
	}
	ap := &AsyncPool{
		tasks:           make(chan func(), maxTasks),
		workers:         make(chan struct{}, maxWorkers),
		stopped:         0,
		stopOnce:        &sync.Once{},
		drainTasksWg:    &sync.WaitGroup{},
		idleDur:         5 * time.Minute,
		asyncPoolCommon: &asyncPoolCommon{},
	}
	ap.doWhenPoolFull = func(task func()) {
		// this doesn't do anything, blockWhenPoolFull is true
	}
	ap.blockWhenPoolFull = true

	for _, op := range opts {
		op(ap)
	}
	return ap
}

func unwrapAsyncPoolCommon(a AsyncPoolItf, f func(ap *asyncPoolCommon)) {
	if v, ok := a.(interface {
		unwrapPoolCommon() *asyncPoolCommon
	}); ok {
		f(v.unwrapPoolCommon())
	}
}

func DropTaskWhenPoolFull() asyncPoolOption {
	return func(a AsyncPoolItf) {
		unwrapAsyncPoolCommon(a, func(ap *asyncPoolCommon) {
			ap.doWhenPoolFull = func(task func()) {
				utillog.DebugLog("Pool is full, task dropped")
				// drop task
			}
			ap.blockWhenPoolFull = false
		})
	}
}

func CallerRunTaskWhenPoolFull() asyncPoolOption {
	return func(a AsyncPoolItf) {
		unwrapAsyncPoolCommon(a, func(ap *asyncPoolCommon) {
			ap.doWhenPoolFull = func(task func()) {
				utillog.DebugLog("Pool is full, caller runs task")
				task()
			}
			ap.blockWhenPoolFull = false
		})
	}
}

// Create AsyncPool with number of workers equals to 4 * num_cpu.
func NewCpuAsyncPool() AsyncPoolItf {
	return NewAntsAsyncPool(runtime.NumCPU() * 4)
}

// Create AsyncPool with number of workers equals to 8 * num_cpu and a task queue of size 500.
func NewIOAsyncPool() AsyncPoolItf {
	return NewAsyncPool(500, runtime.NumCPU()*8)
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

func (p *AsyncPool) Run(f func() error) Future[struct{}] {
	return SubmitAsync(p, func() (struct{}, error) { return struct{}{}, f() })
}

// Submit task to the pool.
//
// If the pool is closed, caller will execute the submitted task directly.
func (p *AsyncPool) Go(f func()) {

	if p.isStopped() {
		if p.blockWhenPoolFull {
			f() // caller runs the task
		} else {
			p.doWhenPoolFull(f)
		}
		return
	}

	p.drainTasksWg.Add(1)
	wrp := func() {
		defer p.drainTasksWg.Done()
		f()
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
			return
		}
	} else {
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
		default:
			// when workers are all busy and queue is full
			defer p.drainTasksWg.Done()
			p.doWhenPoolFull(PanicSafeFunc(f))
		}
	}
}

// spawn a new worker.
func (p *AsyncPool) spawn(first func()) {
	defer func() { <-p.workers }()
	utillog.DebugLog("AyncPool created worker")

	if first != nil {
		PanicSafeFunc(first)()
	}

	idleTimer := time.NewTimer(p.idleDur)
	defer idleTimer.Stop()
	for {
		idleTimer.Reset(p.idleDur)
		select {
		case f, ok := <-p.tasks:
			if !ok {
				return
			}
			PanicSafeFunc(f)()
		case <-idleTimer.C:
			utillog.DebugLog("AsyncPool.Worker has been idle for %v, release worker", p.idleDur)
			return
		}
	}
}

type AntsAsyncPool struct {
	*asyncPoolCommon
	p  *ants.Pool
	wg *sync.WaitGroup
}

func (p *AntsAsyncPool) Run(f func() error) Future[struct{}] {
	return SubmitAsync(p, func() (struct{}, error) { return struct{}{}, f() })
}

func (a *AntsAsyncPool) Go(f func()) {
	a.wg.Add(1)
	wrp := func() {
		defer a.wg.Done()
		PanicSafeRun(f)
	}
	err := a.p.Submit(wrp)
	if err != nil {
		utillog.DebugLog("AntsAsyncPool is full or closed, calling fallback, %v", err)
		a.doWhenPoolFull(wrp)
		return
	}
}

func (a *AntsAsyncPool) Stop() {
	a.p.Release()
}

func (a *AntsAsyncPool) StopAndWait() {
	a.p.Release()
	a.wg.Wait()
}

// Create a bounded pool of goroutines.
//
// The maxTasks determines the capacity of the task queues.
//
// The maxWorkers determines the max number of workers.
//
// By default, if the task queue is full and all workers are busy, the caller of [AsyncPoolItf].Go() is blocked indefinitively until the task can be processed.
// You can use [DropTaskWhenPoolFull] or [CallerRunTaskWhenPoolFull] to change this behaviour.
func NewAntsAsyncPool(maxWorkers int, opts ...asyncPoolOption) AsyncPoolItf {
	ap := &AntsAsyncPool{
		asyncPoolCommon: &asyncPoolCommon{},
	}
	ap.doWhenPoolFull = func(task func()) {
		// this doesn't do anything, blockWhenPoolFull is true
	}
	ap.blockWhenPoolFull = true
	ap.wg = &sync.WaitGroup{}

	for _, op := range opts {
		op(ap)
	}
	ap.p, _ = ants.NewPool(maxWorkers,
		ants.WithExpiryDuration(5*time.Minute),
		ants.WithNonblocking(!ap.blockWhenPoolFull))

	return ap
}
