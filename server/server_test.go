package server

import (
	"syscall"
	"testing"
	"time"
)

func TestBootstrapServer(t *testing.T) {
	args := make([]string, 2)
	args[0] = "profile=dev"
	args[1] = "configFile=../app-conf-dev.yml"

	go func() {
		time.Sleep(5 * time.Second)
		if IsShuttingDown() {
			t.Error()
			return
		}

		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	DefaultBootstrapServer(args)
}
