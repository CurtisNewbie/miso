# Distributed Tasks

Cron-based scheduling with master-worker architecture for distributed task execution across clusters.

## Core Concepts

Distributed tasks use a **Producer-Worker** design pattern:

- **Producer (Master Node)**: Scheduled tasks that run on cron schedule and push tasks to the queue
- **Queue (Redis)**: Shared queue using Redis lists (LPUSH/BRPopAny)
- **Workers (All Nodes)**: All nodes (including master) pull and execute tasks

Only one master node per cluster at a time (via Redis-based leader election).

### Trace Propagation

Tasks include trace context for distributed tracing:

```go
type queuedTask struct {
    Name        string    // Task name
    ScheduledAt atom.Time // When task was scheduled (UTC)
    Producer    string    // Producer node identifier (IP:port)
    TraceId     string    // Trace ID for distributed tracing
}
```

When a producer schedules a task, it includes the current trace ID. Workers pick up the trace context when executing tasks, enabling end-to-end tracing across the distributed system.

### Concurrent Execution Prevention

Each task execution uses a distributed Redis lock (`RLock`) to prevent concurrent execution of the same task across multiple nodes:

- Lock key format: `dtask:{group}:{taskName}`
- Lock timeout with backoff: 1 second
- If lock cannot be acquired, the task is skipped (producer may be outpacing workers)

This ensures idempotency and prevents resource conflicts when tasks overlap.

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

See [config.md](https://github.com/CurtisNewbie/miso/blob/main/doc/config.md) for distributed task scheduling configuration properties.

## Key Behaviors

### Master Election

- Cluster identified by group name
- Redis SETNX-based leader election with TTL (60s)
- Master lock automatically refreshed every 5s
- Only master node runs scheduled cron tasks

### Task Execution

- Single goroutine pulls tasks from all queues using `BRPopAny` with 10s timeout
- Stale task threshold: 5s (older tasks are ignored)
- Worker pool: Calculated with `async.CalcPoolSize(12, 128, 1024)` based on system resources
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

- **Single goroutine architecture**: Uses one BRPopAny call to pull from all task queues, requiring only ONE Redis connection regardless of task count
- **Efficient polling**: BRPopAny blocks up to 10s, reducing CPU usage compared to continuous polling
- **Worker pool sizing**: Automatically calculated with `async.CalcPoolSize(12, 128, 1024)` based on system resources
- **Stale task threshold**: 5s threshold prevents executing old tasks from queue
- **Distributed locks**: Each task uses Redis lock (1s backoff) to prevent concurrent execution

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