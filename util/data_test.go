package util

import (
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

func TestSliceMap(t *testing.T) {
	s := []string{"1", "2", "3"}
	v := SliceMap(s, func(s string) int { return cast.ToInt(s) })
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

	for _, k := range m.Keys() {
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
