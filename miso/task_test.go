package miso

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestTaskScheduling(t *testing.T) {
	c := EmptyRail()
	LoadConfigFromFile("../app-conf-dev.yml", c)
	SetProp(".enabled", "true")

	if _, e := InitRedisFromProp(c); e != nil {
		t.Fatal(e)
	}

	SetScheduleGroup("miso")

	var count int32 = 0
	err := ScheduleNamedDistributedTask("0/1 * * * * ?", true, "AddInt32 Task", func(ec Rail) error {
		atomic.AddInt32(&count, 1)
		logrus.Infof("%v", count)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	StartTaskSchedulerAsync()

	time.Sleep(6 * time.Second)

	StopTaskScheduler()

	v := atomic.LoadInt32(&count)
	if v < 5 {
		t.Fatal(v)
	}
	t.Log("end")
}
