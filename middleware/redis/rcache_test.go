package redis

import (
	"fmt"
	"testing"
	"time"

	"github.com/curtisnewbie/miso/miso"
)

type RCacheDummy struct {
	Name string
	Age  int
}

func preRCacheTest(t *testing.T) miso.Rail {
	rail := miso.EmptyRail()
	miso.SetProp(PropRedisEnabled, true)
	miso.SetLogLevel("debug")
	if _, e := InitRedisFromProp(rail); e != nil {
		t.Fatal(e)
	}
	return rail
}

func TestRcacheWithObject(t *testing.T) {
	rail := preRCacheTest(t)
	exp := 10 * time.Second
	invokeCount := 0
	supplier := func() (RCacheDummy, error) {
		invokeCount++
		rail.Infof("Called supplier, %v", invokeCount)
		return RCacheDummy{
			Name: "Banana",
			Age:  12,
		}, nil
	}

	cache := NewRCache[RCacheDummy]("test0", RCacheConfig{Exp: exp})
	cache.Del(rail, "1")

	dummy, err := cache.Get(rail, "1", supplier)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	rail.Infof("1. got from supplier %+v, invokeCount: %v", dummy, invokeCount)
	if invokeCount != 1 {
		t.Logf("invokeCount: %v", invokeCount)
		t.FailNow()
	}

	dummy, err = cache.Get(rail, "1", supplier)
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

	dummy, err = cache.Get(rail, "1", supplier)
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
	rcache := NewRCache[string]("test2", RCacheConfig{Exp: exp})

	_, e := rcache.Get(rail, "absent key", nil)
	if e == nil || !miso.IsNoneErr(e) {
		t.Fatal(e)
	}

	e = rcache.Put(rail, "1", "3")
	if e != nil {
		t.Fatal(e)
	}

	var val string
	val, e = rcache.Get(rail, "1", nil)
	if e != nil {
		t.Fatal(e)
	}
	if val != "3" {
		t.Fatalf("val '%v' != \"3\"", val)
	}
}

func TestRCache2(t *testing.T) {
	rail := preRCacheTest(t)

	exp := 10 * time.Second
	supplier := func() (string, error) {
		return "", miso.NoneErr
	}

	rcache := NewRCache[string]("test", RCacheConfig{Exp: exp, NoSync: true})

	e := rcache.Put(rail, "1", "2")
	if e != nil {
		t.Fatal(e)
	}

	val, e := rcache.Get(rail, "1", supplier)
	if e != nil {
		t.Fatal(e)
	}
	if val != "2" {
		t.Fatalf("val '%v' != \"2\"", val)
	}

	ok, err := rcache.Exists(rail, "1")
	if err != nil {
		t.Fatal(e)
	}
	if !ok {
		t.Fatal("not ok but should be ok")
	}

	ok, err = rcache.Exists(rail, "nope")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("ok but shouldn't be")
	}

	for i := 0; i < 200; i++ {
		if err := rcache.Put(rail, fmt.Sprintf("%d", i), "1"); err != nil {
			t.Fatal(err)
		}
	}

	if err := rcache.DelAll(rail); err != nil {
		t.Fatal(err)
	}
}
