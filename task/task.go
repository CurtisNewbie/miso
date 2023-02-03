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

// Enable distributed task scheduling
func enableTaskScheduling() {
	commonMut.Lock()
	defer commonMut.Unlock()

	if getState() != pendingState {
		return
	}
	setState(startedState)

	uid, e := uuid.NewUUID()
	if e != nil {
		logrus.Fatalf("NewUUID: %v", e)
	}
	nodeId = uid.String()
	logrus.Infof("Enabling distributed task scheduling, current node id: '%s', group: '%s'", nodeId, group)

	go func() {
		for {
			if getState() == stoppedState {
				return
			}

			tryBecomeMaster()
			time.Sleep(15 * time.Second)
		}
	}()
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
// only the master node can run the scheduled tasks
func ScheduleDistributedTask(cron string, runnable func()) {
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
			runnable()
		}
	})
}

// Start distributed scheduler asynchronously
func StartTaskSchedulerAsync() {
	if getState() != pendingState {
		return
	}

	enableTaskScheduling()
	common.StartSchedulerAsync()
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

	if tryBecomeMaster() {
		stopMasterLockRefreshingTicker()
		redis.GetRedis().Expire(getMasterNodeLockKey(), 3*time.Second) // release master node
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
