package server

import (
	"testing"
	"time"

	"github.com/curtisnewbie/gocommon/common"
)

func TestBootstrapServer(t *testing.T) {
	args := make([]string, 2)

	common.SetProp(common.PROP_APP_NAME, "test-app")

	go func() {
		time.Sleep(5 * time.Second)
		if IsShuttingDown() {
			t.Error()
			return
		}

		// syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		Shutdown()
	}()

	BootstrapServer(args)
}
