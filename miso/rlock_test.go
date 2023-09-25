package miso

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestRLockCallbacks(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	c := EmptyRail()
	LoadConfigFromFile("../app-conf-dev.yml", c)
	if _, e := InitRedisFromProp(c); e != nil {
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

func TestRLock(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	rail := EmptyRail()
	LoadConfigFromFile("../app-conf-dev.yml", rail)
	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}
	rail.SetLogLevel("debug")

	var violated int32 = 0

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		wg.Wait()

		lock := NewRLock(rail, "test:rlock")
		rail.Infof("routine - routine attempts to get lock")
		start := time.Now()

		err := lock.Lock()
		if err == nil {
			rail.Infof("routine - test failed, condition violated, err == nil")
			atomic.StoreInt32(&violated, 1)
		} else {
			rail.Infof("routine - test passed, timed-out, %v, attempted for %v", err, time.Since(start))
		}
	}()

	lock := NewRLock(rail, "test:rlock")
	err := lock.Lock()
	if err != nil {
		t.Fatal(err)
	}

	rail.Infof("main - inside lock")
	wg.Done()
	time.Sleep(3 * time.Second)

	if err := lock.Unlock(); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&violated) == 1 {
		t.Fatal()
	}
}

func TestRLockCount(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	rail := EmptyRail()
	LoadConfigFromFile("../app-conf-dev.yml", rail)
	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}
	rail.SetLogLevel("debug")

	total := 1000
	var count int32 = 0

	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lock := NewCustomRLock(rail, "test:rlock", RLockConfig{
				BackoffDuration: time.Second * 2,
			})

			err := lock.Lock()
			if err != nil {
				t.Logf("failed to obtain lock, %v", err)
				return
			}
			defer lock.Unlock()

			atomic.StoreInt32(&count, atomic.LoadInt32(&count)+1)
		}()
	}
	wg.Wait()

	actual := atomic.LoadInt32(&count)
	if actual != 1000 {
		t.Fatalf("incorrect count, actual: %v", actual)
	}
}
