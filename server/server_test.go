package server

import (
	"syscall"
	"testing"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/task"
	"github.com/sirupsen/logrus"
)

func TestBootstrapServer(t *testing.T) {
	args := make([]string, 2)
	args[0] = "profile=dev"
	args[1] = "configFile=../app-conf-dev.yml"

	task.ScheduleDistributedTask("0/1 * * * * ?", true, func(ec common.ExecContext) {
		logrus.Info("feels gucci")
	})

	go func() {
		time.Sleep(5 * time.Second)
		if IsShuttingDown() {
			t.Error()
			return
		}

		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	BootstrapServer(args)
}
