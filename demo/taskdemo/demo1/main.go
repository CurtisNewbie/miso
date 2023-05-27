package main

import (
	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/task"
)

func main() {
	// set the group name
	task.SetScheduleGroup("gocommon")

	// add task
	task.ScheduleDistributedTask("0/1 * * * * ?", func(c common.ExecContext) {
		// ...
	})

	// start task scheduler
	task.StartTaskSchedulerAsync()

	// stop task scheduler gracefully
	defer task.StopTaskScheduler()
}
