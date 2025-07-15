package miso

import (
	"fmt"
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

	keys := cache.Keys()
	if len(keys) != 1 {
		t.Logf("cache keys.len != 1")
	}
	t.Logf("keys: %v", keys)
}

func TestTTLCacheEvicted(t *testing.T) {
	type ttlDummy struct {
		name string
	}

	cache := NewTTLCache[ttlDummy](1*time.Second, 5)
	cache.OnEvicted(func(key string, tt ttlDummy) {
		t.Logf(">>>>>>> evicted: %v, %#v", key, tt)
	})

	cnt := 0
	elseGet := func() (ttlDummy, bool) {
		cnt += 1
		// rail.Infof("elseGet %v", cnt)
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

	t.Logf(">>>>>>> iter")
	for i := 0; i < 10; i++ {
		v, ok = cache.Get(fmt.Sprintf("abc-%v", i), elseGet)
		if !ok {
			t.Fatal("not ok")
		}
	}
	if cache.Size() > 5 {
		t.Fatalf("size is over 5, actual: %v", cache.Size())
	}
	Infof("cache.size: %v", cache.Size())

	t.Logf("keys: %v", cache.Keys())
}

func TestTTLCacheMaxSize(t *testing.T) {
	rail := EmptyRail()

	type ttlDummy struct {
		name string
	}

	cache := NewTTLCache[ttlDummy](1*time.Second, 10)
	cache.OnEvicted(func(key string, tt ttlDummy) {
		t.Logf(">>>>>>> evicted: %v, %#v", key, tt)
	})

	cnt := 0
	elseGet := func() (ttlDummy, bool) {
		cnt += 1
		rail.Infof("elseGet %v", cnt)
		return ttlDummy{
			name: "myDummy",
		}, true
	}

	for i := 0; i < 10; i++ {
		cache.Get(fmt.Sprintf("%d", i), elseGet)
	}

	if cnt != 10 {
		t.Fatalf("cnt should be 10, but %v", cnt)
	}

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

	ok = cache.PutIfAbsent("abc", ttlDummy{})
	if ok {
		t.Fatal("ok")
	}

	if cnt != 11 {
		t.Fatalf("cnt should be 11, but %v", cnt)
	}

	ok = cache.PutIfAbsent("what", ttlDummy{name: "what"})
	if !ok {
		t.Fatal("not ok")
	}

	if cache.Size() != 10 {
		t.Fatalf("size should be 10, but %v", cache.Size())
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

func BenchmarkTTLCache(b *testing.B) {
	type ttlDummy struct {
		name string
	}
	cache := NewTTLCache[ttlDummy](5*time.Second, 2)
	elseGet := func() (ttlDummy, bool) {
		return ttlDummy{
			name: "myDummy",
		}, true
	}
	_, ok := cache.Get("1", elseGet)
	if !ok {
		b.Fatal("not ok")
	}

	b.Run("cache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = cache.TryGet("1")
		}
	})

	m := map[string]ttlDummy{
		"1": ttlDummy{
			name: "myDummy",
		},
	}
	b.Run("map", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = m["1"]
		}
	})
}

func TestTTLCacheDel(t *testing.T) {
	cache := NewTTLCache[string](5*time.Second, 5)
	cache.Del("1")
	if cache.Size() != 0 {
		t.Fatal("size should be 0")
	}
	v, _ := cache.Get("1", func() (string, bool) { return "1", true })
	if v != "1" {
		t.Fatal("should be one")
	}
	if cache.Size() != 1 {
		t.Fatal("size should be 1")
	}
	cache.Del("2")
	if cache.Size() != 1 {
		t.Fatal("size should be 1")
	}

	cache.Del("1")
	if cache.Size() != 0 {
		t.Fatal("size should be 0")
	}
}

func TestTTLCacheExist(t *testing.T) {
	cache := NewTTLCache[string](100*time.Millisecond, 5)
	cache.Put("k", "v")

	// the key should be evicted already
	if !cache.Exists("k") {
		t.Fatal("should exists")
	}
	if cache.PutIfAbsent("k", "v2") {
		t.Fatal("should not be put into cache until expired")
	}

	time.Sleep(150 * time.Millisecond)
	if cache.Exists("k") {
		t.Fatal("should have expired")
	}

	if !cache.PutIfAbsent("k", "v2") {
		t.Fatal("should be put into cache")
	}
	v, ok := cache.Get("k", nil)
	if !ok {
		t.Fatal("should be ok")
	}
	if v != "v2" {
		t.Fatal("v is not v2")
	}
}
