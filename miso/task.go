package miso

// TODO: Refactor this

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// schedule group name
	group string = "default"

	// identifier for current node
	nodeId string

	// mutex for core proerties (group, nodeId, states, and masterNode election)
	coreMut sync.Mutex

	// _state (atomic int32) of distributed task scheduler, use getState()/setState() to load/store
	_state int32

	masterTicker   *time.Ticker = nil // ticker for refreshing master node lock
	masterTickerMu sync.Mutex         // mutex for masterTicker
)

const (
	taskInitState    int32 = 0 // intial state, no task being scheduled at all
	taskPendingState int32 = 1 // pending state, tasks are scheduled, but the scheduler hasn't been started
	taskStartedState int32 = 2 // started state, scheduler has been started
	taskStoppedState int32 = 3 // stopped state, scheduler has been stopped

	// default ttl for master lock key in redis (1 min)
	defMstLockTtl = 1 * time.Minute
)

func init() {
	SetDefProp(PropTaskSchedulingEnabled, true)

	// set initial state
	setTaskState(taskInitState)
}

// Check if it's disabled (based on configuration, doesn't affect method call)
func IsTaskSchedulingDisabled() bool {
	return !GetPropBool(PropTaskSchedulingEnabled)
}

// Check whether task scheduler has pending tasks, waiting to be started
func IsTaskSchedulerPending() bool {
	return getTaskState() == taskPendingState
}

// Atomically set state (should lock first before invoke this func)
func setTaskState(newState int32) {
	atomic.StoreInt32(&_state, int32(newState))
}

// Atomically load state (it's save to read without locking)
func getTaskState() int32 {
	return atomic.LoadInt32(&_state)
}

// Enable distributed task scheduling, return whether task scheduling is enabled
func enableTaskScheduling(rail Rail) bool {
	coreMut.Lock()
	defer coreMut.Unlock()

	if getTaskState() != taskPendingState {
		return false
	}
	setTaskState(taskStartedState)

	proposedGroup := GetPropStr(ProptaskSchedulingGroup)
	if proposedGroup == "" {
		proposedGroup = GetPropStr(PropAppName)
	}
	if proposedGroup != "" {
		group = proposedGroup
	}

	nodeId = ERand(30)
	rail.Infof("Enable distributed task scheduling, current node id: '%s', group: '%s'", nodeId, group)
	return true
}

// Set the schedule group for current node, by default it's 'default'
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
func IsTaskMaster(rail Rail) bool {
	key := getTaskMasterKey()
	val, e := GetStr(key)
	if e != nil {
		rail.Errorf("check is master failed: %v", e)
		return false
	}
	rail.Debugf("check is master node, key: %v, onRedis: %v, nodeId: %v", key, val, nodeId)
	return val == nodeId
}

// Get lock key for master node
//
// Applications are grouped together as a cluster (each cluster is differentiated by its appGroup name
// we only try to become the master node in our cluster
func getTaskMasterKey() string {
	return "task:master:group:" + group
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
//	ScheduleDistributedTask("0/1 * * * * ?", true, "Very important task", myTask)
func ScheduleDistributedTask(t Job) error {
	Infof("Schedule distributed task '%s' cron: '%s'", t.Name, t.Cron)

	if getTaskState() == taskInitState {
		coreMut.Lock()
		if getTaskState() == taskInitState {
			setTaskState(taskPendingState)
		}
		coreMut.Unlock()
	}

	preWrap := t.Run
	t.Run = func(rail Rail) error {
		if !tryTaskMaster(rail) {
			rail.Debug("Not master node, skip scheduled task")
			return nil
		}
		return preWrap(rail)
	}

	return ScheduleCron(t)
}

// Start distributed scheduler asynchronously
func StartTaskSchedulerAsync(rail Rail) {
	if getTaskState() != taskPendingState {
		return
	}

	if enableTaskScheduling(rail) {
		StartSchedulerAsync()
	}
}

// Start distributed scheduler, current routine is blocked
func StartTaskSchedulerBlocking(rail Rail) {
	if getTaskState() != taskPendingState {
		return
	}

	if enableTaskScheduling(rail) {
		StartSchedulerBlocking()
	}
}

// Shutdown distributed job scheduling
func StopTaskScheduler() {
	coreMut.Lock()
	defer coreMut.Unlock()

	if getTaskState() != taskStartedState {
		return
	}

	setTaskState(taskStoppedState)
	StopScheduler()
	stopTaskMasterLockTicker()

	rail := EmptyRail()
	releaseMasterNodeLock(rail)

	// if we are previously the master node, the lock is refreshed every 5 seconds with the ttl 1m
	// this should be pretty enough to release the lock before the expiration
	if IsTaskMaster(rail) {
		GetRedis().Expire(getTaskMasterKey(), 1*time.Second) // release master node after 1s
	}
}

// Start refreshing master lock ticker
func startTaskMasterLockTicker() {
	masterTickerMu.Lock()
	defer masterTickerMu.Unlock()

	if masterTicker != nil {
		return
	}

	masterTicker = time.NewTicker(5 * time.Second)
	go func(c <-chan time.Time) {
		for {
			refreshTaskMasterLock()
			<-c
		}
	}(masterTicker.C)
}

func releaseMasterNodeLock(rail Rail) {
	cmd := GetRedis().Eval(`
	if (redis.call('EXISTS', KEYS[1]) == 0) then
		return 0;
	end;

	if (redis.call('GET', KEYS[1]) == tostring(ARGV[1])) then
		redis.call('DEL', KEYS[1])
		return 1;
	end;
	return 0;`, []string{getTaskMasterKey()}, nodeId)
	if cmd.Err() != nil {
		rail.Errorf("Failed to release master node lock, %v", cmd.Err())
		return
	}
	rail.Debugf("Release master node lock, %v", cmd.Val())
}

// Stop refreshing master lock ticker
func stopTaskMasterLockTicker() {
	masterTickerMu.Lock()
	defer masterTickerMu.Unlock()
	if masterTicker == nil {
		return
	}

	masterTicker.Stop()
	masterTicker = nil
}

// Refresh master lock key ttl
func refreshTaskMasterLock() error {
	return GetRedis().Expire(getTaskMasterKey(), defMstLockTtl).Err()
}

// Try to become master node
func tryTaskMaster(rail Rail) bool {
	coreMut.Lock()
	defer coreMut.Unlock()

	if IsTaskMaster(rail) {
		return true
	}

	bcmd := GetRedis().SetNX(getTaskMasterKey(), nodeId, defMstLockTtl)
	if bcmd.Err() != nil {
		rail.Errorf("try to become master node: '%v'", bcmd.Err())
		return false
	}

	isMaster := bcmd.Val()
	if isMaster {
		rail.Infof("Elected to be the master node for group: '%s'", group)
		startTaskMasterLockTicker()
	} else {
		stopTaskMasterLockTicker()
	}
	return isMaster
}
