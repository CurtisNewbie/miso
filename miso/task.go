package miso

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	// taskdule group name
	group string = "default"

	// identifier for current node
	nodeId string

	// mutex for core proerties (group, nodeId, states, and masterNode election)
	coreMut sync.Mutex

	// _state (atomic int32) of distributed task taskduler, use getState()/setState() to load/store
	_state int32

	masterTicker   *time.Ticker = nil // ticker for refreshing master node lock
	masterTickerMu sync.Mutex         // mutex for masterTicker
)

const (
	taskInitState    int32 = 0 // intial state, no task being taskduled at all
	taskPendingState int32 = 1 // pending state, tasks are taskduled, but the taskduler hasn't been started
	taskStartedState int32 = 2 // started state, taskduler has been started
	taskStoppedState int32 = 3 // stopped state, taskduler has been stopped

	// default ttl for master lock key in redis (1 min)
	defMstLockTtl = 1 * time.Minute
)

type NamedTask = func(Rail) error
type Task = func(Rail)

func init() {
	SetDefProp(PROP_TASK_SCHEDULING_ENABLED, true)

	// set initial state
	setState(taskInitState)
}

// Check if it's disabled (based on configuration, doesn't affect method call)
func IsTaskSchedulingDisabled() bool {
	return !GetPropBool(PROP_TASK_SCHEDULING_ENABLED)
}

// Check whether task taskduler has pending tasks, waiting to be started
func IsTaskSchedulerPending() bool {
	return getState() == taskPendingState
}

// Atomically set state (should lock first before invoke this func)
func setState(newState int32) {
	atomic.StoreInt32(&_state, int32(newState))
}

// Atomically load state (it's save to read without locking)
func getState() int32 {
	return atomic.LoadInt32(&_state)
}

// Enable distributed task taskduling, return whether task taskduling is enabled
func enableTaskScheduling() bool {
	coreMut.Lock()
	defer coreMut.Unlock()

	if getState() != taskPendingState {
		return false
	}
	setState(taskStartedState)

	proposedGroup := GetPropStr(PROP_TASK_SCHEDULING_GROUP)
	if proposedGroup == "" {
		proposedGroup = GetPropStr(PROP_APP_NAME)
	}
	if proposedGroup != "" {
		group = proposedGroup
	}

	uid, e := uuid.NewUUID()
	if e != nil {
		logrus.Fatalf("NewUUID: %v", e)
	}
	nodeId = uid.String()
	logrus.Infof("Enable distributed task taskduling, current node id: '%s', group: '%s'", nodeId, group)
	return true
}

// Set the taskdule group for current node, by default it's 'default'
func SetScheduleGroup(groupName string) {
	coreMut.Lock()
	defer coreMut.Unlock()

	g := strings.TrimSpace(groupName)
	if g == "" {
		return // still using default group name
	}
	group = g
}

// Check if current node is master
func IsMasterNode() bool {
	key := getMasterNodeLockKey()
	val, e := GetStr(key)
	if e != nil {
		logrus.Errorf("check is master failed: %v", e)
		return false
	}
	logrus.Debugf("check is master node, key: %v, onRedis: %v, nodeId: %v", key, val, nodeId)
	return val == nodeId
}

// Get lock key for master node
//
// Applications are grouped together as a cluster (each cluster is differentiated by its appGroup name
// we only try to become the master node in our cluster
func getMasterNodeLockKey() string {
	return "task:master:group:" + group
}

// taskdule a distributed task.
//
// Applications are grouped together as a cluster (each cluster is differentiated by its group name),
// only the master node can run the taskduled tasks.
//
// Tasks are pending until StartTasktaskdulerAsync() is called.
//
// E.g.,
//
//	task.taskduleDistributedTask("0/1 * * * * ?", true, myTask)
func ScheduleDistributedTask(cron string, withSeconds bool, task Task) error {
	if getState() == taskInitState {
		coreMut.Lock()
		if getState() == taskInitState {
			setState(taskPendingState)
		}
		coreMut.Unlock()
	}

	return ScheduleCron(cron, withSeconds, func() {
		ec := EmptyRail()
		if !tryBecomeMaster() {
			ec.Debug("Not master node, skip taskduled task")
			return
		}

		task(ec)
	})
}

