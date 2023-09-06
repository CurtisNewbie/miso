package miso

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduleCron(t *testing.T) {
	var yoc int32 = 0
	var noc int32 = 0

	t.Log("Yo")
	err := ScheduleCron("*/1 * * * * *", true, func() {
		time.Sleep(1 * time.Second)
		atomic.AddInt32(&yoc, 1)
		t.Log("Yo")
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ScheduleCron("*/1 * * * * *", true, func() {
		time.Sleep(1 * time.Second)
		atomic.AddInt32(&noc, 1)
		t.Log("No")
	})
	if err != nil {
		t.Fatal(err)
	}

	StartSchedulerAsync()

	time.Sleep(10 * time.Second)

	StopScheduler()

	if atomic.LoadInt32(&yoc) < 1 {
		t.Error(yoc)
	}
	if atomic.LoadInt32(&noc) < 1 {
		t.Error(noc)
	}
	t.Logf("yoc: %v, noc: %v", atomic.LoadInt32(&yoc), atomic.LoadInt32(&noc))

}
