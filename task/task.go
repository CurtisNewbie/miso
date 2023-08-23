package task

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/curtisnewbie/miso/core"
	"github.com/curtisnewbie/miso/redis"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
	initState    int32 = 0 // intial state, no task being scheduled at all
	pendingState int32 = 1 // pending state, tasks are scheduled, but the scheduler hasn't been started
	startedState int32 = 2 // started state, scheduler has been started
	stoppedState int32 = 3 // stopped state, scheduler has been stopped

	// default ttl for master lock key in redis (1 min)
	defMstLockTtl = 1 * time.Minute
)

type NamedTask = func(core.Rail) error
type Task = func(core.Rail)

func init() {
	core.SetDefProp(core.PROP_TASK_SCHEDULING_ENABLED, true)

	// set initial state
	setState(initState)
}

// Check if it's disabled (based on configuration, doesn't affect method call)
func IsTaskSchedulingDisabled() bool {
	return !core.GetPropBool(core.PROP_TASK_SCHEDULING_ENABLED)
}

// Check whether task scheduler has pending tasks, waiting to be started
func IsTaskSchedulerPending() bool {
	return getState() == pendingState
}

// Atomically set state (should lock first before invoke this func)
func setState(newState int32) {
	atomic.StoreInt32(&_state, int32(newState))
}

// Atomically load state (it's save to read without locking)
func getState() int32 {
	return atomic.LoadInt32(&_state)
}

// Enable distributed task scheduling, return whether task scheduling is enabled
func enableTaskScheduling() bool {
	coreMut.Lock()
	defer coreMut.Unlock()

	if getState() != pendingState {
		return false
	}
	setState(startedState)

	proposedGroup := core.GetPropStr(core.PROP_TASK_SCHEDULING_GROUP)
	if proposedGroup == "" {
		proposedGroup = core.GetPropStr(core.PROP_APP_NAME)
	}
	if proposedGroup != "" {
		group = proposedGroup
	}

	uid, e := uuid.NewUUID()
	if e != nil {
		logrus.Fatalf("NewUUID: %v", e)
	}
	nodeId = uid.String()
	logrus.Infof("Enable distributed task scheduling, current node id: '%s', group: '%s'", nodeId, group)
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
func IsMasterNode() bool {
	key := getMasterNodeLockKey()
	val, e := redis.GetStr(key)
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

// Schedule a distributed task.
//
// Applications are grouped together as a cluster (each cluster is differentiated by its group name),
// only the master node can run the scheduled tasks.
//
// Tasks are pending until StartTaskSchedulerAsync() is called.
//
// E.g.,
//
//	task.ScheduleDistributedTask("0/1 * * * * ?", true, myTask)
func ScheduleDistributedTask(cron string, withSeconds bool, task Task) error {
	if getState() == initState {
		coreMut.Lock()
		if getState() == initState {
			setState(pendingState)
		}
		coreMut.Unlock()
	}

	return core.ScheduleCron(cron, withSeconds, func() {
		ec := core.EmptyRail()
		if !tryBecomeMaster() {
			ec.Debug("Not master node, skip scheduled task")
			return
		}

		task(ec)
	})
}

// Schedule a named distributed task
//
// Applications are grouped together as a cluster (each cluster is differentiated by its group name),
// only the master node can run the scheduled tasks.
//
// Tasks are pending until StartTaskSchedulerAsync() is called.
//
// E.g.,
//
//	ScheduleNamedDistributedTask("0/1 * * * * ?", true, "Very important task", myTask)
func ScheduleNamedDistributedTask(cron string, withSeconds bool, name string, task NamedTask) error {
	logrus.Infof("Schedule distributed task '%s' cron: '%s'", name, cron)
	return ScheduleDistributedTask(cron, withSeconds, func(ec core.Rail) {
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

// Start distributed scheduler asynchronously
func StartTaskSchedulerAsync() {
	if getState() != pendingState {
		return
	}

	if enableTaskScheduling() {
		core.StartSchedulerAsync()
	}
}

// Start distributed scheduler, current routine is blocked
func StartTaskSchedulerBlocking() {
	if getState() != pendingState {
		return
	}

	if enableTaskScheduling() {
		core.StartSchedulerBlocking()
	}
}

// Shutdown distributed job scheduling
func StopTaskScheduler() {
	coreMut.Lock()
	defer coreMut.Unlock()

	if getState() != startedState {
		return
	}
	setState(stoppedState)

	core.StopScheduler()

	stopMasterLockRefreshingTicker()
	releaseMasterNodeLock()

	// if we are previously the master node, the lock is refreshed every 5 seconds with the ttl 1m
	// this should be pretty enough to release the lock before the expiration
	if IsMasterNode() {
		redis.GetRedis().Expire(getMasterNodeLockKey(), 1*time.Second) // release master node after 1s
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
	cmd := redis.GetRedis().Eval(`
	if (redis.call('EXISTS', KEYS[1]) == 0) then
		return 0;
	end;

	if (redis.call('GET', KEYS[1]) == tostring(ARGV[1])) then
		redis.call('DEL', KEYS[1])
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
	return redis.GetRedis().Expire(getMasterNodeLockKey(), defMstLockTtl).Err()
}

// Try to become master node
func tryBecomeMaster() bool {
	coreMut.Lock()
	defer coreMut.Unlock()

	if IsMasterNode() {
		return true
	}

	bcmd := redis.GetRedis().SetNX(getMasterNodeLockKey(), nodeId, defMstLockTtl)
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
