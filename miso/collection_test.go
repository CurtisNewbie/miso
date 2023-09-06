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
