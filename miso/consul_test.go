package miso

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func PreTest() {
	args := make([]string, 2)
	args[0] = "profile=dev"
	args[1] = "configFile=../app-conf-dev.yml"
	DefaultReadConfig(args, EmptyRail())
}

func TestPollServiceListInstances(t *testing.T) {
	PreTest()
	rail := EmptyRail()

	_, err := GetConsulClient()
	if err != nil {
		t.Error(err)
		return
	}

	address, err := ConsulResolveServiceAddr("vfm")
	if err != nil {
		t.Error(err)
		return
	}
	logrus.Infof("(first try) Address resolved: %s", address)

	PollServiceListInstances(rail)
}

func TestResolveServiceAddress(t *testing.T) {
	PreTest()
	rail := EmptyRail()
	rail.SetLogLevel("debug")

	_, err := GetConsulClient()
	if err != nil {
		t.Error(err)
		return
	}

	address, err := ConsulResolveServiceAddr("vfm")
	logrus.Infof("(first try) Address resolved: %s, %v", address, err)

	time.Sleep(1 * time.Second)

	address, err = ConsulResolveServiceAddr("vfm")
	if err != nil {
		t.Error(err)
		return
	}
	logrus.Infof("(second try) Address resolved: %s", address)

}
