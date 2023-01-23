package common

import (
	"sync"
	"time"

	"github.com/go-co-op/gocron"
)

var (
	// lazy-init, cached scheduler
	scheduler *gocron.Scheduler = nil

	// lock for scheduler
	scheLock sync.Mutex
)

// Whether scheduler is initialized
func HasScheduler() bool {
	scheLock.Lock()
	defer scheLock.Unlock()
	return scheduler != nil 
}

// Get the lazy-initialized, cached scheduler
func GetScheduler() *gocron.Scheduler {
	scheLock.Lock()
	defer scheLock.Unlock()

	if scheduler != nil {
		return scheduler
	}

	scheduler = newScheduler()
	return scheduler
}

// create new Schedulr at UTC time, with singleton-mode
func newScheduler() *gocron.Scheduler {
	return gocron.NewScheduler(time.UTC).SingletonMode()
}

func doScheduleCron(s *gocron.Scheduler, cron string, runnable func()) *gocron.Scheduler {
	s.CronWithSeconds(cron).Do(func() {
		runnable()
	})
	return s
}

// add a cron job to scheduler, note that the cron expression includes second, e.g., '*/1 * * * * *'
//
// this func doesn't start the scheduler
func ScheduleCron(cron string, runnable func()) *gocron.Scheduler {
	s := GetScheduler()
	return doScheduleCron(s, cron, runnable)
}
