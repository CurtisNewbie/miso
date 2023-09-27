package miso

import (
	"testing"
	"time"
)

func TestTTLCacheNormal(t *testing.T) {
	rail := EmptyRail()

	type ttlDummy struct {
		name string
	}

	cache := NewTTLCache[ttlDummy](1 * time.Minute)

	cnt := 0
	elseGet := func() (ttlDummy, bool) {
		cnt += 1
		rail.Infof("elseGet %v", cnt)
		return ttlDummy{
			name: "myDummy",
		}, true
	}

	v, ok := cache.Get("abc", elseGet)
	if !ok {
		t.Fatal("not ok")
	}

	if v.name != "myDummy" {
		t.Fatalf("name not myDummy, but %v", v.name)
	}

	// key should valid, should return the first cached dummy
	v, ok = cache.Get("abc", elseGet)
	if !ok {
		t.Fatal("not ok")
	}

	if v.name != "myDummy" {
		t.Fatalf("name not myDummy, but %v", v.name)
	}

	if cnt > 1 {
		t.Fatalf("cnt should be 1, but %v", cnt)
	}
}

func TestTTLCacheEvicted(t *testing.T) {
	rail := EmptyRail()

	type ttlDummy struct {
		name string
	}

	cache := NewTTLCache[ttlDummy](1 * time.Second)

	cnt := 0
	elseGet := func() (ttlDummy, bool) {
		cnt += 1
		rail.Infof("elseGet %v", cnt)
		return ttlDummy{
			name: "myDummy",
		}, true
	}

	v, ok := cache.Get("abc", elseGet)
	if !ok {
		t.Fatal("not ok")
	}

	if v.name != "myDummy" {
		t.Fatalf("name not myDummy, but %v", v.name)
	}

	// the key should be evicted already
	time.Sleep(2 * time.Second)

	v, ok = cache.Get("abc", elseGet)
	if !ok {
		t.Fatal("not ok")
	}

	if v.name != "myDummy" {
		t.Fatalf("name not myDummy, but %v", v.name)
	}

	if cnt != 2 {
		t.Fatalf("cnt should be 2, but %v", cnt)
	}
}
