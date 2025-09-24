# Distributed Task Scheduling

Miso provides basic cron-based scheduling functionality. It also wraps the cron scheduler to support distributed task scheduling.

A cluster is distinguished by a group name, which by default is `${app.name}`. Each cluster can only have one master node.

Inspired by [github.com/rq/rq](https://github.com/rq/rq), the Producer/Worker model is applied, the master node acts as the **Producer** that is responsible for pushing scheduled tasks to the **Queue** (Redis List). The Queue is shared among the cluster, **Workers** (all the nodes, including the master node) constantly pull tasks from the Queue and run them.

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