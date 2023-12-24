package miso

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestTaskScheduling(t *testing.T) {
	rail := EmptyRail()
	LoadConfigFromFile("../conf_dev.yml", rail)
	SetProp(".enabled", "true")
	SetLogLevel("debug")

	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}

	SetScheduleGroup("miso")

	var count int32 = 0
	j := Job{
		Name:            "AddInt32 Task",
		Cron:            "0/1 * * * * ?",
		CronWithSeconds: true,
		Run: func(rail Rail) error {
			atomic.AddInt32(&count, 1)
			rail.Infof("%v", count)
			return nil
		},
	}

	err := ScheduleDistributedTask(j)
	if err != nil {
		t.Fatal(err)
	}

	StartTaskSchedulerAsync(rail)

	time.Sleep(6 * time.Second)

	StopTaskScheduler()

	v := atomic.LoadInt32(&count)
	if v < 5 {
		t.Fatal(v)
	}
	t.Log("end")
}
