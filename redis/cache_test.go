package redis

import (
	"testing"
	"time"

	"github.com/curtisnewbie/miso/core"
)

type Dummy struct {
	Name string
	Age  int
}

func TestLazyObjRcache(t *testing.T) {
	c := core.EmptyRail()
	core.LoadConfigFromFile("../app-conf-dev.yml", c)
	if _, e := InitRedisFromProp(); e != nil {
		t.Fatal(e)
	}
	keypre := "test:lazy:obj:rcache:key:"
	exp := 60 * time.Second
	cache := NewLazyObjectRCache[Dummy](exp)

	supplier := func() (Dummy, bool, error) {
		c.Info("Called supplier")
		return Dummy{
			Name: "Banana",
			Age:  12,
		}, true, nil
	}

	cache.Del(c, keypre+"1")

	dummy, ok, err := cache.GetElse(c, keypre+"1", supplier)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("!ok")
	}
	c.Infof("1. %+v", dummy)

	dummy, ok, err = cache.GetElse(c, keypre+"1", supplier)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("!ok")
	}
	c.Infof("2. %+v", dummy)

	cache.Del(c, keypre+"1")

	dummy, ok, err = cache.GetElse(c, keypre+"1", supplier)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("!ok")
	}
	c.Infof("3. %+v", dummy)
}

func TestRCache(t *testing.T) {
	c := core.EmptyRail()
	core.LoadConfigFromFile("../app-conf-dev.yml", c)
	if _, e := InitRedisFromProp(); e != nil {
		t.Fatal(e)
	}

	keypre := "test:rcache:key:"
	exp := 60 * time.Second

	rcache := NewRCache(exp)

	val, e := rcache.Get(c, "absent key")
	if e != nil {
		t.Fatal(e)
	}
	if val != "" {
		t.Fatal(val)
	}

	e = rcache.Put(c, keypre+"1", "2")
	if e != nil {
		t.Fatal(e)
	}

	val, e = rcache.GetElse(c, keypre+"1", nil)
	if e != nil {
		t.Fatal(e)
	}
	if val != "2" {
		t.Fatalf("val '%v' != \"2\"", val)
	}
}

func TestLKazyRCache(t *testing.T) {
	c := core.EmptyRail()
	core.LoadConfigFromFile("../app-conf-dev.yml", c)
	InitRedisFromProp()

	keypre := "test:rcache:key:"
	exp := 60 * time.Second

	rcache := NewLazyRCache(exp)
	e := rcache.Put(c, keypre+"1", "2")
	if e != nil {
		t.Fatal(e)
	}

	val, e := rcache.GetElse(c, keypre+"1", nil)
	if e != nil {
		t.Fatal(e)
	}
	if val != "2" {
		t.Fatalf("val '%v' != \"2\"", val)
	}
}
