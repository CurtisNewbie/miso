package async

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/curtisnewbie/miso/util/ptr"
	"github.com/curtisnewbie/miso/util/utillog"
	"github.com/panjf2000/ants"
)

var (
	_ AsyncPool = (*BoundedAsyncPool)(nil)
	_ AsyncPool = (*AntsAsyncPool)(nil)
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
// You can use [FallbackDropTask] or [FallbackCallerRun] to change this behaviour.
type BoundedAsyncPool struct {
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

type asyncPoolOptions struct {
	doWhenPoolFull    func(task func())
	blockWhenPoolFull *bool

	taskQueueSize *int // used by [NewAsyncPool] only.
}

func withAsyncPoolOptions(a *asyncPoolOptions) asyncPoolOption {
	return func(cp *asyncPoolOptions) {
		cp.blockWhenPoolFull = a.blockWhenPoolFull
		cp.doWhenPoolFull = a.doWhenPoolFull
		cp.taskQueueSize = a.taskQueueSize
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
// You can use [FallbackDropTask] or [FallbackCallerRun] to change this behaviour.
//
// Since v0.3.10, migrate to [NewAsyncPool] if possible.
func NewBoundedAsyncPool(maxTasks int, maxWorkers int, opts ...asyncPoolOption) *BoundedAsyncPool {
	if maxTasks < 0 {
		maxTasks = 0
	}
	if maxWorkers < 1 {
		maxWorkers = 0
	}
	ap := &BoundedAsyncPool{
		tasks:        make(chan func(), maxTasks),
		workers:      make(chan struct{}, maxWorkers),
		stopped:      0,
		stopOnce:     &sync.Once{},
		drainTasksWg: &sync.WaitGroup{},
		idleDur:      idleDur,
		asyncPoolCommon: &asyncPoolCommon{
			doWhenPoolFull:    func(task func()) {},
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
	}
	return ap
}

// Drop task when pool is full.
func FallbackDropTask() asyncPoolOption {
	return func(ap *asyncPoolOptions) {
		ap.doWhenPoolFull = func(task func()) {
			utillog.DebugLog("Pool is full, task dropped")
			// drop task
		}
		ap.blockWhenPoolFull = ptr.BoolPtr(false)
	}
}

// Fallback to caller runs when pool is full.
func FallbackCallerRun() asyncPoolOption {
	return func(ap *asyncPoolOptions) {
		ap.doWhenPoolFull = func(task func()) {
			utillog.DebugLog("Pool is full, caller runs task")
			task()
		}
		ap.blockWhenPoolFull = ptr.BoolPtr(false)
	}
}

// With task queue
func WithTaskQueue(queueSize int) asyncPoolOption {
	return func(ap *asyncPoolOptions) {
		ap.taskQueueSize = &queueSize
	}
}

func MaxProcs() int {
	return runtime.GOMAXPROCS(0)
}

// Return multi * GOMAXPROCS or min whichever is greater.
func CalcPoolSize(multi int, min int) int {
	if min < 1 {
		min = 1
	}
	n := multi * MaxProcs()
	if n < min {
		return min
	}
	return n
}

// Create AsyncPool with number of workers equals to 4 * GOMAXPROCS.
//
// Deprecated: Since v0.3.10. Do not use this.
// Pick [NewAntsAsyncPool] or [NewBoundedAsyncPool] based on your use case.
// Find proper worker pool size based on N * GOMAXPROCS, e.g., in Redis connection pool, N is 10, in web server connection pool, N can be 258.
func NewCpuAsyncPool() AsyncPool {
	return NewAntsAsyncPool(MaxProcs() * 4)
}

// Create AsyncPool with number of workers equals to 8 * GOMAXPROCS and a task queue of size 100.
//
// Deprecated: Since v0.3.10. Do not use this.
// Pick [NewAntsAsyncPool] or [NewBoundedAsyncPool] based on your use case.
// Find proper worker pool size based on N * GOMAXPROCS, e.g., in Redis connection pool, N is 10, in web server connection pool, N can be 258.
func NewIOAsyncPool() AsyncPool {
	return NewBoundedAsyncPool(100, MaxProcs()*8)
}

// Stop the pool.
//
// Once the pool is stopped, new tasks submitted are executed directly by the caller.
func (p *BoundedAsyncPool) Stop() {
	if atomic.CompareAndSwapInt32(&p.stopped, 0, 1) {
		p.stopOnce.Do(func() { close(p.tasks) })
	}
}

// Stop the pool and wait until existing workers drain all the remaining tasks.
//
// Once the pool is stopped, new tasks submitted are executed directly by the caller.
func (p *BoundedAsyncPool) StopAndWait() {
	p.Stop()
	p.drainTasksWg.Wait()
}

func (p *BoundedAsyncPool) isStopped() bool {
	return atomic.LoadInt32(&p.stopped) == 1
}

func (p *BoundedAsyncPool) Run(f func() error) Future[struct{}] {
	return Submit(p, func() (struct{}, error) { return struct{}{}, f() })
}

// Submit task to the pool.
//
// If the pool is closed, caller will execute the submitted task directly.
func (p *BoundedAsyncPool) Go(f func()) {

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
func (p *BoundedAsyncPool) spawn(first func()) {
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
	return Submit(p, func() (struct{}, error) { return struct{}{}, f() })
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
// Since v0.3.10, migrate to [NewAsyncPool] if possible.
func NewAntsAsyncPool(maxWorkers int, opts ...asyncPoolOption) *AntsAsyncPool {
	ap := &AntsAsyncPool{
		asyncPoolCommon: &asyncPoolCommon{
			doWhenPoolFull:    func(task func()) {},
			blockWhenPoolFull: true,
		},
		wg: &sync.WaitGroup{},
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
// exhausted, use [WithTaskQueue] to specify the task queue size.
//
// You can also use [FallbackDropTask] or [FallbackCallerRun] to change default behaviours.
//
// Decide whether you need a task queue based on your use case.
//
// Find proper worker pool size based on N * GOMAXPROCS, e.g., in Redis connection pool, N might be 10; in web server connection pool, N can be 258 and so on.
//
// When the tasks are CPU intensive, N should be relatively small, e.g., N=1 or N=2.
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
