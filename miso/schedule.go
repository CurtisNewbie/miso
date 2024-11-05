package miso

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
)

type Job struct {
	Name                   string           // name of the job.
	Cron                   string           // cron expr.
	CronWithSeconds        bool             // whether cron expr contains the second field.
	Run                    func(Rail) error // actual job execution logic.
	LogJobExec             bool             // whether job execution should be logged, error msg is always logged and is not affected by this option.
	TriggeredOnBoostrapped bool             // whether job should be triggered when server is fully bootstrapped
}

// Hook triggered before job's execution.
type PreJobHook func(rail Rail, inf JobInf) error

// Hook triggered after job's execution.
type PostJobHook func(rail Rail, inf JobInf, stats JobExecStats) error

type JobExecStats struct {
	Time time.Duration
	Err  error
}

type JobInf struct {
	Name            string
	Cron            string
	CronWithSeconds bool
}

var (
	_scheduler     *gocron.Scheduler = nil
	_schedulerOnce sync.Once

	preJobHooks  = []PreJobHook{}
	postJobHooks = []PostJobHook{}
)

func init() {
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Cron Scheduler",
		Condition: func(rail Rail) (bool, error) { return HasScheduledJobs(), nil },
		Bootstrap: SchedulerBootstrap,
	})
}

// Whether scheduler is initialized
func HasScheduledJobs() bool {
	return getScheduler().Len() > 0
}

// Get the lazy-initialized, cached scheduler
func getScheduler() *gocron.Scheduler {
	_schedulerOnce.Do(func() {
		_scheduler = newScheduler()
		_scheduler.ChangeLocation(time.Local)
	})
	return _scheduler
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

		inf := JobInf{
			Name:            job.Name,
			Cron:            job.Cron,
			CronWithSeconds: job.CronWithSeconds,
		}

		for _, hook := range preJobHooks {
			if err := hook(rail, inf); err != nil {
				rail.Errorf("PreJobHook returns err for job: %v, %v", job.Name, err)
			}
		}

		if job.LogJobExec {
			rail.Infof("Running job '%s'", job.Name)
		}

		start := time.Now()
		errRun := job.Run(rail)
		took := time.Since(start)
		if errRun == nil {
			if job.LogJobExec {
				rail.Infof("Job '%s' finished, took: %s", job.Name, took)
			}
		} else {
			rail.Errorf("Job '%s' failed, took: %s, %v", job.Name, took, errRun)
		}

		if len(postJobHooks) > 0 {
			stats := JobExecStats{Time: took, Err: errRun}
			for _, hook := range postJobHooks {
				if err := hook(rail, inf, stats); err != nil {
					rail.Errorf("PostJobHook returns err for job: %v, %v", job.Name, err)
				}
			}
		}
	}

	if job.CronWithSeconds {
		_, err = s.CronWithSeconds(job.Cron).Tag(job.Name).Do(wrappedJob)
	} else {
		_, err = s.Cron(job.Cron).Tag(job.Name).Do(wrappedJob)
	}

	if err != nil {
		return fmt.Errorf("failed to schedule cron job, cron: %v, withSeconds: %v, %w", job.Cron, job.CronWithSeconds, err)
	}

	PostServerBootstrap(func(rail Rail) error {
		taggedJobs, err := getScheduler().FindJobsByTag(job.Name)
		if err != nil {
			rail.Warnf("Failed to FindJobsByTag, jobName: %v, %v", job.Name, err)
			return nil
		}

		for _, j := range taggedJobs {
			rail.Debugf("Job '%v' next run scheduled at: %v", job.Name, j.NextRun())
		}

		if job.TriggeredOnBoostrapped {
			if err := getScheduler().RunByTag(job.Name); err != nil {
				rail.Errorf("Failed to triggered immediately on server bootstrapped, jobName: %v, %v", job.Name, err)
			} else {
				rail.Debugf("Job %v triggered on server bootstrapped", job.Name)
			}
		}
		return nil
	})

	return nil
}

// Stop scheduler
func StopScheduler() {
	getScheduler().Stop()
}

// Start scheduler and block current routine
func StartSchedulerBlocking() {
	getScheduler().StartBlocking()
}

// Start scheduler asynchronously
func StartSchedulerAsync() {
	getScheduler().StartAsync()
}

// add a cron job to scheduler, note that the cron expression includes second, e.g., '*/1 * * * * *'
//
// this func doesn't start the scheduler
func ScheduleCron(job Job) error {
	s := getScheduler()
	return doScheduleCron(s, job)
}

// Callback triggered before job execution.
//
// The job and other callbacks will still be executed even if one of the callback returns error.
//
// Callback will be ignored, if the scheduler is already running.
func PreJobExec(hook PreJobHook) {
	if getScheduler().IsRunning() {
		Warn("Ignored PreJobHook, cron scheduler is already running")
		return
	}
	preJobHooks = append(preJobHooks, hook)
}

// Callback triggered after job execution.
//
// Other callbacks will still be executed even if one of them returns error.
//
// Callback will be ignored, if the scheduler is already running.
func PostJobExec(hook PostJobHook) {
	if getScheduler().IsRunning() {
		Warn("Ignored PostJobHook, cron scheduler is already running")
		return
	}
	postJobHooks = append(postJobHooks, hook)
}

func SchedulerBootstrap(rail Rail) error {
	StartSchedulerAsync()
	rail.Info("Cron Scheduler started")
	AddShutdownHook(func() { StopScheduler() })
	return nil
}

// Runner that triggers run on every tick.
//
// Create TickRunner using func NewTickRunner(...).
type TickRunner struct {
	task   func()
	ticker *time.Ticker
	mu     sync.Mutex
	freq   time.Duration
}

func NewTickRuner(freq time.Duration, run func()) *TickRunner {
	return &TickRunner{task: run, freq: freq}
}

func (t *TickRunner) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ticker != nil {
		return
	}

	t.ticker = time.NewTicker(t.freq)
	c := t.ticker.C
	go func() {
		for {
			t.task()
			<-c
		}
	}()
}

func (t *TickRunner) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ticker == nil {
		return
	}
	t.ticker.Stop()
	t.ticker = nil
}
