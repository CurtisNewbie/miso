package miso

import (
	"testing"

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

	_, err := GetConsulClient()
	if err != nil {
		t.Error(err)
		return
	}

	address, err := ConsulResolveServiceAddr("file-service")
	if err != nil {
		t.Error(err)
		return
	}
	logrus.Infof("(first try) Address resolved: %s", address)

	PollServiceListInstances()
}

func TestResolveServiceAddress(t *testing.T) {
	PreTest()

	_, err := GetConsulClient()
	if err != nil {
		t.Error(err)
		return
	}

	address, err := ConsulResolveServiceAddr("file-service")
	if err != nil {
		t.Error(err)
		return
	}
	logrus.Infof("(first try) Address resolved: %s", address)

	address, err = ConsulResolveServiceAddr("file-service")
	if err != nil {
		t.Error(err)
		return
	}
	logrus.Infof("(second try) Address resolved: %s", address)

}
