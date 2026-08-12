package miso

import (
	"bytes"
	"fmt"
	"testing"
	"time"
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

	resolved = ResolveArg("abc.${def: 123_456 }.com")
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
	t.Logf("1. %#v", v)
	if len(v) != 3 {
		t.Fatal("len != 3")
	}

	SetProp("test", []string{"apple", "orange", "juice"})
	v = GetPropStrSlice("test")
	t.Logf("2. %#v", v)
	v[0] = "ah"
	t.Logf("3. %#v", v)
	t.Logf("4. %#v", GetPropStrSlice("test"))
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
parent-node-test:
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
	m := GetPropAny("parent-node-test")
	t.Logf("< %+v", m)
	m.(map[string]any)["node-a"] = "wut"
	t.Logf("> %+v", m)
	t.Logf("<> %+v", GetPropAny("parent-node-test"))
	for i, v := range GetPropChild("parent-node-test") {
		t.Logf("%v, %v", i, v)
	}
}

func TestPropKeyCache(t *testing.T) {
	err := LoadConfigFromStr(`
prop-cache-test:
  name: "v1"
  count: 1
`, EmptyRail())
	if err != nil {
		t.Fatal(err)
	}

	type Cfg struct {
		Name  string
		Count int
	}

	c := NewRefreshedCache[Cfg](50*time.Millisecond, func() Cfg {
		return UnmarshalFromPropKeyAs[Cfg]("prop-cache-test")
	})
	defer c.Stop()

	// value loaded on creation
	v := c.Get()
	if v.Name != "v1" || v.Count != 1 {
		t.Fatalf("unexpected: %+v", v)
	}

	// update config, cache should refresh within refreshEvery
	SetProp("prop-cache-test.name", "v2")
	SetProp("prop-cache-test.count", 2)

	deadline := time.Now().Add(3 * time.Second)
	for {
		v = c.Get()
		if v.Name == "v2" && v.Count == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("cache not refreshed in time: %+v", v)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestPropKeyCacheNoRefresh(t *testing.T) {
	err := LoadConfigFromStr(`
prop-cache-test-no-refresh:
  name: "v1"
`, EmptyRail())
	if err != nil {
		t.Fatal(err)
	}

	type Cfg struct {
		Name string
	}

	// refreshEvery <= 0, no background refresh
	c := NewRefreshedCache[Cfg](0, func() Cfg {
		return UnmarshalFromPropKeyAs[Cfg]("prop-cache-test-no-refresh")
	})
	defer c.Stop()

	SetProp("prop-cache-test-no-refresh.name", "v2")

	time.Sleep(100 * time.Millisecond)
	if v := c.Get(); v.Name != "v1" {
		t.Fatalf("expected stale value 'v1', got '%s'", v.Name)
	}
}

func TestPropFunc(t *testing.T) {
	ac := newAppConfig()

	// Test 1: RegisterPropFunc + GetPropStr resolves
	ac.RegisterPropFunc("encrypt", func(arg string) (string, error) {
		return "encrypted:" + arg, nil
	})
	ac.SetProp("secret", "encrypt(mysecret)")
	resolved := ac.GetPropStr("secret")
	if resolved != "encrypted:mysecret" {
		t.Fatalf("expected 'encrypted:mysecret', got '%s'", resolved)
	}
	t.Logf("Test 1 passed: resolved = %s", resolved)

	// Test 2: Non-matching value returns as-is
	ac.SetProp("plain", "plaintext")
	plain := ac.GetPropStr("plain")
	if plain != "plaintext" {
		t.Fatalf("expected 'plaintext', got '%s'", plain)
	}
	t.Logf("Test 2 passed: plain = %s", plain)

	// Test 3: Error returns empty string
	ac.RegisterPropFunc("fail", func(arg string) (string, error) {
		return "", fmt.Errorf("intentional error")
	})
	ac.SetProp("failprop", "fail(something)")
	failResult := ac.GetPropStr("failprop")
	if failResult != "" {
		t.Fatalf("expected empty string on error, got '%s'", failResult)
	}
	t.Logf("Test 3 passed: error returns empty string")

	// Test 4: UnmarshalFromPropKey resolves nested string fields
	type NestedConfig struct {
		Name   string
		Secret string
		Inner  struct {
			Value string
		}
	}
	err := ac.LoadConfigFromStr(`
nested:
  name: "normal"
  secret: "encrypt(nested-secret)"
  inner:
    value: "encrypt(inner-value)"
`)
	if err != nil {
		t.Fatal(err)
	}
	var cfg NestedConfig
	ac.UnmarshalFromPropKey("nested", &cfg)
	if cfg.Name != "normal" {
		t.Fatalf("expected 'normal', got '%s'", cfg.Name)
	}
	if cfg.Secret != "encrypted:nested-secret" {
		t.Fatalf("expected 'encrypted:nested-secret', got '%s'", cfg.Secret)
	}
	if cfg.Inner.Value != "encrypted:inner-value" {
		t.Fatalf("expected 'encrypted:inner-value', got '%s'", cfg.Inner.Value)
	}
	t.Logf("Test 4 passed: nested struct resolved, cfg = %+v", cfg)

	// Test 5: Multiple registered funcs
	ac.RegisterPropFunc("vault", func(arg string) (string, error) {
		return "vault:" + arg, nil
	})
	ac.RegisterPropFunc("decrypt", func(arg string) (string, error) {
		return "decrypted:" + arg, nil
	})
	ac.SetProp("vault-secret", "vault(secret/path)")
	ac.SetProp("decrypt-secret", "decrypt(cipher)")
	vaultResult := ac.GetPropStr("vault-secret")
	if vaultResult != "vault:secret/path" {
		t.Fatalf("expected 'vault:secret/path', got '%s'", vaultResult)
	}
	decryptResult := ac.GetPropStr("decrypt-secret")
	if decryptResult != "decrypted:cipher" {
		t.Fatalf("expected 'decrypted:cipher', got '%s'", decryptResult)
	}
	t.Logf("Test 5 passed: multiple funcs work, vault = %s, decrypt = %s", vaultResult, decryptResult)

	// Test 6: UnmarshalFromPropKey for slice of structs resolves prop funcs in elements
	type ChatAppConfig struct {
		Name      string
		AppSecret string
		Inner     struct {
			Value string
		}
	}
	err = ac.LoadConfigFromStr(`
chat-apps:
  - name: "app-a"
    app-secret: "encrypt(app-a-secret)"
    inner:
      value: "encrypt(app-a-inner)"
  - name: "app-b"
    app-secret: "encrypt(app-b-secret)"
`)
	if err != nil {
		t.Fatal(err)
	}
	var apps []ChatAppConfig
	ac.UnmarshalFromPropKey("chat-apps", &apps)
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}
	if apps[0].Name != "app-a" || apps[0].AppSecret != "encrypted:app-a-secret" {
		t.Fatalf("unexpected apps[0]: %+v", apps[0])
	}
	if apps[0].Inner.Value != "encrypted:app-a-inner" {
		t.Fatalf("unexpected apps[0].Inner: %+v", apps[0].Inner)
	}
	if apps[1].Name != "app-b" || apps[1].AppSecret != "encrypted:app-b-secret" {
		t.Fatalf("unexpected apps[1]: %+v", apps[1])
	}
	t.Logf("Test 6 passed: slice of structs resolved, apps = %+v", apps)

	// Test 7: map[string]Struct and []map[string]string resolve prop funcs
	type AppCfg struct {
		Secret string
	}
	err = ac.LoadConfigFromStr(`
app-map:
  a:
    secret: "encrypt(a-secret)"
  b:
    secret: "encrypt(b-secret)"
props-list:
  - name: "encrypt(list-name)"
    token: "encrypt(list-token)"
  - name: "plain-name"
`)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]AppCfg
	ac.UnmarshalFromPropKey("app-map", &m)
	if m["a"].Secret != "encrypted:a-secret" || m["b"].Secret != "encrypted:b-secret" {
		t.Fatalf("unexpected map resolution: %+v", m)
	}
	var l []map[string]string
	ac.UnmarshalFromPropKey("props-list", &l)
	if len(l) != 2 || l[0]["name"] != "encrypted:list-name" || l[0]["token"] != "encrypted:list-token" || l[1]["name"] != "plain-name" {
		t.Fatalf("unexpected slice-of-map resolution: %+v", l)
	}
	t.Logf("Test 7 passed: map[string]struct + []map[string]string resolved, m = %+v, l = %+v", m, l)

	// Test 8: regression coverage for resolver shapes: []string / map[string]string
	// struct fields, pointer fields, []*T, map[string]*T and [][]string
	type Inner struct {
		Token string
	}
	type AppCfg2 struct {
		Name    string
		Aliases []string
		Labels  map[string]string
		Inner   *Inner
	}
	err = ac.LoadConfigFromStr(`
shapes:
  name: "shapes-name"
  aliases: ["encrypt(alias-1)", "plain-alias"]
  labels:
    k1: "encrypt(label-1)"
    k2: "plain-label"
  inner:
    token: "encrypt(ptr-token)"
apps-ptr:
  - name: "app-x"
    inner:
      token: "encrypt(x-token)"
ptr-map:
  x:
    name: "app-y"
    inner:
      token: "encrypt(y-token)"
nested-strs:
  - ["encrypt(n1)", "plain-n2"]
  - ["encrypt(n3)"]
`)
	if err != nil {
		t.Fatal(err)
	}
	var shapes AppCfg2
	ac.UnmarshalFromPropKey("shapes", &shapes)
	if shapes.Name != "shapes-name" {
		t.Fatalf("unexpected shapes.Name: %+v", shapes)
	}
	if len(shapes.Aliases) != 2 || shapes.Aliases[0] != "encrypted:alias-1" || shapes.Aliases[1] != "plain-alias" {
		t.Fatalf("unexpected shapes.Aliases: %+v", shapes.Aliases)
	}
	if shapes.Labels["k1"] != "encrypted:label-1" || shapes.Labels["k2"] != "plain-label" {
		t.Fatalf("unexpected shapes.Labels: %+v", shapes.Labels)
	}
	if shapes.Inner == nil || shapes.Inner.Token != "encrypted:ptr-token" {
		t.Fatalf("unexpected shapes.Inner: %+v", shapes.Inner)
	}
	var apps2 []*AppCfg2
	ac.UnmarshalFromPropKey("apps-ptr", &apps2)
	if len(apps2) != 1 || apps2[0].Name != "app-x" || apps2[0].Inner == nil || apps2[0].Inner.Token != "encrypted:x-token" {
		t.Fatalf("unexpected apps2: %+v", apps2)
	}
	var pm map[string]*AppCfg2
	ac.UnmarshalFromPropKey("ptr-map", &pm)
	if pm["x"] == nil || pm["x"].Name != "app-y" || pm["x"].Inner == nil || pm["x"].Inner.Token != "encrypted:y-token" {
		t.Fatalf("unexpected ptr-map: %+v", pm)
	}
	var ns [][]string
	ac.UnmarshalFromPropKey("nested-strs", &ns)
	if len(ns) != 2 || ns[0][0] != "encrypted:n1" || ns[0][1] != "plain-n2" || ns[1][0] != "encrypted:n3" {
		t.Fatalf("unexpected nested-strs: %+v", ns)
	}
	t.Logf("Test 8 passed: resolver shapes verified, shapes = %+v, apps2 = %+v, pm = %+v, ns = %+v", shapes, apps2, pm, ns)
}
