package main

import (
	"os"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
	"github.com/curtisnewbie/gocommon/task"
)

func main() {
	// add tasks
	task.ScheduleDistributedTask("0 0/15 * * * *", func(c common.ExecContext) {
	})

	// bootstrap server
	server.DefaultBootstrapServer(os.Args, common.EmptyExecContext())
}
