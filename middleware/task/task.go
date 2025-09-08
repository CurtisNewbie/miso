package task

import (
	"context"
	"errors"
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
	defMstLockTtl    = 1 * time.Minute
	defMstLockTtlSec = 60
)

func init() {
	// run before SchedulerBootstrap
	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Bootstrap Distributed Task Scheduler",
		Condition: distriTaskBootstrapCondition,
		Bootstrap: distriTaskBootstrap,
		Order:     miso.BootstrapOrderL4,
	})
}

var module = miso.InitAppModuleFunc(func() *taskModule {
	return &taskModule{
		dtaskMut:      &sync.Mutex{},
		group:         "default",
		masterTickers: map[string]*time.Ticker{},
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
	masterTickers map[string]*time.Ticker
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
func (m *taskModule) isTaskMaster(rail miso.Rail, jobName string) bool {
	key := m.getTaskMasterKey(jobName)
	val, e := redis.GetStr(key)
	if e != nil {
		rail.Errorf("Check is master failed: %v", e)
		return false
	}
	rail.Debugf("Check is master node, key: %v, onRedis: %v, nodeId: %v, job: %v", key, val, m.nodeId, jobName)
	return val == m.nodeId
}

// Get lock key for master node
//
// Applications are grouped together as a cluster (each cluster is differentiated by its appGroup name
// we only try to become the master node in our cluster
func (m *taskModule) getTaskMasterKey(jobName string) string {
	return "task:master:group:" + m.group + ":" + jobName
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
	t.Run = func(rail miso.Rail) error {
		if miso.GetPropBool("task.scheduling." + t.Name + ".disabled") {
			rail.Debugf("Task '%v' disabled, skipped", t.Name)

			m.dtaskMut.Lock()
			defer m.dtaskMut.Unlock()
			m.releaseTaskMaster(rail, t.Name)
			return nil
		}

		m.dtaskMut.Lock()
		if !m.tryTaskMaster(rail, t.Name) {
			m.dtaskMut.Unlock()
			rail.Debugf("Not master node, skip scheduled task '%v'", t.Name)
			return nil
		}
		m.dtaskMut.Unlock()

		if logJobExec {
			rail.Infof("Running task '%s'", t.Name)
			miso.LogJobNextRun(rail, t.Name)
		}

		start := time.Now()
		err := actualRun(rail)
		took := time.Since(start)

		if logJobExec {
			rail.Infof("Task '%s' finished, took: %s", t.Name, took)
		}

		return err
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
	return nil
}

func (m *taskModule) stop() {
	rail := miso.EmptyRail()
	m.dtaskMut.Lock()
	defer m.dtaskMut.Unlock()

	miso.StopScheduler()
	for _, dt := range m.dtasks {
		m.releaseTaskMaster(rail, dt.Name)
	}
}

// Start refreshing master lock ticker
func (m *taskModule) startTaskMasterLockTicker(jobName string) {
	if m.masterTickers[jobName] != nil {
		return
	}
	tk := time.NewTicker(5 * time.Second)
	m.masterTickers[jobName] = tk
	go func(c <-chan time.Time) {
		for {
			_ = m.refreshTaskMasterLock(jobName)
			<-c
		}
	}(tk.C)
}

func (m *taskModule) releaseMasterNodeLock(jobName string) {
	cmd := redis.GetRedis().Eval(context.Background(), `
	if (redis.call('EXISTS', KEYS[1]) == 0) then
		return 0;
	end;

	if (redis.call('GET', KEYS[1]) == tostring(ARGV[1])) then
		redis.call('DEL', KEYS[1])
		return 1;
	end;
	return 0;`, []string{m.getTaskMasterKey(jobName)}, m.nodeId)
	if cmd.Err() != nil {
		if errors.Is(cmd.Err(), redis.Nil) {
			return
		}
		miso.Errorf("Failed to release master node lock for %v, %v", jobName, miso.WrapErr(cmd.Err()))
		return
	}
	miso.Debugf("Release master node lock for %v, released? %v", jobName, cmd.Val() == int64(1))
}

// Stop refreshing master lock ticker
func (m *taskModule) stopTaskMasterLockTicker(jobName string) {
	if m.masterTickers[jobName] == nil {
		return
	}

	m.masterTickers[jobName].Stop()
	m.masterTickers[jobName] = nil
}

// Refresh master lock key ttl
func (m *taskModule) refreshTaskMasterLock(jobName string) error {
	err := redis.GetRedis().Eval(context.Background(), `
	if (redis.call('GET', KEYS[1]) == tostring(ARGV[1])) then
		redis.call('EXPIRE', KEYS[1], tonumber(ARGV[2]))
		return 1;
	end;
	return 0;
	`, []string{m.getTaskMasterKey(jobName)}, m.nodeId, defMstLockTtlSec).Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			miso.Debugf("Failed to refreshTaskMasterLock for job '%v', masterLock not obtained", jobName)
			return nil
		}
		miso.Errorf("Failed to refreshTaskMasterLock for job '%v', %v", jobName, err)
	} else {
		miso.Debugf("Did refreshTaskMasterLock for job '%v'", jobName)
	}
	return err
}

// Try to become master node
func (m *taskModule) tryTaskMaster(rail miso.Rail, jobName string) bool {
	if m.isTaskMaster(rail, jobName) {
		return true
	}

	bcmd := redis.GetRedis().SetNX(rail.Context(), m.getTaskMasterKey(jobName), m.nodeId, defMstLockTtl)
	if bcmd.Err() != nil {
		rail.Errorf("Try to become master node for job %v, %v", jobName, miso.WrapErr(bcmd.Err()))
		return false
	}

	isMaster := bcmd.Val()
	if isMaster {
		rail.Infof("Elected to be the master node for group: '%s', job: '%v'", m.group, jobName)
		m.startTaskMasterLockTicker(jobName)
	} else {
		m.stopTaskMasterLockTicker(jobName)
	}
	return isMaster
}

func (m *taskModule) releaseTaskMaster(rail miso.Rail, jobName string) {
	if m.isTaskMaster(rail, jobName) {
		rail.Infof("Releasing master node for job %v", jobName)
		m.stopTaskMasterLockTicker(jobName)
		m.releaseMasterNodeLock(jobName)
	}
}

func (m *taskModule) registerTasks(tasks []miso.Job) error {
	if len(tasks) < 1 {
		return nil
	}
	for _, d := range tasks {
		if err := miso.ScheduleCron(d); err != nil {
			return fmt.Errorf("failed to schedule cron job, %+v, %w", d, miso.WrapErr(err))
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
