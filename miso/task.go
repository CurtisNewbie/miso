package miso

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	// mutex for core proerties (group, nodeId, states, and masterNode election)
	dtaskMut sync.Mutex

	// schedule group name
	group string = "default"

	// identifier for current node
	nodeId string

	// tasks
	dtasks []Job = []Job{}

	// ticker for refreshing master node lock
	masterTicker *time.Ticker = nil
)

const (
	// default ttl for master lock key in redis (1 min)
	defMstLockTtl = 1 * time.Minute
)

func init() {
	SetDefProp(PropTaskSchedulingEnabled, true)

	// run before SchedulerBootstrap
	RegisterBootstrapCallback(ComponentBootstrap{
		Name: "Bootstrap Distributed Task Scheduler",
		Condition: func(rail Rail) (bool, error) {
			return !IsTaskSchedulingDisabled() && len(dtasks) > 0, nil
		},
		Bootstrap: DistriTaskBootstrap,
		Order:     BootstrapOrderL4,
	})
}

// Check if it's disabled (based on configuration, doesn't affect method call)
func IsTaskSchedulingDisabled() bool {
	return !GetPropBool(PropTaskSchedulingEnabled)
}

func registerTasks(rail Rail, tasks []Job) error {
	if len(tasks) < 1 {
		return nil
	}
	for _, d := range tasks {
		if err := ScheduleCron(d); err != nil {
			return fmt.Errorf("failed to schedule cron job, %+v, %w", d, err)
		}
	}
	return nil
}

func prepareTaskScheduling(rail Rail, tasks []Job) error {
	if len(tasks) < 1 {
		return nil
	}
	proposedGroup := GetPropStr(ProptaskSchedulingGroup)
	if proposedGroup == "" {
		proposedGroup = GetPropStr(PropAppName)
	}
	if proposedGroup != "" {
		group = proposedGroup
	}
	nodeId = ERand(30)

	if err := registerTasks(rail, tasks); err != nil {
		return err
	}
	rail.Infof("Scheduled %d distributed tasks, current node id: '%s', group: '%s'", len(tasks), nodeId, group)
	return nil
}

// Set the schedule group for current node, by default it's 'default'
func SetScheduleGroup(groupName string) {
	dtaskMut.Lock()
	defer dtaskMut.Unlock()

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
//	job := miso.Job{
//		Name:            "Very important task",
//		Cron:            "0/1 * * * * ?",
//		CronWithSeconds: true,
//		Run: MyTask,
//	}
//	ScheduleDistributedTask(job)
func ScheduleDistributedTask(t Job) error {
	Infof("Schedule distributed task '%s' cron: '%s'", t.Name, t.Cron)
	actualRun := t.Run
	t.Run = func(rail Rail) error {
		dtaskMut.Lock()
		if !tryTaskMaster(rail) {
			rail.Debug("Not master node, skip scheduled task")
			return nil
		}
		dtaskMut.Unlock()
		return actualRun(rail)
	}
	dtasks = append(dtasks, t)
	return nil
}

// Start distributed scheduler asynchronously
func StartTaskSchedulerAsync(rail Rail) error {
	dtaskMut.Lock()
	defer dtaskMut.Unlock()
	if len(dtasks) < 1 {
		return nil
	}
	if err := prepareTaskScheduling(rail, dtasks); err != nil {
		return err
	}
	StartSchedulerAsync()
	return nil
}

// Start distributed scheduler, current routine is blocked
func StartTaskSchedulerBlocking(rail Rail) error {
	dtaskMut.Lock()
	defer dtaskMut.Unlock()
	if len(dtasks) < 1 {
		return nil
	}
	if err := prepareTaskScheduling(rail, dtasks); err != nil {
		return err
	}
	StartSchedulerBlocking()
	return nil
}

// Shutdown distributed job scheduling
func StopTaskScheduler() {
	dtaskMut.Lock()
	defer dtaskMut.Unlock()

	StopScheduler()
	stopTaskMasterLockTicker()
	releaseMasterNodeLock(EmptyRail())
}

// Start refreshing master lock ticker
func startTaskMasterLockTicker() {
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
	rail.Debugf("Release master node lock, released? %v", cmd.Val())
}

// Stop refreshing master lock ticker
func stopTaskMasterLockTicker() {
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

func DistriTaskBootstrap(rail Rail) error {
	AddShutdownHook(func() { StopTaskScheduler() })
	dtaskMut.Lock()
	defer dtaskMut.Unlock()
	if err := prepareTaskScheduling(rail, dtasks); err != nil {
		return fmt.Errorf("failed to prepareTaskScheduling, %w", err)
	}
	tryTaskMaster(rail)
	return nil
}
