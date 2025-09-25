package task

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/middleware/redis"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/hash"
)

const (
	// default ttl for master lock key in redis (1 min)
	defMstLockTtl    = 1 * time.Minute
	defMstLockTtlSec = 60
)

var (
	staleTaskThreshold = 5 * time.Second
)

func init() {
	// run before SchedulerBootstrap
	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Bootstrap Distributed Task Scheduler",
		Condition: distriTaskBootstrapCondition,
		Bootstrap: distriTaskBootstrap,
		Order:     miso.BootstrapOrderL4,
	})
	miso.BeforeWebRouteRegister(func() error {
		registerRouteForJobTriggers()
		return nil
	})
}

var module = miso.InitAppModuleFunc(func() *taskModule {
	return &taskModule{
		dtaskMut:       &sync.Mutex{},
		workerRegistry: hash.NewStrRWMap[func(miso.Rail) error](),
		group:          "default",
		workerPool:     util.NewAntsAsyncPool(500),
		workerWg:       &sync.WaitGroup{},
	}
})

type taskModule struct {
	// mutex for core proerties (group, nodeId, states, and masterNode election)
	dtaskMut *sync.Mutex

	// schedule group name
	group string

	// identifier for current node
	nodeId string

	// tasks
	dtasks []miso.Job

	// ticker for refreshing master node lock
	refreshMasterTicker *time.Ticker

	// ticker for pulling tasks from task queue
	cancelPullTaskRunner func()

	// task workerRegistry
	workerRegistry *hash.StrRWMap[func(miso.Rail) error]

	// worker pool
	workerPool util.AsyncPoolItf
	workerWg   *sync.WaitGroup
}

func (m *taskModule) enabled() bool {
	return miso.GetPropBool(PropTaskSchedulingEnabled) && len(m.dtasks) > 0
}

func (m *taskModule) prepareTaskScheduling(rail miso.Rail, tasks []miso.Job) error {
	if len(tasks) < 1 {
		return nil
	}
	proposedGroup := miso.GetPropStr(PropTaskSchedulingGroup)
	if proposedGroup != "" {
		m.group = proposedGroup
	}
	m.nodeId = util.ERand(30)

	if err := m.registerTasks(tasks); err != nil {
		return err
	}

	{
		cr, cancel := rail.WithCancel()
		m.cancelPullTaskRunner = cancel
		go func() {
			for {
				select {
				case <-cr.Done():
					return
				default:
					if err := m.pullTasks(miso.EmptyRail()); err != nil {
						miso.Errorf("Pull tasks queue failed, %v", err)
						time.Sleep(time.Millisecond * 500) // backoff from error
					}
				}
			}
		}()
		rail.Infof("Subscribed to distributed task queue: '%v'", m.getTaskQueueKey())
	}

	rail.Infof("Scheduled %d distributed tasks, current node id: '%s', group: '%s'", len(tasks), m.nodeId, m.group)
	return nil
}

// Set the schedule group for current node, by default it's 'default'
func (m *taskModule) setScheduleGroup(groupName string) {
	m.dtaskMut.Lock()
	defer m.dtaskMut.Unlock()

	g := strings.TrimSpace(groupName)
	if g == "" {
		return // still using default group name
	}
	m.group = g
}

// Check if current node is master
func (m *taskModule) isTaskMaster(rail miso.Rail) bool {
	key := m.getTaskMasterKey()
	val, e := redis.GetStr(key)
	if e != nil {
		rail.Errorf("Check is master failed: %v", e)
		return false
	}
	rail.Debugf("Check is master node, key: %v, onRedis: %v, nodeId: %v", key, val, m.nodeId)
	return val == m.nodeId
}

// Get lock key for master node
//
// Applications are grouped together as a cluster (each cluster is differentiated by its appGroup name
// we only try to become the master node in our cluster
func (m *taskModule) getTaskMasterKey() string {
	return "task:master:group:" + m.group
}

func (m *taskModule) getTaskQueueKey() string {
	return "task:master:queue:" + m.group
}

func (m *taskModule) scheduleTasks(tasks ...miso.Job) error {
	for _, t := range tasks {
		if err := m.scheduleTask(t); err != nil {
			return err
		}
	}
	return nil
}

