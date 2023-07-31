package task

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/redis"
	"github.com/sirupsen/logrus"
)

func TestTaskScheduling(t *testing.T) {
	c := common.EmptyExecContext()
	common.LoadConfigFromFile("../app-conf-dev.yml", c)
	common.SetProp("redis.enabled", "true")

	if _, e := redis.InitRedisFromProp(); e != nil {
		t.Fatal(e)
	}

	SetScheduleGroup("gocommon")

	var count int32 = 0
	ScheduleNamedDistributedTask("0/1 * * * * ?", true, "AddInt32 Task", func(ec common.ExecContext) error {
		atomic.AddInt32(&count, 1)
		logrus.Infof("%v", count)
		return nil
	})

	StartTaskSchedulerAsync()

	time.Sleep(6 * time.Second)

	StopTaskScheduler()

	v := atomic.LoadInt32(&count)
	if v < 5 {
		t.Fatal(v)
	}
	t.Log("end")
}
