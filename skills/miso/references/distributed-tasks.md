# Distributed Tasks

Cron-based scheduling with master-worker architecture for distributed task execution across clusters.

## Core Concepts

Distributed tasks use a **Producer-Worker** design pattern:

- **Producer (Master Node)**: Scheduled tasks that run on cron schedule and push tasks to the queue
- **Queue (Redis)**: Shared queue using Redis lists (LPUSH/BRPOP)
- **Workers (All Nodes)**: All nodes (including master) pull and execute tasks

Only one master node per cluster at a time (via Redis-based leader election).

## Job Structure

```go
type Job struct {
    Name                   string                // Unique job name
    Cron                   string                // Cron expression
    Run                    func(rail Rail) error // Execution logic
    CronWithSeconds        bool                  // Include seconds in cron (default false)
    LogJobExec             bool                  // Log job execution (errors always logged)
    TriggeredOnBootstraped bool                  // Trigger on server startup
    LogErrWarnLevel        bool                  // Use WARN level for errors (default ERROR)
}
```

## Basic Usage

### Register and Start Tasks

```go
import "github.com/curtisnewbie/miso/middleware/task"

func main() {
    // Set schedule group name (cluster identifier)
    task.SetScheduleGroup("myApp")

    // Register distributed task
    err := task.ScheduleDistributedTask(miso.Job{
        Cron:            "*/15 * * * *",        // Every 15 minutes
        CronWithSeconds: false,                 // Standard 5-field cron
        Name:            "MyDistributedTask",
        Run: func(rail miso.Rail) error {
            return doSomething(rail)
        },
    })
    if err != nil {
        panic(err)
    }

    // Bootstrap server (automatically starts scheduler)
    miso.BootstrapServer(os.Args)
}
```

### Cron Expressions

- **6-field** (with seconds): `"0 */5 * * * *"` (every 5 minutes)
- **5-field** (standard): `"*/15 * * * *"` (every 15 minutes)

## Configuration

```yaml
# conf.yml
task:
  scheduling:
    enabled: true
    group: "${app.name}"  # Cluster group name

# Enable job trigger API endpoints (for manual triggering)
scheduler:
  api:
    trigger:
      job:
        enabled: true
```

### Properties

| Property | Default | Description |
|----------|---------|-------------|
| `task.scheduling.enabled` | `true` | Enable/disable task scheduling |
| `task.scheduling.group` | `${app.name}` | Schedule group name (cluster identifier) |
| `scheduler.api.trigger.job.enabled` | `false` | Enable job trigger API endpoints |

## Key Behaviors

### Master Election

- Cluster identified by group name
- Redis SETNX-based leader election with TTL (60s)
- Master lock automatically refreshed every 5s
- Only master node runs scheduled cron tasks

### Task Execution

- All nodes continuously poll queue (BRPOP with 1s timeout)
- Stale task threshold: 5s (older tasks are ignored)
- Worker pool: Calculated with size range 12-1024 workers
- Tasks logged with timing info

### Bootstrapping

- Tasks with `TriggeredOnBootstraped: true` run once after full bootstrap
- Scheduler starts automatically if `task.scheduling.enabled: true`
- Stops gracefully on shutdown

## Advanced Features

### Task Hooks

```go
// Pre-execution hook
miso.RegisterPreJobHook(func(rail miso.Rail, inf miso.JobInf) error {
    rail.Infof("Job '%s' about to run", inf.Name)
    return nil
})

// Post-execution hook
miso.RegisterPostJobHook(func(rail miso.Rail, inf miso.JobInf, stats miso.JobExecStats) error {
    rail.Infof("Job '%s' completed in %s", inf.Name, stats.Time)
    if stats.Err != nil {
        // Handle error
    }
    return nil
})
```

### Manual Task Triggering

```go
// Trigger distributed task by name via API
GET /debug/task/trigger?name=MyDistributedTask
```

### Debugging Tools

```go
// Disable task workers (for debugging)
POST /debug/task/disable-workers
{
    "tasks": ["TaskName1", "TaskName2"]  // Use "*" for all tasks
}

// Enable task workers
POST /debug/task/enable-workers
{
    "tasks": ["TaskName1", "TaskName2"]
}
```

## Import

Required middleware import:

```go
import _ "github.com/curtisnewbie/miso/middleware/task"
```

The middleware registers bootstrap callbacks automatically.

## Performance Considerations

- Each task worker occupies one Redis connection
- With 30+ tasks and low GOMAXPROCS, consider increasing Redis pool size
- Stale task threshold prevents executing old tasks from queue
- Worker pool sized automatically based on system resources

## Differences from Local Scheduling

| Feature | Local (`ScheduleCron`) | Distributed (`ScheduleDistributedTask`) |
|---------|----------------------|------------------------------------------|
| Execution | Single node only | All nodes in cluster |
| Master election | No | Yes (Redis-based) |
| Queue | No | Yes (Redis list) |
| Scalability | Limited | Horizontal |
| Dependencies | None | Redis |

## Example: Data Sync Job

```go
err := task.ScheduleDistributedTask(miso.Job{
    Name:       "SyncUserData",
    Cron:       "0 0 2 * * ?",  // 2:00 AM daily
    Run: func(rail miso.Rail) error {
        var users []User
        if err := dbquery.GetDB().Find(&users).Error; err != nil {
            return errs.WrapErr(err, "failed to fetch users")
        }

        for _, user := range users {
            if err := syncToExternalService(rail, user); err != nil {
                rail.ErrorIf(err, "failed to sync user %s", user.ID)
            }
        }
        return nil
    },
    LogJobExec: true,
})
```

## References

- Core docs: `/Users/zhuangyongjie/dev/git/miso/doc/dtask.md`
- Implementation: `/Users/zhuangyongjie/dev/git/miso/middleware/task/task.go`
- Schedule types: `/Users/zhuangyongjie/dev/git/miso/miso/schedule.go`