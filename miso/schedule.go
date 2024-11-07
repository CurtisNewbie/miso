package miso

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
)

var scheduleModule = InitAppModuleFunc(func() *scheduleMdoule {
	return &scheduleMdoule{
		scheduler: gocron.NewScheduler(time.Local),
	}
})

func init() {
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Cron Scheduler",
		Condition: schedulerBootstrapCondition,
		Bootstrap: schedulerBootstrap,
	})
}

type Job struct {
	Name                   string           // name of the job.
	Cron                   string           // cron expr.
	CronWithSeconds        bool             // whether cron expr contains the second field.
	Run                    func(Rail) error // actual job execution logic.
	LogJobExec             bool             // should job execution be logged, error msg is always logged regardless.
	TriggeredOnBoostrapped bool             // should job be triggered when server is fully bootstrapped
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

type scheduleMdoule struct {
	scheduler *gocron.Scheduler

	preJobHooks  []PreJobHook
	postJobHooks []PostJobHook
}

func (m *scheduleMdoule) stop() {
	m.scheduler.Stop()
}

func (m *scheduleMdoule) doScheduleCron(job Job) error {
	var err error
	s := m.scheduler

	wrappedJob := func() {
		rail := EmptyRail()

		inf := JobInf{
			Name:            job.Name,
			Cron:            job.Cron,
			CronWithSeconds: job.CronWithSeconds,
		}

		for _, hook := range m.preJobHooks {
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

		if len(m.postJobHooks) > 0 {
			stats := JobExecStats{Time: took, Err: errRun}
			for _, hook := range m.postJobHooks {
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
		taggedJobs, err := m.scheduler.FindJobsByTag(job.Name)
		if err != nil {
			rail.Warnf("Failed to FindJobsByTag, jobName: %v, %v", job.Name, err)
			return nil
		}

		for _, j := range taggedJobs {
			rail.Debugf("Job '%v' next run scheduled at: %v", job.Name, j.NextRun())
		}

		if job.TriggeredOnBoostrapped {
			if err := m.scheduler.RunByTag(job.Name); err != nil {
				rail.Errorf("Failed to triggered immediately on server bootstrapped, jobName: %v, %v", job.Name, err)
			} else {
				rail.Debugf("Job %v triggered on server bootstrapped", job.Name)
			}
		}
		return nil
	})

	return nil
}

// Start scheduler and block current routine
func (m *scheduleMdoule) startBlocking() {
	m.scheduler.StartBlocking()
}

func (m *scheduleMdoule) startAsync() {
	m.scheduler.StartAsync()
}
func (m *scheduleMdoule) preJobExec(hook PreJobHook) {
	if m.scheduler.IsRunning() {
		Warn("Ignored PreJobHook, cron scheduler is already running")
		return
	}
	m.preJobHooks = append(m.preJobHooks, hook)
}
func (m *scheduleMdoule) postJobExec(hook PostJobHook) {
	if m.scheduler.IsRunning() {
		Warn("Ignored PostJobHook, cron scheduler is already running")
		return
	}
	m.postJobHooks = append(m.postJobHooks, hook)
}

func (m *scheduleMdoule) hasScheduledJobs() bool {
	return m.scheduler.Len() > 0
}

// Whether scheduler is initialized
func HasScheduledJobs() bool {
	return scheduleModule().hasScheduledJobs()
}

// Stop scheduler
func StopScheduler() {
	scheduleModule().stop()
}

// Start scheduler and block current routine
func StartSchedulerBlocking() {
	scheduleModule().startBlocking()
}

// Start scheduler asynchronously
func StartSchedulerAsync() {
	scheduleModule().startAsync()
}

// add a cron job to scheduler, note that the cron expression includes second, e.g., '*/1 * * * * *'
//
// this func doesn't start the scheduler
func ScheduleCron(job Job) error {
	return scheduleModule().doScheduleCron(job)
}

// Callback triggered before job execution.
//
// The job and other callbacks will still be executed even if one of the callback returns error.
//
// Callback will be ignored, if the scheduler is already running.
func PreJobExec(hook PreJobHook) {
	scheduleModule().preJobExec(hook)
}

// Callback triggered after job execution.
//
// Other callbacks will still be executed even if one of them returns error.
//
// Callback will be ignored, if the scheduler is already running.
func PostJobExec(hook PostJobHook) {
	scheduleModule().postJobExec(hook)
}

func schedulerBootstrapCondition(rail Rail) (bool, error) {
	return scheduleModule().hasScheduledJobs(), nil
}

func schedulerBootstrap(rail Rail) error {
	m := scheduleModule()
	m.startAsync()
	rail.Info("Cron Scheduler started")
	AddShutdownHook(func() { m.stop() })
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
