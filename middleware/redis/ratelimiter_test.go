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
	for i := range 10 {
		aw.SubmitAsync(func() (int, error) {
			if rl.Acquire() {
				return i, nil
			}
			return -1, miso.NewErrf("Rate limited")
		})
	}
	fut := aw.Await()
	anyErr := false
	for i, f := range fut {
		v, err := f.Get()
		t.Logf("%v - %v, %v", i, v, err)
		if err != nil {
			anyErr = true
		}
	}
	if !anyErr {
		t.Fatal("should rate limit")
	}
}
