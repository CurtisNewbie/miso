package common

import "testing"

func TestSet(t *testing.T) {
	s := NewSet[string]()
	if s.Has("k") {
		t.Error("set should not have k")
		return
	}

	if !s.Add("k") {
		t.Error("k should be added")
		return
	}

	if !s.Has("k") {
		t.Error("set doesn't have k")
		return
	}

	if s.Add("k") {
		t.Error("k shouldn't be added")
		return
	}

	if s.Size() != 1 {
		t.Error("size should be 1")
		return
	}

	if s.IsEmpty() {
		t.Error("set should not be empty")
		return
	}

	s.AddThen("apple").AddThen("Juice")

	t.Logf("Set: %s", s.String())
}
