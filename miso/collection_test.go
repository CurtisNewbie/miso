package miso

import "testing"

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
