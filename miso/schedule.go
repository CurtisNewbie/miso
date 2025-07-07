package miso

import (
	"fmt"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/util"
	"github.com/go-co-op/gocron"
	"github.com/robfig/cron"
)

var scheduleModule = InitAppModuleFunc(func() *scheduleMdoule {
	return &scheduleMdoule{
		scheduler: gocron.NewScheduler(time.Local),
	}
})
var (
	cronWithSecParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
)

func init() {
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Cron Scheduler",
		Condition: schedulerBootstrapCondition,
		Bootstrap: schedulerBootstrap,
	})
}

type Job struct {
	Name                   string                // name of the job.
	Cron                   string                // cron expr.
	CronWithSeconds        bool                  // Deprecated: since v0.2.2, this is set by miso. This field is left for backward compatibility only.
	Run                    func(rail Rail) error // actual job execution logic.
	LogJobExec             bool                  // should job execution be logged, error msg is always logged regardless.
	TriggeredOnBoostrapped bool                  // should job be triggered when server is fully bootstrapped

	concRunMutex *sync.Mutex
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

func (m *scheduleMdoule) wrapJob(job Job) func() {
	if job.concRunMutex == nil {
		job.concRunMutex = &sync.Mutex{}
	}
	return util.PanicSafeFunc(func() {
		if ok := job.concRunMutex.TryLock(); !ok {
			return
		}
		defer job.concRunMutex.Unlock()

		rail := EmptyRail()

		inf := JobInf{
			Name: job.Name,
			Cron: job.Cron,
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
				m.logNextRun(rail, job.Name, false)
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
	})
}

func (m *scheduleMdoule) guessCronWithSceond(cronExpr string) bool {
	_, err := cronWithSecParser.Parse(cronExpr)
	return err == nil
}

func (m *scheduleMdoule) doScheduleCron(job Job) error {
	var err error
	s := m.scheduler
	wrappedJob := m.wrapJob(job)
	job.CronWithSeconds = m.guessCronWithSceond(job.Cron)
	if job.CronWithSeconds {
		_, err = s.CronWithSeconds(job.Cron).Tag(job.Name).Do(wrappedJob)
	} else {
		_, err = s.Cron(job.Cron).Tag(job.Name).Do(wrappedJob)
	}
	if err != nil {
		return fmt.Errorf("failed to schedule cron job, cron: %v, %w", job.Cron, err)
	}

	OnAppReady(func(rail Rail) error {
		m.logNextRun(rail, job.Name, true)
		if job.TriggeredOnBoostrapped {
			if err := m.scheduler.RunByTag(job.Name); err != nil {
				rail.Errorf("Failed to triggered immediately on server bootstrapped, jobName: %v, %v", job.Name, err)
			} else {
				rail.Debugf("Job '%v' triggered on server bootstrapped", job.Name)
			}
		}
		return nil
	})

	return nil
}

func (m *scheduleMdoule) logNextRun(rail Rail, jobName string, debug bool) {
	taggedJobs, _ := m.scheduler.FindJobsByTag(jobName)
	for _, j := range taggedJobs {
		if debug {
			rail.Debugf("Job '%v' next run scheduled at: %v", jobName, j.NextRun())
		} else {
			rail.Infof("Job '%v' next run scheduled at: %v", jobName, j.NextRun())
		}
	}
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
	PostServerBootstrap(func(rail Rail) error {
		rail.Info("Cron Scheduler started")
		m.startAsync()
		AddAsyncShutdownHook(func() { m.stop() })
		return nil
	})

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

// Deprecated: since v0.2.2, please migrate to [CronExprEveryXSec] instead.
func CronEveryXSec(n int, options ...func(j Job) Job) Job {
	j := Job{
		Cron: CronExprEveryXSec(n),
	}
	return buildCronJob(j, options...)
}

// Deprecated: since v0.2.2, please migrate to [CronExprEveryXMin] instead.
func CronEveryXMin(n int, options ...func(j Job) Job) Job {
	j := Job{
		Cron: CronExprEveryXMin(n),
	}
	return buildCronJob(j, options...)
}

// Deprecated: since v0.2.2, please migrate to [CronExprEveryXHour] instead.
func CronEveryXHour(n int, options ...func(j Job) Job) Job {
	j := Job{
		Cron: CronExprEveryXHour(n),
	}
	return buildCronJob(j, options...)
}

func buildCronJob(j Job, options ...func(j Job) Job) Job {
	for _, op := range options {
		j = op(j)
	}
	return j
}

// Build cron expression (with sceond field) for every X seconds.
func CronExprEveryXSec(n int) string {
	return fmt.Sprintf("*/%d * * * * *", n)
}

// Build cron expression (with sceond field) for every X minutes.
func CronExprEveryXMin(n int) string {
	return fmt.Sprintf("0 */%d * * * *", n)
}

// Build cron expression (with sceond field) for every X hours.
func CronExprEveryXHour(n int) string {
	return fmt.Sprintf("0 0 */%d * * *", n)
}
