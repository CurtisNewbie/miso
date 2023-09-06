package miso

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestRLock(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	c := EmptyRail()
	LoadConfigFromFile("../app-conf-dev.yml", c)
	if _, e := InitRedisFromProp(); e != nil {
		t.Fatal(e)
	}

	var violated int32 = 0

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		wg.Wait()

		err := RLockExec(c, "test:rlock", func() error {
			t.Log("shouldn't be here")
			atomic.StoreInt32(&violated, 1)
			return nil
		})

		if err == nil {
			t.Logf("test failed, condition violated, err == nil")
			atomic.StoreInt32(&violated, 1)
		} else {
			t.Logf("test passed, timed-out %v", err)
		}
	}()

	RLockExec(c, "test:rlock", func() error {
		t.Log("inside lock")
		wg.Done()
		time.Sleep(3 * time.Second)
		return nil
	})

	if atomic.LoadInt32(&violated) == 1 {
		t.Fatal()
	}

}
