package task

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/middleware/redis"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
)

const (
	// default ttl for master lock key in redis (1 min)
	defMstLockTtl = 1 * time.Minute

	moduleKey = "_miso:internal:task:module"
)

func init() {
	miso.SetDefProp(PropTaskSchedulingEnabled, true)
	miso.SetDefProp(PropTaskSchedulingGroup, "${app.name}")

	// run before SchedulerBootstrap
	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Bootstrap Distributed Task Scheduler",
		Condition: distriTaskBootstrapCondition,
		Bootstrap: distriTaskBootstrap,
		Order:     miso.BootstrapOrderL4,
	})
}

//lint:ignore U1000 for future use
var appModule, module = miso.InitAppModuleFunc(moduleKey, func(app *miso.MisoApp) *taskModule {
	return &taskModule{
		dtaskMut: &sync.Mutex{},
		group:    "default",
		conf:     app.Config(),
		app:      app,
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
	masterTicker *time.Ticker

	app  *miso.MisoApp
	conf *miso.AppConfig
}

func (m *taskModule) enabled() bool {
	return m.conf.GetPropBool(PropTaskSchedulingEnabled) && len(m.dtasks) > 0
}

func (m *taskModule) prepareTaskScheduling(rail miso.Rail, tasks []miso.Job) error {
	if len(tasks) < 1 {
		return nil
	}
	proposedGroup := m.conf.GetPropStr(PropTaskSchedulingGroup)
	if proposedGroup != "" {
		m.group = proposedGroup
	}
	m.nodeId = util.ERand(30)

	if err := m.registerTasks(tasks); err != nil {
		return err
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
		rail.Errorf("check is master failed: %v", e)
		return false
	}
	rail.Debugf("check is master node, key: %v, onRedis: %v, nodeId: %v", key, val, m.nodeId)
	return val == m.nodeId
}

// Get lock key for master node
//
// Applications are grouped together as a cluster (each cluster is differentiated by its appGroup name
// we only try to become the master node in our cluster
func (m *taskModule) getTaskMasterKey() string {
	return "task:master:group:" + m.group
}

func (m *taskModule) scheduleDistributedTask(t miso.Job) error {
	m.dtaskMut.Lock()
	defer m.dtaskMut.Unlock()

	miso.Infof("Schedule distributed task '%s' cron: '%s'", t.Name, t.Cron)
	actualRun := t.Run
	t.Run = func(rail miso.Rail) error {
		m.dtaskMut.Lock()
		if !m.tryTaskMaster(rail) {
			m.dtaskMut.Unlock()
			rail.Debug("Not master node, skip scheduled task")
			return nil
		}
		m.dtaskMut.Unlock()
		return actualRun(rail)
	}
	m.dtasks = append(m.dtasks, t)
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
	m.tryTaskMaster(rail)
	return nil
}

func (m *taskModule) stop() {
	m.dtaskMut.Lock()
	defer m.dtaskMut.Unlock()

	miso.StopScheduler()
	m.stopTaskMasterLockTicker()
	m.releaseMasterNodeLock()
}

// Start refreshing master lock ticker
func (m *taskModule) startTaskMasterLockTicker() {
	if m.masterTicker != nil {
		return
	}
	m.masterTicker = time.NewTicker(5 * time.Second)
	go func(c <-chan time.Time) {
		for {
			m.refreshTaskMasterLock()
			<-c
		}
	}(m.masterTicker.C)
}

func (m *taskModule) releaseMasterNodeLock() {
	cmd := redis.GetAppRedis(m.app).Eval(`
	if (redis.call('EXISTS', KEYS[1]) == 0) then
		return 0;
	end;

	if (redis.call('GET', KEYS[1]) == tostring(ARGV[1])) then
		redis.call('DEL', KEYS[1])
		return 1;
	end;
	return 0;`, []string{m.getTaskMasterKey()}, m.nodeId)
	if cmd.Err() != nil {
		miso.Errorf("Failed to release master node lock, %v", cmd.Err())
		return
	}
	miso.Debugf("Release master node lock, released? %v", cmd.Val())
}

// Stop refreshing master lock ticker
func (m *taskModule) stopTaskMasterLockTicker() {
	if m.masterTicker == nil {
		return
	}

	m.masterTicker.Stop()
	m.masterTicker = nil
}

// Refresh master lock key ttl
func (m *taskModule) refreshTaskMasterLock() error {
	return redis.GetAppRedis(m.app).Expire(m.getTaskMasterKey(), defMstLockTtl).Err()
}

// Try to become master node
func (m *taskModule) tryTaskMaster(rail miso.Rail) bool {
	if m.isTaskMaster(rail) {
		return true
	}

	bcmd := redis.GetAppRedis(m.app).SetNX(m.getTaskMasterKey(), m.nodeId, defMstLockTtl)
	if bcmd.Err() != nil {
		rail.Errorf("try to become master node: '%v'", bcmd.Err())
		return false
	}

	isMaster := bcmd.Val()
	if isMaster {
		rail.Infof("Elected to be the master node for group: '%s'", m.group)
		m.startTaskMasterLockTicker()
	} else {
		m.stopTaskMasterLockTicker()
	}
	return isMaster
}

func (m *taskModule) registerTasks(tasks []miso.Job) error {
	if len(tasks) < 1 {
		return nil
	}
	for _, d := range tasks {
		if err := miso.ScheduleCron(d); err != nil {
			return fmt.Errorf("failed to schedule cron job, %+v, %w", d, err)
		}
	}
	return nil
}

// Set the schedule group for current node, by default it's 'default'
func SetScheduleGroup(groupName string) {
	module().setScheduleGroup(groupName)
}

// Check if current node is master
func IsTaskMaster(rail miso.Rail) bool {
	return module().isTaskMaster(rail)
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
func ScheduleDistributedTask(t miso.Job) error {
	return module().scheduleDistributedTask(t)
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

func distriTaskBootstrapCondition(app *miso.MisoApp, rail miso.Rail) (bool, error) {
	return appModule(app).enabled(), nil
}

func distriTaskBootstrap(app *miso.MisoApp, rail miso.Rail) error {
	m := appModule(app)
	app.AddOrderedShutdownHook(miso.DefShutdownOrder-1, func() { m.stop() })
	return m.bootstrapAsComponent(rail)
}
