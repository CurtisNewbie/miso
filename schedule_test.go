package gocommon

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduleCron(t *testing.T) {
	var c int32 = 0

	t.Log("Yo")
	s := ScheduleCron("*/1 * * * * *", func() {
		atomic.AddInt32(&c, 1)

		t.Log("Yo")
	})
	s.StartAsync()

	time.Sleep(2*time.Second)

	if atomic.LoadInt32(&c) < 1 {
		t.Error(c)
	}
}
