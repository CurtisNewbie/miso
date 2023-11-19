package miso

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
)

type schedState = int

type Job struct {
	Name            string
	Cron            string
	CronWithSeconds bool
	Run             func(Rail) error
}

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

func doScheduleCron(s *gocron.Scheduler, job Job) error {
	var err error

	wrappedJob := func() {
		rail := EmptyRail()
		rail.Infof("Running job '%s'", job.Name)
		start := time.Now()
		e := job.Run(rail)
		if e == nil {
			rail.Infof("Job '%s' finished, took: %s", job.Name, time.Since(start))
			return
		}
		rail.Errorf("Job '%s' failed, took: %s, %v", job.Name, time.Since(start), e)
	}

	if job.CronWithSeconds {
		_, err = s.CronWithSeconds(job.Cron).Do(wrappedJob)
	} else {
		_, err = s.Cron(job.Cron).Do(wrappedJob)
	}

	if err != nil {
		return fmt.Errorf("failed to schedule cron job, cron: %v, withSeconds: %v, %w", job.Cron, job.CronWithSeconds, err)
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
func ScheduleCron(job Job) error {
	s := getScheduler()
	return doScheduleCron(s, job)
}
