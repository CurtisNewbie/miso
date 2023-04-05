package task

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/redis"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	// schedule group name
	group string = "default"

	// identifier for current node
	nodeId string

	// mutex for common proerties (group, nodeId)
	commonMut sync.Mutex

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

func init() {
	common.SetDefProp(common.PROP_TASK_SCHEDULING_ENABLED, true)

	// set initial state
	setState(initState)
}

// Check if it's disabled (based on configuration, doesn't affect method call)
func IsTaskSchedulingDisabled() bool {
	return !common.GetPropBool(common.PROP_TASK_SCHEDULING_ENABLED)
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
	commonMut.Lock()
	defer commonMut.Unlock()

	if getState() != pendingState {
		return false
	}
	setState(startedState)

	proposedGroup := common.GetPropStr(common.PROP_TASK_SCHEDULING_GROUP)
	if proposedGroup == "" {
		proposedGroup = common.GetPropStr(common.PROP_APP_NAME)
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

	/*
			Not really needed, because we always check and try to become master when tasks are triggered,
			and whenever we are the master, we create a ticker thread to refresh the expiration

		go func() {
			for {
				if getState() == stoppedState {
					return
				}

				tryBecomeMaster()
				time.Sleep(15 * time.Second)
			}
		}()
	*/
}

// Set the schedule group for current node, by default it's 'default'
func SetScheduleGroup(groupName string) {
	commonMut.Lock()
	defer commonMut.Unlock()

	g := strings.TrimSpace(groupName)
	if g == "" {
		return // still using default group name
	}
	group = g
}

// Check if current node is master
func IsMasterNode() bool {
	val, e := redis.GetStr(getMasterNodeLockKey())
	if e != nil {
		logrus.Errorf("IsMasterNode: %v", e)
		return false
	}
	return val == nodeId
}

// Get lock key for master node
//
// Applications are grouped together as a cluster (each cluster is differentiated by its appGroup name
// we only try to become the master node in our cluster
func getMasterNodeLockKey() string {
	return "task:master:group:" + group
}

// Schedule a distributed task
//
// Applications are grouped together as a cluster (each cluster is differentiated by its group name),
// only the master node can run the scheduled tasks.
//
// Tasks are pending until StartTaskSchedulerAsync() is called
func ScheduleDistributedTask(cron string, runnable func(common.ExecContext)) {
	if getState() == initState {
		commonMut.Lock()
		if getState() == initState {
			setState(pendingState)
		}
		commonMut.Unlock()
	}

	common.ScheduleCron(cron, func() {
		if getState() == stoppedState {
			return // extra check, but scheduler is supposed to be stopped before the state is updated, this is quite unnecessary
		}

		if tryBecomeMaster() {
			runnable(common.EmptyExecContext())
		}
	})
}

// Schedule a named distributed task
//
// Applications are grouped together as a cluster (each cluster is differentiated by its group name),
// only the master node can run the scheduled tasks.
//
// Tasks are pending until StartTaskSchedulerAsync() is called
func ScheduleNamedDistributedTask(cron string, name string, runnable func(common.ExecContext)) {
	if getState() == initState {
		commonMut.Lock()
		if getState() == initState {
			setState(pendingState)
		}
		commonMut.Unlock()
	}

	common.ScheduleCron(cron, func() {
		if getState() == stoppedState {
			return // extra check, but scheduler is supposed to be stopped before the state is updated, this is quite unnecessary
		}

		if tryBecomeMaster() {
			ec := common.EmptyExecContext()
			ec.Log.Infof("Running task '%s'", name)
			start := time.Now()
			runnable(ec)
			ec.Log.Infof("Task '%s' finished, took: %s", name, time.Since(start))
		}
	})
}


// Start distributed scheduler asynchronously
func StartTaskSchedulerAsync() {
	if getState() != pendingState {
		return
	}

	if enableTaskScheduling() {
		common.StartSchedulerAsync()
	}
}

// Start distributed scheduler, current routine is blocked
func StartTaskSchedulerBlocking() {
	if getState() != pendingState {
		return
	}

	if enableTaskScheduling() {
		common.StartSchedulerBlocking()
	}
}

// Shutdown distributed job scheduling
func StopTaskScheduler() {
	commonMut.Lock()
	defer commonMut.Unlock()

	if getState() != startedState {
		return
	}
	setState(stoppedState)

	common.StopScheduler()

	stopMasterLockRefreshingTicker()

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
