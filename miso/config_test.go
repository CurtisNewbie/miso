package miso

import (
	"bytes"
	"testing"
)

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

	resolved = ResolveArg("abc.${def:321:123}.com")
	if resolved != "abc.321:123.com" {
		t.Fatal(resolved)
		return
	}
	Infof("resolved: %s", resolved)

	resolved = ResolveArg("abc.${def:123-456}.com")
	if resolved != "abc.123-456.com" {
		t.Fatal(resolved)
		return
	}
	Infof("resolved: %s", resolved)

	resolved = ResolveArg("abc.${def:123_456}.com")
	if resolved != "abc.123_456.com" {
		t.Fatal(resolved)
		return
	}
	Infof("resolved: %s", resolved)

	resolved = ResolveArg("abc.${def:123/456}.com")
	if resolved != `abc.123/456.com` {
		t.Fatal(resolved)
		return
	}
	Infof("resolved: %s", resolved)

	resolved = ResolveArg("abc.${def:123.456}.com")
	if resolved != `abc.123.456.com` {
		t.Fatal(resolved)
		return
	}
	Infof("resolved: %s", resolved)

	resolved = ResolveArg("abc.${def}.com")
	if resolved != `abc..com` {
		t.Fatal(resolved)
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

/*
func TestAlias(t *testing.T) {
	SetProp("v1", true)
	RegisterAlias("v2", "v1")
	t.Logf("v1: %v, v2: %v", GetPropStr("v1"), GetPropStr("v2"))

	v := GetPropBool("v2")
	if !v {
		t.Fatalf("'%v'", v)
	}
	t.Logf("v1: %v, v2: %v", GetPropStr("v1"), GetPropStr("v2"))

	SetProp("v1", false)
	t.Logf("v1: %v, v2: %v", GetPropStr("v1"), GetPropStr("v2"))

	v = GetPropBool("v2")
	if v {
		t.Fatal(v)
	}
	t.Logf("v1: %v, v2: %v", GetPropStr("v1"), GetPropStr("v2"))

	SetProp("v2", true)
	t.Logf("v1: %v, v2: %v", GetPropStr("v1"), GetPropStr("v2"))

	v = GetPropBool("v2")
	if !v {
		t.Fatal(v)
	}
	t.Logf("v1: %v, v2: %v", GetPropStr("v1"), GetPropStr("v2"))

	v = GetPropBool("v1")
	if !v {
		t.Fatal(v)
	}
	t.Logf("v1: %v, v2: %v", GetPropStr("v1"), GetPropStr("v2"))

	SetDefProp("v3", "333")
	RegisterAlias("v4", "v3")
	s := GetPropStr("v4")
	if s != "333" {
		t.Fatal(s)
	}
	t.Logf("v3: %v, v4: %v", GetPropStr("v3"), GetPropStr("v4"))

	SetProp("v4", "444")
	s = GetPropStr("v4")
	if s != "444" {
		t.Fatal(s)
	}
	t.Logf("v3: %v, v4: %v", GetPropStr("v3"), GetPropStr("v4"))

	SetProp("v3", "555")
	s = GetPropStr("v3")
	if s != "555" {
		t.Fatal(s)
	}
	s = GetPropStr("v4")
	if s != "555" {
		t.Fatal(s)
	}
	t.Logf("v3: %v, v4: %v", GetPropStr("v3"), GetPropStr("v4"))

	SetProp("level.v5", "333")
	RegisterAlias("v6", "level.v5")
	s = GetPropStr("v6")
	if s != "333" {
		t.Fatal(s)
	}
	t.Logf("level.v5: %v, v6: %v", GetPropStr("level.v5"), GetPropStr("v6"))

	SetProp("v6", "444")
	s = GetPropStr("v6")
	if s != "444" {
		t.Fatal(s)
	}
	t.Logf("level.v5: %v, v6: %v", GetPropStr("level.v5"), GetPropStr("v6"))

	SetProp("level.v5", "555")
	s = GetPropStr("level.v5")
	if s != "555" {
		t.Fatal(s)
	}
	s = GetPropStr("v6")
	if s != "555" {
		t.Fatal(s)
	}
	t.Logf("level.v5: %v, v6: %v", GetPropStr("level.v5"), GetPropStr("v6"))
}
*/

func TestGetParentNodeAsAsSlice(t *testing.T) {
	err := LoadConfigFromStr(`
test:
  node-a:
    name: "a"
  node-b:
    name: "b"
  node-c:
    name: "c"
`, EmptyRail())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", GetPropAny("test"))
	for i, v := range GetPropChild("test") {
		t.Logf("%v, %v", i, v)
	}
}
