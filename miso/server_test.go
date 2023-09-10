package miso

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestBootstrapServer(t *testing.T) {
	args := make([]string, 2)
	logrus.SetLevel(logrus.DebugLevel)

	SetProp(PROP_APP_NAME, "test-app")

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

func TestPostServerBootstrapCallback(t *testing.T) {
	i := 0

	PostServerBootstrapped(func(rail Rail) error {
		i++ // 1
		if i != 1 {
			t.Fatalf("incorrect order i is %v not 1", i)
		}
		return nil
	})

	PostServerBootstrapped(func(rail Rail) error {
		PostServerBootstrapped(func(rail Rail) error {
			i++ // 3
			if i != 3 {
				t.Fatalf("incorrect order i is %v not 3", i)
			}
			return nil
		})
		return nil
	})

	PostServerBootstrapped(func(rail Rail) error {
		i++ // 2
		if i != 2 {
			t.Fatalf("incorrect order i is %v not 2", i)
		}
		return nil
	})

	rail := EmptyRail()
	callPostServerBootstrapListeners(rail)

	if i != 3 {
		t.Fatalf("i is not 3, but %v", i)
	}
}

func TestPreServerBootstrapCallback(t *testing.T) {
	i := 0

	PreServerBootstrap(func(rail Rail) error {
		i++ // 1
		if i != 1 {
			t.Fatalf("incorrect order i is %v not 1", i)
		}
		return nil
	})

	PreServerBootstrap(func(rail Rail) error {
		PreServerBootstrap(func(rail Rail) error {
			i++ // 3
			if i != 3 {
				t.Fatalf("incorrect order i is %v not 3", i)
			}
			return nil
		})
		return nil
	})

	PreServerBootstrap(func(rail Rail) error {
		i++ // 2
		if i != 2 {
			t.Fatalf("incorrect order i is %v not 2", i)
		}
		return nil
	})

	rail := EmptyRail()
	callPreServerBootstrapListeners(rail)

	if i != 3 {
		t.Fatalf("i is not 3, but %v", i)
	}
}