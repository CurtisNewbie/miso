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

func TestDistinct(t *testing.T) {
	l := Distinct([]string{"a", "b", "c", "c", "d", "c"})
	t.Log(l)
	if len(l) != 4 {
		t.Fatal("len should be 4")
	}
}

func TestFilter(t *testing.T) {
	l := Filter([]string{"a", "b", "c", "c", "d", "c"}, func(v string) bool { return v != "c" })
	t.Log(l)
	if len(l) != 3 {
		t.Fatal("len should be 3")
	}
}

func TestCopyFilter(t *testing.T) {
	l := CopyFilter([]string{"a", "b", "c", "c", "d", "c"}, func(v string) bool { return v != "c" })
	t.Log(l)
	if len(l) != 3 {
		t.Fatal("len should be 3")
	}
}

func TestFastDistinct(t *testing.T) {
	l := FastDistinct([]string{"a", "b", "c", "c", "d", "c"})
	t.Log(l)
	if len(l) != 4 {
		t.Fatal("len should be 4")
	}
}

func BenchmarkDistinct(b *testing.B) {
	sample := []string{"a", "b", "c", "c", "d", "c"}
	b.Run("old", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cp := make([]string, len(sample))
			copy(cp, sample)

			cp = Distinct(cp)
			if len(cp) != 4 {
				b.Fatal("len should be 4")
			}
		}
	})

	b.Run("new", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cp := make([]string, len(sample))
			copy(cp, sample)
			cp = FastDistinct(cp)
			if len(cp) != 4 {
				b.Fatal("len should be 4")
			}
		}
	})
}

func TestMapTo(t *testing.T) {
	s := []string{"1", "2", "3"}
	v := MapTo(s, func(s string) int { return cast.ToInt(s) })
	if len(v) < 3 {
		t.Fatal("len != 3")
	}
	if v[0] != 1 {
		t.Fatal("[0] != 1")
	}
	if v[1] != 2 {
		t.Fatal("[1] != 2")
	}
	if v[2] != 3 {
		t.Fatal("[2] != 3")
	}
	t.Log(v)
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

func TestMergeSlice(t *testing.T) {
	type vt struct {
		name string
		val  int
	}
	vs := []vt{{name: "apple", val: 1}, {name: "juice", val: 2}, {name: "apple", val: 3}}
	m := MergeSlice(vs, func(v vt) string { return v.name })
	t.Logf("merged: %+v", m)

	s, ok := m["apple"]
	if !ok {
		t.Fatal("apple not found")
	}
	if len(s) != 2 {
		t.Fatal("apple should have 2 values")
	}

	s, ok = m["juice"]
	if !ok {
		t.Fatal("juice not found")
	}
	if len(s) != 1 {
		t.Fatal("juice should have 1 value")
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

func TestSliceCop(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	b := SliceCopy(a)
	b[0] = -1
	t.Logf("a: %v, b: %v", a, b)

	c := SliceCopy([]int(nil))
	t.Logf("c: %v", c)
}

func TestSliceRemove(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	b := SliceRemove(a, 1, 3)
	t.Logf("a: %v, b: %v", a, b)
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
