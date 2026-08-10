package task

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/curtisnewbie/miso/middleware/redis"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/hash"
	"github.com/curtisnewbie/miso/util/slutil"
)

func TestPullTasksAnyAllDisabled(t *testing.T) {
	m := &taskModule{
		dtasks:        slutil.NewSyncSlice[miso.Job](1),
		disabledTasks: hash.NewStrRWMap[struct{}](),
	}
	m.dtasks.Append(miso.Job{Name: "disabled"})
	m.disabledTasks.Put("disabled", struct{}{})

	started := time.Now()
	if err := m.pullTasksAny(miso.EmptyRail()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed < taskPollBackoff {
		t.Fatalf("all-disabled poll returned too quickly: %v", elapsed)
	}
}

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
		Name: "AddInt32 Task",
		Cron: "0/1 * * * * ?",
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