// taskdule a named distributed task
//
// Applications are grouped together as a cluster (each cluster is differentiated by its group name),
// only the master node can run the taskduled tasks.
//
// Tasks are pending until StartTasktaskdulerAsync() is called.
//
// E.g.,
//
//	taskduleNamedDistributedTask("0/1 * * * * ?", true, "Very important task", myTask)
func ScheduleNamedDistributedTask(cron string, withSeconds bool, name string, task NamedTask) error {
	logrus.Infof("taskdule distributed task '%s' cron: '%s'", name, cron)
	return ScheduleDistributedTask(cron, withSeconds, func(ec Rail) {
		ec.Infof("Running task '%s'", name)
		start := time.Now()
		e := task(ec)
		if e == nil {
			ec.Infof("Task '%s' finished, took: %s", name, time.Since(start))
			return
		}

		ec.Errorf("Task '%s' failed, took: %s, %v", name, time.Since(start), e)
	})
}

// Start distributed taskduler asynchronously
func StartTaskSchedulerAsync() {
	if getState() != taskPendingState {
		return
	}

	if enableTaskScheduling() {
		StartSchedulerAsync()
	}
}

// Start distributed taskduler, current routine is blocked
func StartTaskSchedulerBlocking() {
	if getState() != taskPendingState {
		return
	}

	if enableTaskScheduling() {
		StartSchedulerBlocking()
	}
}

// Shutdown distributed job taskduling
func StopTaskScheduler() {
	coreMut.Lock()
	defer coreMut.Unlock()

	if getState() != taskStartedState {
		return
	}
	setState(taskStoppedState)

	StopScheduler()

	stopMasterLockRefreshingTicker()
	releaseMasterNodeLock()

	// if we are previously the master node, the lock is refreshed every 5 seconds with the ttl 1m
	// this should be pretty enough to release the lock before the expiration
	if IsMasterNode() {
		GetRedis().Expire(getMasterNodeLockKey(), 1*time.Second) // release master node after 1s
	}
}

// Start refreshing master lock ticker
func startMasterLockRefreshingTicker() {
	masterTickerMu.Lock()
	defer masterTickerMu.Unlock()

	if masterTicker != nil {
		return
	}

	masterTicker = time.NewTicker(5 * time.Second)
	go func(c <-chan time.Time) {
		for {
			refreshMasterLockTtl()
			<-c
		}
	}(masterTicker.C)
}

func releaseMasterNodeLock() {
	cmd := GetRedis().Eval(`
	if (call('EXISTS', KEYS[1]) == 0) then
		return 0;
	end;

	if (call('GET', KEYS[1]) == tostring(ARGV[1])) then
		call('DEL', KEYS[1])
		return 1;
	end;
	return 0;`, []string{getMasterNodeLockKey()}, nodeId)
	if cmd.Err() != nil {
		logrus.Errorf("Failed to release master node lock, %v", cmd.Err())
		return
	}
	logrus.Debugf("Release master node lock, %v", cmd.Val())
}

// Stop refreshing master lock ticker
func stopMasterLockRefreshingTicker() {
	masterTickerMu.Lock()
	defer masterTickerMu.Unlock()
	if masterTicker == nil {
		return
	}

	masterTicker.Stop()
	masterTicker = nil
}

// Refresh master lock key ttl
func refreshMasterLockTtl() error {
	return GetRedis().Expire(getMasterNodeLockKey(), defMstLockTtl).Err()
}

// Try to become master node
func tryBecomeMaster() bool {
	coreMut.Lock()
	defer coreMut.Unlock()

	if IsMasterNode() {
		return true
	}

	bcmd := GetRedis().SetNX(getMasterNodeLockKey(), nodeId, defMstLockTtl)
	if bcmd.Err() != nil {
		logrus.Errorf("try to become master node: '%v'", bcmd.Err())
		return false
	}

	isMaster := bcmd.Val()
	if isMaster {
		logrus.Infof("Elected to be the master node for group: '%s'", group)
		startMasterLockRefreshingTicker()
	} else {
		stopMasterLockRefreshingTicker()
	}
	return isMaster
}
