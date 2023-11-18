package miso

import (
	"testing"
	"time"
)

type RCacheDummy struct {
	Name string
	Age  int
}

func preRCacheTest(t *testing.T) Rail {
	rail := EmptyRail()
	SetProp(PropRedisEnabled, true)
	rail.SetLogLevel("debug")
	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}
	return rail
}

func TestLazyObjRcache(t *testing.T) {
	rail := preRCacheTest(t)
	exp := 10 * time.Second
	invokeCount := 0
	supplier := func(rail Rail, _ string) (RCacheDummy, error) {
		invokeCount++
		rail.Infof("Called supplier, %v", invokeCount)
		return RCacheDummy{
			Name: "Banana",
			Age:  12,
		}, nil
	}

	cache := NewLazyORCache("test", supplier, RCacheConfig{Exp: exp})
	cache.Del(rail, "1")

	dummy, err := cache.Get(rail, "1")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	rail.Infof("1. got from supplier %+v, invokeCount: %v", dummy, invokeCount)
	if invokeCount != 1 {
		t.Logf("invokeCount: %v", invokeCount)
		t.FailNow()
	}

	dummy, err = cache.Get(rail, "1")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	rail.Infof("2. got from cache %+v, invokeCount: %v", dummy, invokeCount)

	if invokeCount != 1 {
		t.Logf("invokeCount: %v", invokeCount)
		t.FailNow()
	}

	cache.Del(rail, "1")

	dummy, err = cache.Get(rail, "1")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if invokeCount != 2 {
		t.Logf("invokeCount: %v", invokeCount)
		t.FailNow()
	}

	rail.Infof("3. got from supplier %+v, invokeCount: %v", dummy, invokeCount)
}

func TestRCache(t *testing.T) {
	rail := preRCacheTest(t)
	exp := 10 * time.Second
	rcache := NewRCache("test", nil, RCacheConfig{Exp: exp})

	_, e := rcache.Get(rail, "absent key")
	if e == nil || !IsNoneErr(e) {
		t.Fatal(e)
	}

	e = rcache.Put(rail, "1", "3")
	if e != nil {
		t.Fatal(e)
	}

	var val string
	val, e = rcache.Get(rail, "1")
	if e != nil {
		t.Fatal(e)
	}
	if val != "3" {
		t.Fatalf("val '%v' != \"3\"", val)
	}
}

func TestLazyRCache(t *testing.T) {
	rail := preRCacheTest(t)

	exp := 10 * time.Second

	rcache := NewLazyRCache("test",
		func(rail Rail, key string) (string, error) {
			return "", NoneErr
		},
		RCacheConfig{Exp: exp},
	)

	e := rcache.Put(rail, "1", "2")
	if e != nil {
		t.Fatal(e)
	}

	val, e := rcache.Get(rail, "1")
	if e != nil {
		t.Fatal(e)
	}
	if val != "2" {
		t.Fatalf("val '%v' != \"2\"", val)
	}
}
