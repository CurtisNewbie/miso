package miso

import (
	"bytes"
	"testing"
)

func TestParseArg(t *testing.T) {
	args := make([]string, 2)
	args[1] = "configFile=../testdata/conf_dev.yml"
	DefaultReadConfig(args, EmptyRail())

	if m := GetPropBool(PropProdMode); !m {
		t.Error(m)
	}
	if !IsProdMode() {
		t.Error()
	}

	if s := GetPropStr(PropServerHost); s != "localhost" {
		t.Error(s)
	}
	if s := GetPropStr(PropServerPort); s != "8081" {
		t.Error(s)
	}
	if s := GetPropStr(PropServerGracefulShutdownTimeSec); s != "5" {
		t.Error(s)
	}

	if s := GetPropStr("file.base"); s != "test-base" {
		t.Error(s)
	}
	if s := GetPropStr("file.temp"); s != "temp" {
		t.Error(s)
	}

	if s := GetPropStr(PropConsuleRegisterName); s != "test-service" {
		t.Error(s)
	}
	if s := GetPropStr(PropConsulAddress); s != "localhost:8500" {
		t.Error(s)
	}
	if s := GetPropStr(PropConsulHealthCheckInterval); s != "5s" {
		t.Error(s)
	}
	if s := GetPropStr(PropConsulHealthcheckTimeout); s != "5s" {
		t.Error(s)
	}
	if s := GetPropStr(PropConsulHealthCheckFailedDeregAfter); s != "30s" {
		t.Error(s)
	}

	if s := GetPropStr("client.fileServiceUrl"); s != "http://localhost:8080" {
		t.Error(s)
	}
	if s := GetPropStr("client.authServiceUrl"); s != "http://localhost:8081" {
		t.Error(s)
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
	args[1] = "configFile=../testdata/conf_test.yml"
	DefaultReadConfig(args, EmptyRail())

	t.Logf("PRODUCTION MODE: %t", GetPropBool(PropProdMode))
	t.Logf("Is PROD MODE: %t", IsProdMode())

	if m := GetPropBool(PropProdMode); m {
		t.Error(m)
	}
	if IsProdMode() {
		t.Error()
	}

	if s := GetPropStr(PropServerHost); s != "localhost" {
		t.Error(s)
	}
	if s := GetPropStr(PropServerPort); s != "8081" {
		t.Error(s)
	}

	if s := GetPropStr("file.base"); s != "/tmp/base" {
		t.Error(s)
	}
	if s := GetPropStr("file.temp"); s != "/tmp/temp" {
		t.Error(s)
	}

	if s := GetPropStr(PropConsuleRegisterName); s != "test-service" {
		t.Error(s)
	}
	if s := GetPropStr(PropConsulAddress); s != "localhost:8500" {
		t.Error(s)
	}
	if s := GetPropStr(PropConsulHealthCheckInterval); s != "5s" {
		t.Error(s)
	}
	if s := GetPropStr(PropConsulHealthCheckFailedDeregAfter); s != "30s" {
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
	SetLogLevel("debug")
	SetEnv("abc", "123")
	resolved := ResolveArg("${abc}")
	if resolved != "123" {
		t.Errorf("resolved is not '%s' but '%s'", "123", resolved)
		return
	}
	Infof("resolved: %s", resolved)

	resolved = ResolveArg("${abc}.com")
	if resolved != "123.com" {
		t.Errorf("resolved is not '%s' but '%s'", "123.com", resolved)
		return
	}
	Infof("resolved: %s", resolved)

	resolved = ResolveArg("abc.${abc}.com")
	if resolved != "abc.123.com" {
		t.Errorf("resolved is not '%s' but '%s'", "abc.123.com", resolved)
		return
	}
	Infof("resolved: %s", resolved)
}

func TestArgKeyVal(t *testing.T) {
	kv := ArgKeyVal([]string{"fruit=apple", "content=juice", "content=jay"})
	v, ok := kv["fruit"]
	if !ok {
		t.Fatal("kv doesn't contain fruit")
	}
	if len(v) < 1 || v[0] != "apple" {
		t.Fatal("value should be apple")
	}
	t.Logf("%+v", v)

	v, ok = kv["content"]
	if !ok || len(v) < 2 || v[0] != "juice" || v[1] != "jay" {
		t.Fatalf("value should be juice, jay, but: %v", v)
	}
	t.Logf("%+v", v)
}

func BenchmarkGetProbool(b *testing.B) {
	args := make([]string, 2)
	args[1] = "configFile=../conf_dev.yml"
	DefaultReadConfig(args, EmptyRail())
	SetProp("correct_type", true)

	slowGetPropBool := func(prop string) bool {
		return returnWithReadLock(globalConfig(), func() bool { return globalConfig().vp.GetBool(prop) })
	}

	b.Run("GetPropBool_correct_type", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			GetPropBool("correct_type")
		}
	})
	b.Run("slowGetPropBool_correct_type", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slowGetPropBool("correct_type")
		}
	})

	SetProp("incorrect_type", "true")
	b.Run("GetPropBool_incorrect_type", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			GetPropBool("incorrect_type")
		}
	})
	b.Run("slowGetPropBool_incorrect_type", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slowGetPropBool("incorrect_type")
		}
	})

	SetProp("incorrect_type_2", "nope")
	b.Run("GetPropBool_incorrect_type_2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			GetPropBool("incorrect_type_2")
		}
	})
	b.Run("slowGetPropBool_incorrect_type_2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slowGetPropBool("incorrect_type_2")
		}
	})
}

func TestLoadConfigFromReader(t *testing.T) {
	SetDefProp("switch", false)
	b := bytes.NewReader([]byte(`
switch: "true"
test: "TestLoadConfigFromReader"
`))
	if err := LoadConfigFromReader(b, EmptyRail()); err != nil {
		t.Fatal(err)
	}
	if !GetPropBool("switch") {
		t.Fatal("should be true")
	}
	if GetPropStr("test") != "TestLoadConfigFromReader" {
		t.Fatal("incorrect test value")
	}
}

func TestLoadConfigFromStr(t *testing.T) {
	SetDefProp("switch", false)
	s := `
switch: "true"
test: "TestLoadConfigFromReader"
`
	if err := LoadConfigFromStr(s, EmptyRail()); err != nil {
		t.Fatal(err)
	}
	if !GetPropBool("switch") {
		t.Fatal("should be true")
	}
	if GetPropStr("test") != "TestLoadConfigFromReader" {
		t.Fatal("incorrect test value")
	}
}

func TestPropSlice(t *testing.T) {
	SetProp("test", "apple,  orange, juice")
	v := GetPropStrSlice("test")
	t.Logf("%#v", v)
	if len(v) != 3 {
		t.Fatal("len != 3")
	}
}
