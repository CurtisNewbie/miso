package redis

import (
	"testing"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/sirupsen/logrus"
)

func TestRLock(t *testing.T) {
	c := common.EmptyExecContext()
	c.Log.Logger.SetLevel(logrus.DebugLevel)
	common.LoadConfigFromFile("../app-conf-dev.yml", c)
	if _, e := InitRedisFromProp(); e != nil {
		t.Fatal(e)
	}

	RLockExec(c, "test:rlock", func() error {
		time.Sleep(22 * time.Second)
		return nil
	})
}
