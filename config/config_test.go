package config

import (
	"testing"

	"github.com/curtisnewbie/gocommon/test"
	"github.com/sirupsen/logrus"
)

func TestParseProfile(t *testing.T) {

	args := make([]string, 2)
	args[0] = "profile=abc"
	args[1] = "--someflag"

	profile := ParseProfile(args)
	if profile != "abc" {
		t.Errorf("Expected abc, but got: %v", profile)
	}

	args2 := make([]string, 1)
	args2[0] = "--someflag"

	profile = ParseProfile(args2)
	if profile != "dev" {
		t.Errorf("Expected dev, but got: %v", profile)
	}
}

func TestDefaultParseProfConf(t *testing.T) {

	args := make([]string, 2)
	args[0] = "profile=dev"
	args[1] = "configFile=../app-conf-dev.json"
	profile, conf := DefaultParseProfConf(args)
	if profile != "dev" {
		t.Errorf("Profile incorrect, %s", profile)
		return
	}

	if conf == nil {
		t.Errorf("conf is nil, %+v", conf)
	}
}

func TestResolveArgForParsedConf(t *testing.T) {
	SetEnv("DB_USER", "root")
	SetEnv("DB_PASSWORD", "123456")
	SetEnv("DB_DATABASE", "fileServer")
	SetEnv("DB_HOST", "localhost")
	SetEnv("DB_PORT", "3306")
	SetEnv("REDIS_ADD", "localhost")
	SetEnv("REDIS_PORT", "6379")
	SetEnv("REDIS_USERNAME", "admin")
	SetEnv("REDIS_PASSWORD", "654321")
	SetEnv("SERVER_HOST", "localhost")
	SetEnv("SERVER_PORT", "8081")
	SetEnv("FILE_BASE", "/tmp/base")
	SetEnv("FILE_TEMP", "/tmp/temp")
	SetEnv("CONSUL_REGNAME", "test-service")
	SetEnv("CONSUL_ADD", "localhost:8500")
	SetEnv("CONSUL_HC_URL", "/some/health")
	SetEnv("CONSUL_HC_ITV", "5s")
	SetEnv("CONSUL_HC_TO", "5s")
	SetEnv("CONSUL_HC_DEREG_AFT", "30s")
	SetEnv("CLIENT_FS", "http://localhost:8080")
	SetEnv("CLIENT_AS", "http://localhost:8081")

	args := make([]string, 2)
	args[0] = "profile=dev"
	args[1] = "configFile=../app-conf-test.json"
	_, conf := DefaultParseProfConf(args)

	test.TestEqual(t, conf.DBConf.User, "root")
	test.TestEqual(t, conf.DBConf.Password, "123456")
	test.TestEqual(t, conf.DBConf.Database, "fileServer")
	test.TestEqual(t, conf.DBConf.Host, "localhost")
	test.TestEqual(t, conf.DBConf.Port, "3306")

	test.TestEqual(t, conf.RedisConf.Address, "localhost")
	test.TestEqual(t, conf.RedisConf.Port, "6379")
	test.TestEqual(t, conf.RedisConf.Username, "admin")
	test.TestEqual(t, conf.RedisConf.Password, "654321")

	test.TestEqual(t, conf.ServerConf.Host, "localhost")
	test.TestEqual(t, conf.ServerConf.Port, "8081")

	test.TestEqual(t, conf.FileConf.Base, "/tmp/base")
	test.TestEqual(t, conf.FileConf.Temp, "/tmp/temp")

	test.TestEqual(t, conf.ConsulConf.RegisterName, "test-service")
	test.TestEqual(t, conf.ConsulConf.ConsulAddress, "localhost:8500")
	test.TestEqual(t, conf.ConsulConf.HealthCheckUrl, "/some/health")
	test.TestEqual(t, conf.ConsulConf.HealthCheckInterval, "5s")
	test.TestEqual(t, conf.ConsulConf.HealthCheckFailedDeregisterAfter, "30s")

	test.TestEqual(t, conf.ClientConf.FileServiceUrl, "http://localhost:8080")
	test.TestEqual(t, conf.ClientConf.AuthServiceUrl, "http://localhost:8081")
}

func TestResolveArg(t *testing.T) {
	SetEnv("abc", "123")
	resolved := ResolveArg("${abc}")
	if resolved != "123" {
		t.Errorf("resolved is not '%s' but '%s'", "123", resolved)
		return
	}
	logrus.Infof("resolved: %s", resolved)
}
