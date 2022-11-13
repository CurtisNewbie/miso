package gocommon

import (
	"syscall"
	"testing"
	"time"
)

func TestBootstrapServer(t *testing.T) {
	args := make([]string, 2)
	args[0] = "profile=dev"
	args[1] = "configFile=app-conf-dev.json"
	DefaultReadConfig(args)

	go func() {
		time.Sleep(5*time.Second)
		if IsShuttingDown() {
			t.Error()
			return
		}

		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	BootstrapServer()
}
