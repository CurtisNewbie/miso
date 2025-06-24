package redis

import (
	"testing"
	"time"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
)

func TestRateLimiter(t *testing.T) {
	rail := miso.EmptyRail()
	miso.LoadConfigFromFile("../../testdata/conf_dev.yml", rail)
	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}
	miso.SetLogLevel("debug")

	rl := NewRateLimiter("abc", 3, time.Second*1)
	aw := util.NewAwaitFutures[int](nil)
	for range 3 {
		for i := range 30 {
			aw.SubmitAsync(func() (int, error) {
				ok, err := rl.Acquire()
				if err != nil {
					t.Fatal(err)
				}
				if ok {
					return i, nil
				}
				return -1, miso.NewErrf("Rate limited")
			})
		}
		time.Sleep(time.Second)
	}
	fut := aw.Await()
	anyErr := false
	sum := 0
	for i, f := range fut {
		v, err := f.Get()
		t.Logf("%v - %v, %v", i, v, err)
		if err != nil {
			anyErr = true
		} else {
			sum += 1
		}
	}
	if !anyErr {
		t.Fatal("should rate limit")
	}
	t.Logf("rate: ~ %v/s", sum/3)
}