func (m *taskModule) scheduleTask(t miso.Job) error {
	m.dtaskMut.Lock()
	defer m.dtaskMut.Unlock()

	logJobExec := t.LogJobExec
	t.LogJobExec = false

	miso.Infof("Schedule task '%s' cron: '%s'", t.Name, t.Cron)
	actualRun := t.Run

	// worker
	m.workerRegistry.Put(t.Name, func(rail miso.Rail) error {
		if miso.GetPropBool("task.scheduling." + t.Name + ".disabled") {
			rail.Debugf("Task '%v' disabled, skipped", t.Name)
			return nil
		}

		lock := redis.NewRLockf(rail, "dtask:%v:%v", m.group, t.Name)
		if ok, err := lock.TryLock(redis.WithBackoff(time.Second * 1)); err != nil || !ok {
			if err != nil {
				return err
			}

			// happens when producer out runs workers
			rail.Debugf("Task '%v' is running by other nodes, abort", t.Name)
			return nil
		}
		defer lock.Unlock()

		if logJobExec {
			rail.Infof("Running task '%s'", t.Name)
		}

		start := time.Now()
		err := util.PanicSafeRunErr(func() error {
			return actualRun(rail)
		})
		took := time.Since(start)

		if logJobExec {
			rail.Infof("Task '%s' finished, took: %s", t.Name, took)
			miso.LogJobNextRun(rail, t.Name)
		}

		return err
	})

	// producer
	t.Run = func(rail miso.Rail) error {
		if miso.GetPropBool("task.scheduling." + t.Name + ".disabled") {
			rail.Debugf("Task '%v' disabled, skipped", t.Name)
			return nil
		}

		m.dtaskMut.Lock()
		if !m.tryTaskMaster(rail) {
			m.dtaskMut.Unlock()
			rail.Debugf("Not master node, skip scheduled task '%v'", t.Name)
			return nil
		}
		m.dtaskMut.Unlock()

		// produce task
		return m.produceTask(rail, t.Name)
	}
	m.dtasks = append(m.dtasks, t)
	return nil
}

type queuedTask struct {
	Name        string
	ScheduledAt util.Time
}

func (m *taskModule) produceTask(rail miso.Rail, name string) error {
	qt := queuedTask{
		Name:        name,
		ScheduledAt: util.NowUTC(),
	}
	return redis.LPushJson(rail, m.getTaskQueueKey(), qt)
}

func (m *taskModule) pullTasks(rail miso.Rail) error {

	v, ok, err := redis.BRPopJson[queuedTask](rail, time.Second*10, m.getTaskQueueKey())
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	for _, qt := range v {
		if qt.ScheduledAt.Before(util.NowUTC().Add(-staleTaskThreshold)) {
			rail.Warnf("Task was triggered 5s ago, ignore, %v, scheduledAt: %v", qt.Name, qt.ScheduledAt)
			continue
		}

		if err := m.triggerWorker(miso.EmptyRail(), qt.Name); err != nil {
			rail.Errorf("Failed to trigger worker, task: '%v'", qt.Name)
		}
	}
	return nil
}

func (m *taskModule) triggerWorker(rail miso.Rail, name string) error {
	f, ok := m.workerRegistry.Get(name)
	if !ok {
		return errs.NewErrf("Task no found")
	}
	m.workerWg.Add(1)
	m.workerPool.Go(func() {
		defer m.workerWg.Done()
		rail = rail.NewCtx()
		err := f(rail)
		if err != nil {
			rail.Errorf("Failed to run task '%v'", name)
		}
	})
	return nil
}

func (m *taskModule) startTaskSchedulerAsync(rail miso.Rail) error {
	m.dtaskMut.Lock()
	defer m.dtaskMut.Unlock()

	if len(m.dtasks) < 1 {
		return nil
	}
	if err := m.prepareTaskScheduling(rail, m.dtasks); err != nil {
		return err
	}
	miso.StartSchedulerAsync()
	return nil
}

// Start distributed scheduler, current routine is blocked
func (m *taskModule) startTaskSchedulerBlocking(rail miso.Rail) error {
	m.dtaskMut.Lock()
	defer m.dtaskMut.Unlock()
	if len(m.dtasks) < 1 {
		return nil
	}
	if err := m.prepareTaskScheduling(rail, m.dtasks); err != nil {
		return err
	}
	miso.StartSchedulerBlocking()
	return nil
}

func (m *taskModule) bootstrapAsComponent(rail miso.Rail) error {
	m.dtaskMut.Lock()
	defer m.dtaskMut.Unlock()
	if err := m.prepareTaskScheduling(rail, m.dtasks); err != nil {
		return fmt.Errorf("failed to prepareTaskScheduling, %w", err)
	}
	return nil
}

func (m *taskModule) stop() {
	rail := miso.EmptyRail()
	m.dtaskMut.Lock()
	defer m.dtaskMut.Unlock()

	miso.StopScheduler()
	m.releaseMaster(rail)
	if m.cancelPullTaskRunner != nil {
		m.cancelPullTaskRunner()
	}
	m.workerWg.Wait()
}

// Start refreshing master lock ticker
func (m *taskModule) startTaskMasterLockTicker() {
	if m.refreshMasterTicker != nil {
		return
	}
	tk := time.NewTicker(5 * time.Second)
	m.refreshMasterTicker = tk
	go func(c <-chan time.Time) {
		for {
			rail := miso.EmptyRail()
			_ = m.refreshTaskMasterLock(rail)
			<-c
		}
	}(tk.C)
}

