# Distributed Task Scheduling

Miso provides basic cron-based scheduling functionality. It also wraps the cron scheduler to support distributed task scheduling. A cluster is distinguished by a group name, which by default is `${app.name}`. Each cluster of nodes can only have one master, and the master node is responsible for running all the tasks. If there is any distributed task registered in an app, it will try to obtain the master lock on startup. You can think of it as a combination of cron scheduler and redis based lock.

```go
func main() {
    // set the group name
    miso.SetScheduleGroup("myApp")

    // add task
    err = miso.ScheduleDistributedTask(miso.Job{
        Cron:            "*/15 * * * *",
        CronWithSeconds: false,
        Name:            "MyDistributedTask",
        Run: func(miso miso.Rail) error {
            return jobDoSomething(rail)
        },
    })
    if err != nil {
        panic(err) // for demo only
    }

    // start task scheduler
    miso.StartTaskSchedulerAsync()

    // stop task scheduler gracefully
    defer miso.StopTaskScheduler()
}
```

The code above is automatically handled by `miso.BootstrapServer(...)` func.

```go
func main() {
    // add tasks
    err = miso.ScheduleDistributedTask(miso.Job{
        Cron:            "*/15 * * * *",
        CronWithSeconds: false,
        Name:            "MyDistributedTask",
        Run: func(miso miso.Rail) error {
            return jobDoSomething(rail)
        },
    })
    if err != nil {
        panic(err) // for demo only
    }

    // bootstrap server
    miso.BootstrapServer(os.Args)
}
```