package miso

import (
	"testing"
	"time"
)

type RCacheDummy struct {
	Name string
	Age  int
}

func TestLazyObjRcache(t *testing.T) {
	rail := EmptyRail()
	LoadConfigFromFile("../app-conf-dev.yml", rail)
	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}
	keypre := "test:lazy:obj:rcache:key:"
	exp := 60 * time.Second
	cache := NewLazyObjectRCache[RCacheDummy](exp)

	supplier := func() (RCacheDummy, bool, error) {
		rail.Info("Called supplier")
		return RCacheDummy{
			Name: "Banana",
			Age:  12,
		}, true, nil
	}

	cache.Del(rail, keypre+"1")

	dummy, ok, err := cache.GetElse(rail, keypre+"1", supplier)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("!ok")
	}
	rail.Infof("1. %+v", dummy)

	dummy, ok, err = cache.GetElse(rail, keypre+"1", supplier)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("!ok")
	}
	rail.Infof("2. %+v", dummy)

	cache.Del(rail, keypre+"1")

	dummy, ok, err = cache.GetElse(rail, keypre+"1", supplier)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("!ok")
	}
	rail.Infof("3. %+v", dummy)
}

func TestRCache(t *testing.T) {
	rail := EmptyRail()
	LoadConfigFromFile("../app-conf-dev.yml", rail)
	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}

	keypre := "test:rcache:key:"
	exp := 60 * time.Second

	rcache := NewRCache(exp)

	val, e := rcache.Get(rail, "absent key")
	if e != nil {
		t.Fatal(e)
	}
	if val != "" {
		t.Fatal(val)
	}

	e = rcache.Put(rail, keypre+"1", "2")
	if e != nil {
		t.Fatal(e)
	}

	val, e = rcache.GetElse(rail, keypre+"1", nil)
	if e != nil {
		t.Fatal(e)
	}
	if val != "2" {
		t.Fatalf("val '%v' != \"2\"", val)
	}
}

func TestLKazyRCache(t *testing.T) {
	rail := EmptyRail()
	LoadConfigFromFile("../app-conf-dev.yml", rail)
	InitRedisFromProp(rail)

	keypre := "test:rcache:key:"
	exp := 60 * time.Second

	rcache := NewLazyRCache(exp)
	e := rcache.Put(rail, keypre+"1", "2")
	if e != nil {
		t.Fatal(e)
	}

	val, e := rcache.GetElse(rail, keypre+"1", nil)
	if e != nil {
		t.Fatal(e)
	}
	if val != "2" {
		t.Fatalf("val '%v' != \"2\"", val)
	}
}
