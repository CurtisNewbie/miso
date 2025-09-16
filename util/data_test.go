package util

import (
	"sync"
	"testing"

	"github.com/spf13/cast"
)

func TestSet(t *testing.T) {
	s := NewSet[string]()
	if s.Has("k") {
		t.Fatal("set should not have k")
	}

	if !s.Add("k") {
		t.Fatal("k should be added")
	}

	if !s.Has("k") {
		t.Fatal("set doesn't have k")
	}

	if s.Add("k") {
		t.Fatal("k shouldn't be added")
	}

	if s.Size() != 1 {
		t.Fatal("size should be 1")
	}

	if s.IsEmpty() {
		t.Fatal("set should not be empty")
	}

	s.AddThen("apple").AddThen("Juice")

	s.AddAll([]string{"orange", "juice"})

	if !s.Has("orange") {
		t.Fatal("set doesn't contain orange")
	}

	if !s.Has("juice") {
		t.Fatal("set doesn't contain juice")
	}

	t.Logf("Set: %s", s.String())

	s.Del("juice")
	if s.Has("juice") {
		t.Fatal("set shouldn't contain juice")
	}
	t.Logf("Set: %v", s)
}

func TestSet2(t *testing.T) {
	s := NewSetFromSlice[string]([]string{"a", "b", "c"})
	if !s.Has("a") {
		t.Fatal("set should have a")
	}
	if s.Size() != 3 {
		t.Fatal("set's size should be 3")
	}
	t.Logf("set: %v", s)
}

func TestRWMap(t *testing.T) {
	m := NewRWMap[string, string]()
	aw := NewAwaitFutures[any](NewAsyncPool(100, 5))
	keys := []string{"ky", "kn", "ke"}

	aw.SubmitAsync(func() (any, error) {
		m.Put("ky", "yes")
		return nil, nil
	})

	aw.SubmitAsync(func() (any, error) {
		m.Put("kn", "no")
		return nil, nil
	})

	aw.SubmitAsync(func() (any, error) {
		m.GetElse("ke", func(k string) string { return k + "lse" })
		return nil, nil
	})
	aw.Await()

	for _, k := range keys {
		v, ok := m.Get(k)
		if !ok {
			t.Fatal("!ok")
		}
		t.Logf("%v -> %v", k, v)
	}
}

func TestStack(t *testing.T) {
	s := NewStack[*string](3)
	v1 := "1"
	v2 := "2"
	v3 := "3"
	s.Push(&v1)
	s.Push(&v2)
	s.Push(&v3)
	t.Logf("stack: %v", s)
	t.Logf("copy: %+v", s.Slice())

	fef := func(v *string) bool {
		t.Logf("foreach: %v", *v)
		return true
	}
	s.ForEach(fef)

	p, ok := s.Peek()
	if !ok {
		t.Fatal("cannot peek")
	}
	t.Logf("peeked: %v", *p)

	expected := []string{"3", "2", "1"}
	for _, ex := range expected {
		if s.Empty() {
			t.Fatal("empty")
		}
		v, ok := s.Pop()
		if !ok {
			t.Fatal("cannot pop")
		}
		if *v != ex {
			t.Fatalf("not %v", ex)
		}
		t.Logf("popped: %v", ex)
		t.Logf("stack: %v", s)
	}

	if !s.Empty() {
		t.Fatal("not empty")
	}
	_, ok = s.Pop()
	if ok {
		t.Fatal("can pop")
	}
	s.ForEach(fef)

	v4 := "4"
	s.Push(&v4)
	t.Logf("stack: %v", s)
	s.ForEach(fef)
}

func TestHeap(t *testing.T) {
	h := NewHeap[int](10, func(iv, jv int) bool {
		return iv < jv
	})
	for i := 0; i < 10; i++ {
		h.Push(10 - i)
	}

	t.Logf("peek: %v", h.heap.Peek())
	n := h.Pop()
	t.Logf("poped: %v", n)

	for i := 0; i < 10; i++ {
		h.Push(i)
	}

	prev := -1
	for h.Len() > 0 {
		t.Logf("peek: %v", h.Peek())
		p := h.Pop()
		t.Log(p)
		if p < prev {
			t.Fatal("Wrong order")
		}
		prev = p
	}
}

func TestStrRWMap(t *testing.T) {
	m := NewStrRWMap[string]()
	aw := NewAwaitFutures[any](NewAsyncPool(100, 5))
	keys := []string{"ky", "kn", "ke"}

	aw.SubmitAsync(func() (any, error) {
		m.Put("ky", "yes")
		return nil, nil
	})

	aw.SubmitAsync(func() (any, error) {
		m.Put("kn", "no")
		return nil, nil
	})

	aw.SubmitAsync(func() (any, error) {
		m.GetElse("ke", func(k string) string { return k + "lse" })
		return nil, nil
	})
	aw.Await()

	for _, k := range keys {
		v, ok := m.Get(k)
		if !ok {
			t.Fatal("!ok")
		}
		t.Logf("%v -> %v", k, v)
	}
}

func BenchmarkRWMap(b *testing.B) {
	m := NewRWMap[string, string]()
	sm := NewStrRWMap[string]()

	keyCnt := 30
	keys := []string{}
	for i := range keyCnt {
		s := cast.ToString(i)
		keys = append(keys, s)
		m.Put(s, s)
		sm.Put(s, s)
	}

	/*
		shards = 32
		keyCnt = 30

		goos: darwin
		goarch: arm64
		pkg: github.com/curtisnewbie/miso/util
		cpu: Apple M3 Pro
		=== RUN   BenchmarkRWMap
		BenchmarkRWMap
		=== RUN   BenchmarkRWMap/RWMap.Get
		BenchmarkRWMap/RWMap.Get
		BenchmarkRWMap/RWMap.Get-11               452205              2616 ns/op              50 B/op          1 allocs/op
		=== RUN   BenchmarkRWMap/StrRWMap.Get
		BenchmarkRWMap/StrRWMap.Get
		BenchmarkRWMap/StrRWMap.Get-11           1704829               692.7 ns/op            48 B/op          1 allocs/op
	*/

	sg1 := sync.WaitGroup{}
	b.Run("RWMap.Get", func(b *testing.B) {
		for range b.N {
			sg1.Add(1)
			go func() {
				defer sg1.Done()
				for _, k := range keys {
					m.Get(k)
				}
			}()
		}
		sg1.Wait()
	})

	sg2 := sync.WaitGroup{}
	b.Run("StrRWMap.Get", func(b *testing.B) {
		for range b.N {
			sg2.Add(1)
			go func() {
				defer sg2.Done()
				for _, k := range keys {
					sm.Get(k)
				}
			}()
		}
		sg2.Wait()
	})
}

func TestStrSliceMap(t *testing.T) {
	var l []struct {
		Key   string
		Value string
	}
	l = append(l, struct {
		Key   string
		Value string
	}{"1", "1"}, struct {
		Key   string
		Value string
	}{"1", "2"}, struct {
		Key   string
		Value string
	}{"2", "3"})

	m := StrSliceMap(l, func(v struct {
		Key   string
		Value string
	}) string {
		return v.Key
	}, func(v struct {
		Key   string
		Value string
	}) string {
		return v.Value
	})

	t.Logf("%#v", m)
}
