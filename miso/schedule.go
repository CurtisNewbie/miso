package miso

import (
	"fmt"
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
	state = scheInitState
)

const (
	scheInitState    schedState = 0
	scheStartedState schedState = 1
	scheStoppedState schedState = 2
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

// Create new Schedulr at UTC time, with default-mode
func newScheduler() *gocron.Scheduler {
	sche := gocron.NewScheduler(time.UTC)
	return sche
}

func doScheduleCron(s *gocron.Scheduler, cron string, withSeconds bool, runnable func()) error {
	var err error
	if withSeconds {
		_, err = s.CronWithSeconds(cron).Do(func() {
			runnable()
		})
	} else {
		_, err = s.Cron(cron).Do(func() {
			runnable()
		})
	}
	if err != nil {
		return fmt.Errorf("failed to schedule cron job, cron: %v, withSeconds: %v, %w", cron, withSeconds, err)
	}
	return nil
}

// Stop scheduler
func StopScheduler() {
	scheLock.Lock()
	if scheduler == nil || state != scheStartedState {
		return
	}
	state = scheStoppedState
	scheLock.Unlock()

	getScheduler().Stop()
}

// Start scheduler and block current routine
func StartSchedulerBlocking() {
	scheLock.Lock()
	defer scheLock.Unlock()

	if scheduler == nil || state != scheInitState {
		return
	}

	state = scheStartedState
	getScheduler().StartBlocking()
}

// Start scheduler asynchronously
func StartSchedulerAsync() {
	scheLock.Lock()
	if scheduler == nil || state != scheInitState {
		return
	}
	state = scheStartedState
	scheLock.Unlock()

	getScheduler().StartAsync()
}

// add a cron job to scheduler, note that the cron expression includes second, e.g., '*/1 * * * * *'
//
// this func doesn't start the scheduler
func ScheduleCron(cron string, withSeconds bool, runnable func()) error {
	s := getScheduler()
	return doScheduleCron(s, cron, withSeconds, runnable)
}
