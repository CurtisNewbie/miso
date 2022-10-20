package consul

import (
	"testing"

	"github.com/curtisnewbie/gocommon/config"
	"github.com/sirupsen/logrus"
)

func PreTest() *config.Configuration {
	args := make([]string, 2)
	args[0] = "profile=dev"
	args[1] = "configFile=../app-conf-dev.json"
	_, conf := config.DefaultParseProfConf(args)
	return conf
}

func TestPollServiceListInstances(t *testing.T) {

	conf := PreTest()

	_, err := InitConsulClient(conf.ConsulConf)
	if err != nil {
		t.Error(err)
		return
	}

	address, err := ResolveServiceAddress("file-service")
	if err != nil {
		t.Error(err)
		return
	}
	logrus.Infof("(first try) Address resolved: %s", address)

	PollServiceListInstances()
	PollServiceListInstances()
}

func TestResolveServiceAddress(t *testing.T) {
	conf := PreTest()

	_, err := InitConsulClient(conf.ConsulConf)
	if err != nil {
		t.Error(err)
		return
	}

	address, err := ResolveServiceAddress("file-service")
	if err != nil {
		t.Error(err)
		return
	}
	logrus.Infof("(first try) Address resolved: %s", address)

	address, err = ResolveServiceAddress("file-service")
	if err != nil {
		t.Error(err)
		return
	}
	logrus.Infof("(second try) Address resolved: %s", address)

}