func (m *taskModule) releaseMasterNodeLock() {
	v, err := redis.Eval(miso.EmptyRail(), `
	if (redis.call('EXISTS', KEYS[1]) == 0) then
		return 0;
	end;

	if (redis.call('GET', KEYS[1]) == tostring(ARGV[1])) then
		redis.call('DEL', KEYS[1])
		return 1;
	end;
	return 0;`, []string{m.getTaskMasterKey()}, m.nodeId)
	if err != nil {
		miso.Errorf("Failed to release master node lock, %v", err)
		return
	}
	miso.Debugf("Release master node lock, released? %v", v == int64(1))
}

// Stop refreshing master lock ticker
func (m *taskModule) stopTaskMasterLockTicker() {
	if m.refreshMasterTicker == nil {
		return
	}

	m.refreshMasterTicker.Stop()
	m.refreshMasterTicker = nil
}

// Refresh master lock key ttl
func (m *taskModule) refreshTaskMasterLock(rail miso.Rail) error {
	_, err := redis.Eval(rail, `
	if (redis.call('GET', KEYS[1]) == tostring(ARGV[1])) then
		redis.call('EXPIRE', KEYS[1], tonumber(ARGV[2]))
		return 1;
	end;
	return 0;
	`, []string{m.getTaskMasterKey()}, m.nodeId, defMstLockTtlSec)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			rail.Debugf("Failed to refreshTaskMasterLock, masterLock not obtained")
			return nil
		}
		rail.Errorf("Failed to refreshTaskMasterLock, %v", err)
	} else {
		rail.Debugf("Did refreshTaskMasterLock")
	}
	return err
}

// Try to become master node
func (m *taskModule) tryTaskMaster(rail miso.Rail) bool {
	if m.isTaskMaster(rail) {
		return true
	}

	ok, err := redis.SetNX(rail, m.getTaskMasterKey(), m.nodeId, defMstLockTtl)
	if err != nil {
		rail.Errorf("Try to become master node %v", errs.WrapErr(err))
		return false
	}

	isMaster := ok
	if isMaster {
		rail.Infof("Elected to be the master node for group: '%s'", m.group)
		m.startTaskMasterLockTicker()
	} else {
		m.stopTaskMasterLockTicker()
	}
	return isMaster
}

func (m *taskModule) releaseMaster(rail miso.Rail) {
	if m.isTaskMaster(rail) {
		rail.Infof("Releasing master node")
		m.stopTaskMasterLockTicker()
		m.releaseMasterNodeLock()
	}
}

func (m *taskModule) registerTasks(tasks []miso.Job) error {
	if len(tasks) < 1 {
		return nil
	}
	for _, d := range tasks {
		if err := miso.ScheduleCron(d); err != nil {
			errs.WrapErrf(err, "failed to schedule cron job, %+v", d)
		}
	}
	return nil
}

// Set the schedule group for current node, by default it's 'default'
func SetScheduleGroup(groupName string) {
	module().setScheduleGroup(groupName)
}

// Schedule a named distributed task
//
// Applications are grouped together as a cluster (each cluster is differentiated by its group name),
// only the master node can run the Scheduled tasks.
//
// Tasks are pending until StartTaskSchedulerAsync() is called.
//
// E.g.,
//
//	job := miso.Job{
//		Name:            "Very important task",
//		Cron:            "0/1 * * * * ?",
//		CronWithSeconds: true,
//		Run: MyTask,
//	}
//	ScheduleDistributedTask(job)
func ScheduleDistributedTask(t ...miso.Job) error {
	return module().scheduleTasks(t...)
}

// Start distributed scheduler asynchronously
func StartTaskSchedulerAsync(rail miso.Rail) error {
	return module().startTaskSchedulerAsync(rail)
}

// Start distributed scheduler, current routine is blocked
func StartTaskSchedulerBlocking(rail miso.Rail) error {
	return module().startTaskSchedulerBlocking(rail)
}

// Shutdown distributed job scheduling
func StopTaskScheduler() {
	module().stop()
}

func distriTaskBootstrapCondition(rail miso.Rail) (bool, error) {
	return module().enabled(), nil
}

func distriTaskBootstrap(rail miso.Rail) error {
	m := module()
	_ = redis.GetRedis() // check if redis is initialized
	miso.AddOrderedShutdownHook(miso.DefShutdownOrder-1, func() { m.stop() })
	return m.bootstrapAsComponent(rail)
}

// enable api to manually trigger tasks
func registerRouteForJobTriggers() {
	if !miso.GetPropBool(PropTaskSchedulingApiTriggerJobEnabled) {
		return
	}

	miso.HttpGet("/debug/task/trigger", miso.RawHandler(func(inb *miso.Inbound) {
		rail := inb.Rail()
		name := inb.Query("name")
		err := module().triggerWorker(rail, name)
		inb.HandleResult(nil, err)
	})).DocQueryParam("name", "job name").Desc("Manually Trigger Task By Name")
}
