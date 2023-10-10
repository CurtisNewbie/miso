package miso

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTTLCacheNormal(t *testing.T) {
	rail := EmptyRail()

	type ttlDummy struct {
		name string
	}

	cache := NewTTLCache[ttlDummy](1*time.Minute, 5)

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

	cache := NewTTLCache[ttlDummy](1*time.Second, 5)

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

func TestTTLCacheMaxSize(t *testing.T) {
	rail := EmptyRail()

	type ttlDummy struct {
		name string
	}

	cache := NewTTLCache[ttlDummy](1*time.Second, 10)

	cnt := 0
	elseGet := func() (ttlDummy, bool) {
		cnt += 1
		rail.Infof("elseGet %v", cnt)
		return ttlDummy{
			name: "myDummy",
		}, true
	}

	for i := 0; i < 10; i++ {
		cache.Get(ERand(5), elseGet)
	}

	if cnt != 10 {
		t.Fatalf("cnt should be 10, but %v", cnt)
	}

	// all the key should be evicted already
	time.Sleep(2 * time.Second)

	v, ok := cache.Get("abc", elseGet)
	if !ok {
		t.Fatal("not ok")
	}

	if v.name != "myDummy" {
		t.Fatalf("name not myDummy, but %v", v.name)
	}

	if cnt != 11 {
		t.Fatalf("cnt should be 11, but %v", cnt)
	}

	v, ok = cache.Get("abc", elseGet)
	if !ok {
		t.Fatal("not ok")
	}

	if v.name != "myDummy" {
		t.Fatalf("name not myDummy, but %v", v.name)
	}

	if cnt != 11 {
		t.Fatalf("cnt should be 11, but %v", cnt)
	}
}

func TestTTLCacheConcurrent(t *testing.T) {
	rail := EmptyRail()

	type ttlDummy struct {
		name string
	}

	cache := NewTTLCache[ttlDummy](1*time.Minute, 10)

	var cnt int32 = 0
	elseGet := func() (ttlDummy, bool) {
		rail.Infof("elseGet %v", atomic.AddInt32(&cnt, 1))
		return ttlDummy{
			name: "myDummy",
		}, true
	}

	var failed int32 = 0

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(t *testing.T) {
			defer wg.Done()
			v, ok := cache.Get("abc", elseGet)
			if !ok {
				t.Log("not ok")
				atomic.AddInt32(&failed, 1)
			}

			if v.name != "myDummy" {
				t.Logf("name not myDummy, but %v", v.name)
				atomic.AddInt32(&failed, 1)
			}
		}(t)
	}
	wg.Wait()

	if cnt != 1 {
		t.Fatalf("cnt should be 10, but %v", cnt)
	}
	if failed > 0 {
		t.Fail()
	}
}

func TestTinyTTLCacheMaxSize(t *testing.T) {
	rail := EmptyRail()

	type ttlDummy struct {
		name string
	}

	cache := NewTinyTTLCache[ttlDummy](1*time.Second, 10)

	cnt := 0
	elseGet := func() (ttlDummy, bool) {
		cnt += 1
		rail.Infof("elseGet %v", cnt)
		return ttlDummy{
			name: "myDummy",
		}, true
	}

	for i := 0; i < 10; i++ {
		cache.Get(ERand(5), elseGet)
	}

	if cnt != 10 {
		t.Fatalf("cnt should be 10, but %v", cnt)
	}

	// all the key should be evicted already
	time.Sleep(2 * time.Second)

	v, ok := cache.Get("abc", elseGet)
	if !ok {
		t.Fatal("not ok")
	}

	if v.name != "myDummy" {
		t.Fatalf("name not myDummy, but %v", v.name)
	}

	if cnt != 11 {
		t.Fatalf("cnt should be 11, but %v", cnt)
	}

	v, ok = cache.Get("abc", elseGet)
	if !ok {
		t.Fatal("not ok")
	}

	if v.name != "myDummy" {
		t.Fatalf("name not myDummy, but %v", v.name)
	}

	if cnt != 11 {
		t.Fatalf("cnt should be 11, but %v", cnt)
	}
}
