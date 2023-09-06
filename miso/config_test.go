package miso

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestParseProfile(t *testing.T) {

	args := make([]string, 2)
	args[0] = "profile=abc"
	args[1] = "--someflag"

	profile := GuessProfile(args)
	if profile != "abc" {
		t.Errorf("Expected abc, but got: %v", profile)
	}

	args2 := make([]string, 1)
	args2[0] = "--someflag"

	profile = GuessProfile(args2)
	if profile != "dev" {
		t.Errorf("Expected dev, but got: %v", profile)
	}
}

func TestParseArg(t *testing.T) {
	args := make([]string, 2)
	args[0] = "profile=dev"
	args[1] = "configFile=../app-conf-dev.yml"
	DefaultReadConfig(args, EmptyRail())

	if m := GetPropBool(PROP_PRODUCTION_MODE); !m {
		t.Error(m)
	}
	if !IsProdMode() {
		t.Error()
	}
	if s := GetPropStr(PROP_MYSQL_USER); s != "root" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_MYSQL_PASSWORD); s != "123456" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_MYSQL_DATABASE); s != "fileServer" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_MYSQL_HOST); s != "localhost" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_MYSQL_PORT); s != "3306" {
		t.Error(s)
	}

	if s := GetPropBool(PROP_REDIS_ENABLED); s {
		t.Error(s)
	}
	if s := GetPropStr(PROP_REDIS_ADDRESS); s != "localhost" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_REDIS_PORT); s != "6379" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_REDIS_USERNAME); s != "" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_REDIS_PASSWORD); s != "" {
		t.Error(s)
	}

	if s := GetPropStr(PROP_SERVER_HOST); s != "localhost" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_SERVER_PORT); s != "8081" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC); s != "5" {
		t.Error(s)
	}

	if s := GetPropStr("file.base"); s != "test-base" {
		t.Error(s)
	}
	if s := GetPropStr("file.temp"); s != "temp" {
		t.Error(s)
	}

	if s := GetPropStr(PROP_CONSUL_REGISTER_NAME); s != "test-service" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_CONSUL_CONSUL_ADDRESS); s != "localhost:8500" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_CONSUL_HEALTHCHECK_URL); s != "/some/health" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_CONSUL_HEALTHCHECK_INTERVAL); s != "5s" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_CONSUL_HEALTHCHECK_TIMEOUT); s != "5s" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_CONSUL_HEALTHCHECK_FAILED_DEREG_AFTER); s != "30s" {
		t.Error(s)
	}

	if s := GetPropStr("client.fileServiceUrl"); s != "http://localhost:8080" {
		t.Error(s)
	}
	if s := GetPropStr("client.authServiceUrl"); s != "http://localhost:8081" {
		t.Error(s)
	}

	if sl := GetPropStringSlice("rabbitmq.declaration.queue"); len(sl) != 2 {
		t.Error(sl)
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
	args[1] = "configFile=../app-conf-test.yml"
	DefaultReadConfig(args, EmptyRail())

	t.Logf("PRODUCTION MODE: %t", GetPropBool(PROP_PRODUCTION_MODE))
	t.Logf("Is PROD MODE: %t", IsProdMode())

	if m := GetPropBool(PROP_PRODUCTION_MODE); m {
		t.Error(m)
	}
	if IsProdMode() {
		t.Error()
	}
	if s := GetPropStr(PROP_MYSQL_USER); s != "root" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_MYSQL_PASSWORD); s != "123456" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_MYSQL_DATABASE); s != "fileServer" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_MYSQL_HOST); s != "localhost" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_MYSQL_PORT); s != "3306" {
		t.Error(s)
	}

	if s := GetPropStr(PROP_REDIS_ADDRESS); s != "localhost" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_REDIS_PORT); s != "6379" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_REDIS_USERNAME); s != "admin" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_REDIS_PASSWORD); s != "654321" {
		t.Error(s)
	}

	if s := GetPropStr(PROP_SERVER_HOST); s != "localhost" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_SERVER_PORT); s != "8081" {
		t.Error(s)
	}

	if s := GetPropStr("file.base"); s != "/tmp/base" {
		t.Error(s)
	}
	if s := GetPropStr("file.temp"); s != "/tmp/temp" {
		t.Error(s)
	}

	if s := GetPropStr(PROP_CONSUL_REGISTER_NAME); s != "test-service" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_CONSUL_CONSUL_ADDRESS); s != "localhost:8500" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_CONSUL_HEALTHCHECK_URL); s != "/some/health" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_CONSUL_HEALTHCHECK_INTERVAL); s != "5s" {
		t.Error(s)
	}
	if s := GetPropStr(PROP_CONSUL_HEALTHCHECK_FAILED_DEREG_AFTER); s != "30s" {
		t.Error(s)
	}

	if s := GetPropStr("client.fileServiceUrl"); s != "http://localhost:8080" {
		t.Error(s)
	}
	if s := GetPropStr("client.authServiceUrl"); s != "http://localhost:8081" {
		t.Error(s)
	}
}

func TestResolveArg(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	SetEnv("abc", "123")
	resolved := ResolveArg("${abc}")
	if resolved != "123" {
		t.Errorf("resolved is not '%s' but '%s'", "123", resolved)
		return
	}
	logrus.Infof("resolved: %s", resolved)

	resolved = ResolveArg("${abc}.com")
	if resolved != "123.com" {
		t.Errorf("resolved is not '%s' but '%s'", "123.com", resolved)
		return
	}
	logrus.Infof("resolved: %s", resolved)

	resolved = ResolveArg("abc.${abc}.com")
	if resolved != "abc.123.com" {
		t.Errorf("resolved is not '%s' but '%s'", "abc.123.com", resolved)
		return
	}
	logrus.Infof("resolved: %s", resolved)
}

func TestArgKeyVal(t *testing.T) {
	kv := ArgKeyVal([]string{"fruit=apple", "content=juice"})
	v, ok := kv["fruit"]
	if !ok {
		t.Fatal("kv doesn't contain fruit")
	}
	if v != "apple" {
		t.Fatal("value should be apple")
	}
	t.Logf("%+v", v)
}
