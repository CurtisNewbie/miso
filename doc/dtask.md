# Distributed Task Scheduling

Miso provides basic cron-based scheduling functionality. It also wraps the cron scheduler to support distributed task scheduling. A cluster is distinguished by a group name, each cluster of nodes can only have one master, and the master node is responsible for running all the tasks.

```go
func main() {
    // set the group name
    miso.SetScheduleGroup("myApp")

    // add task
    miso.ScheduleDistributedTask("0/1 * * * * ?", true, func(rail miso.Rail) {
        // ...
    })

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
    miso.ScheduleDistributedTask("0 0/15 * * * *", true, func(rail miso.Rail) {
    })

    // bootstrap server
    miso.BootstrapServer(os.Args)
}
```