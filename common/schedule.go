package common

import (
	"sync"
	"time"

	"github.com/go-co-op/gocron"
)

type schedState = int

var (
	// lazy-init, cached scheduler
	scheduler *gocron.Scheduler = nil

	// lock for scheduler
	scheLock sync.Mutex

	// state of scheduler
	state = initState
)

const (
	initState    schedState = 0
	startedState schedState = 1
	stoppedState schedState = 2
)

// Whether scheduler is initialized
func HasScheduler() bool {
	scheLock.Lock()
	defer scheLock.Unlock()
	return scheduler != nil
}

// Get the lazy-initialized, cached scheduler
func getScheduler() *gocron.Scheduler {
	scheLock.Lock()
	defer scheLock.Unlock()

	if scheduler != nil {
		return scheduler
	}

	scheduler = newScheduler()
	return scheduler
}

// Create new Schedulr at UTC time, with singleton-mode
func newScheduler() *gocron.Scheduler {
	return gocron.NewScheduler(time.UTC).SingletonMode()
}

func doScheduleCron(s *gocron.Scheduler, cron string, runnable func()) *gocron.Scheduler {
	s.CronWithSeconds(cron).Do(func() {
		runnable()
	})
	return s
}

// Stop scheduler
func StopScheduler() {
	scheLock.Lock()
	if scheduler == nil || state != startedState {
		return
	}
	state = stoppedState
	scheLock.Unlock()

	getScheduler().Stop()
}

// Start scheduler and block current routine
func StartSchedulerBlocking() {
	scheLock.Lock()
	defer scheLock.Unlock()

	if scheduler == nil || state != initState {
		return
	}

	state = startedState
	getScheduler().StartBlocking()
}

// Start scheduler asynchronously
func StartSchedulerAsync() {
	scheLock.Lock()
	if scheduler == nil || state != initState {
		return
	}
	state = startedState
	scheLock.Unlock()

	getScheduler().StartAsync()
}

// add a cron job to scheduler, note that the cron expression includes second, e.g., '*/1 * * * * *'
//
// this func doesn't start the scheduler
func ScheduleCron(cron string, runnable func()) *gocron.Scheduler {
	s := getScheduler()
	return doScheduleCron(s, cron, runnable)
}
