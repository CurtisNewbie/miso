package miso

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduleCron(t *testing.T) {
	var yoc int32 = 0
	var noc int32 = 0

	t.Log("Yo")

	err := ScheduleCron(Job{
		Name:            "myjob",
		Cron:            "*/1 * * * * *",
		CronWithSeconds: true,
		Run: func(rail Rail) error {
			time.Sleep(1 * time.Second)
			atomic.AddInt32(&yoc, 1)
			t.Log("Yo")
			return nil
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	err = ScheduleCron(Job{
		Name:            "myjob",
		Cron:            "*/1 * * * * *",
		CronWithSeconds: true,
		Run: func(rail Rail) error {
			time.Sleep(1 * time.Second)
			atomic.AddInt32(&noc, 1)
			t.Log("No")
			return nil
		},
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

func TestJobListener(t *testing.T) {
	PreJobExec(func(rail Rail, inf JobInf) error {
		rail.Infof("Pre job execution, name: %v", inf.Name)
		return errors.New("should still run pre")
	})

	PostJobExec(func(rail Rail, jobInf JobInf, stats JobExecStats) error {
		rail.Infof("post 1 job execution, name: %v, err: %v, took: %v", jobInf.Name, stats.Err, stats.Time)
		return errors.New("should still run post 2")
	})

	PostJobExec(func(rail Rail, jobInf JobInf, stats JobExecStats) error {
		rail.Infof("post 2 job execution, name: %v, err: %v, took: %v", jobInf.Name, stats.Err, stats.Time)
		return nil
	})

	err := ScheduleCron(Job{
		Name:            "myjob",
		Cron:            "*/1 * * * * *",
		CronWithSeconds: true,
		Run: func(rail Rail) error {
			time.Sleep(500 * time.Millisecond)
			t.Log("Yo")
			return nil
		},
	})

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	StartSchedulerAsync()

	time.Sleep(1 * time.Second)

	StopScheduler()
}
