package miso

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/util/async"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/strutil"
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
	BeforeWebRouteRegister(func(rail Rail) error {
		registerRouteForJobTriggers()
		return nil
	})
}

type Job struct {
	Name                   string                // name of the job.
	Cron                   string                // cron expr.
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
	return async.PanicSafeFunc(func() {
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
			if errors.Is(errRun, ErrServerShuttingDown) {
				rail.Warnf("Job '%s' failed, took: %s, %v", job.Name, took, errRun)
			} else {
				rail.Errorf("Job '%s' failed, took: %s, %v", job.Name, took, errRun)
			}
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
	if m.guessCronWithSceond(job.Cron) {
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
				rail.Errorf("Failed to trigger immediately on server bootstrapped, jobName: %v, %v", job.Name, err)
			} else {
				rail.Infof("Job '%v' triggered on server bootstrapped", job.Name)
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

func LogJobNextRun(rail Rail, jobName string) {
	scheduleModule().logNextRun(rail, jobName, false)
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

// Trigger Named Job.
func TriggerJob(rail Rail, name string) error {
	if strutil.IsBlankStr(name) {
		return errs.NewErrf("Job name is empty")
	}

	m := scheduleModule()
	if err := m.scheduler.RunByTag(name); err != nil {
		rail.Errorf("Failed to triggered job, jobName: %v, %v", name, err)
		return err
	} else {
		rail.Debugf("Job '%v' triggered", name)
		return nil
	}
}

// enable api to manually trigger jobs
func registerRouteForJobTriggers() {
	if !GetPropBool(PropSchedApiTriggerJobEnabled) {
		return
	}

	HttpGet("/debug/job/trigger", RawHandler(func(inb *Inbound) {
		rail := inb.Rail()
		name := inb.Query("name")
		err := TriggerJob(rail, name)
		if err == nil {
			rail.Infof("Triggered job %v through api", name)
		}
		inb.HandleResult(nil, err)
	})).DocQueryParam("name", "job name").Desc("Manually Trigger Cron Job By Name")
}
