package task

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/curtisnewbie/miso/middleware/redis"
	"github.com/curtisnewbie/miso/miso"
)

func TestTaskScheduling(t *testing.T) {
	rail := miso.EmptyRail()
	miso.LoadConfigFromFile("../conf_dev.yml", rail)
	miso.SetProp(".enabled", "true")
	miso.SetLogLevel("debug")

	if _, e := redis.InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}

	SetScheduleGroup("miso")

	var count int32 = 0
	j := miso.Job{
		Name:            "AddInt32 Task",
		Cron:            "0/1 * * * * ?",
		CronWithSeconds: true,
		Run: func(rail miso.Rail) error {
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
