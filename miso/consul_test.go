package miso

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func PreTest() {
	args := make([]string, 2)
	args[0] = "profile=dev"
	args[1] = "configFile=../conf_dev.yml"
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
	PollServiceListInstances(rail)

	address, err := ConsulResolveServiceAddr("vfm")
	if err != nil {
		t.Error(err)
		return
	}
	logrus.Infof("Address resolved: %s", address)

	resolved, err := ConsulResolveRequestUrl("vfm", "/file")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Log(resolved)
	if resolved != "http://"+address+"/file" {
		t.FailNow()
	}
}

func TestResolveServiceAddress(t *testing.T) {
	PreTest()
	SetLogLevel("debug")

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
