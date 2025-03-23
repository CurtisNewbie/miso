package redis

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/curtisnewbie/miso/miso"
)

func TestRLockCallbacks(t *testing.T) {
	rail := miso.EmptyRail()
	miso.LoadConfigFromFile("../../testdata/conf_dev.yml", rail)
	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}
	miso.SetLogLevel("debug")

	var violated int32 = 0

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		wg.Wait()

		err := RLockExec(rail, "test:rlock", func() error {
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

	RLockExec(rail, "test:rlock", func() error {
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
	rail := miso.EmptyRail()
	miso.LoadConfigFromFile("../../testdata/conf_dev.yml", rail)
	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}
	miso.SetLogLevel("trace")

	var violated int32 = 0
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		wg.Wait()

		lock := NewCustomRLock(rail, "test:rlock", RLockConfig{BackoffDuration: 1 * time.Second})
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
	rail.Infof("main - released lock")

	if atomic.LoadInt32(&violated) == 1 {
		t.Fatal()
	}
}

func TestRLockCount(t *testing.T) {
	miso.SetLogLevel("debug")

	lockRefreshTime = time.Millisecond

	rail := miso.EmptyRail()
	miso.LoadConfigFromFile("../../testdata/conf_dev.yml", rail)
	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}
	miso.SetLogLevel("debug")

	total := 1000
	var count int32 = 0

	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lock := NewCustomRLock(rail, "test:rlock", RLockConfig{
				BackoffDuration: time.Second * 5,
			})

			err := lock.Lock()
			if err != nil {
				t.Logf("failed to obtain lock, %v", err)
				return
			}
			defer lock.Unlock()

			time.Sleep(time.Millisecond * 3)
			atomic.StoreInt32(&count, atomic.LoadInt32(&count)+1)
		}()
	}
	wg.Wait()

	actual := atomic.LoadInt32(&count)
	if actual != 1000 {
		t.Fatalf("incorrect count, actual: %v", actual)
	}
}

func TestCancelRefresher(t *testing.T) {
	rail := miso.EmptyRail()
	miso.LoadConfigFromFile("../../testdata/conf_dev.yml", rail)
	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}
	miso.SetLogLevel("trace")

	lockRefreshTime = time.Second
	lock := NewRLock(rail, "mylock")
	if err := lock.Lock(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Second)
	lock.Unlock()
	time.Sleep(5 * time.Second)
}

func TestRLockTryLock(t *testing.T) {
	rail := miso.EmptyRail()
	miso.LoadConfigFromFile("../../testdata/conf_dev.yml", rail)
	if _, e := InitRedisFromProp(rail); e != nil {
		miso.Error(e)
		t.Fatal(e)
	}
	miso.SetLogLevel("trace")

	var violated int32 = 0
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		wg.Wait()

		lock := NewCustomRLock(rail, "test:rlock", RLockConfig{BackoffDuration: 1 * time.Second})
		rail.Infof("routine - routine attempts to get lock")
		start := time.Now()

		ok, err := lock.TryLock()
		if err != nil {
			t.Logf("TryLock err != nil, %v", err)
			atomic.StoreInt32(&violated, 1)
			return
		}
		if ok {
			rail.Infof("routine - test failed, condition violated, ok")
			atomic.StoreInt32(&violated, 1)
		} else {
			rail.Infof("routine - test passed, timed-out, %v, attempted for %v", err, time.Since(start))
		}
	}()

	lock := NewRLock(rail, "test:rlock")
	ok, err := lock.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("not ok")
	}

	rail.Infof("main - inside lock")
	wg.Done()
	time.Sleep(3 * time.Second)

	if err := lock.Unlock(); err != nil {
		t.Fatal(err)
	}
	rail.Infof("main - released lock")

	if atomic.LoadInt32(&violated) == 1 {
		t.Fatal()
	}
}
