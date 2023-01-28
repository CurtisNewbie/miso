package common

import (
	"testing"
)

func TestFirstChar(t *testing.T) {
	var l int
	var c string
	if l, c = FirstChar(""); l != 0 || c != "" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, c)

	if l, c = FirstChar("A"); l != 1 || c != "A" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, c)

	if l, c = FirstChar("abccc d"); l != 7 || c != "a" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, c)
}

func TestLastChar(t *testing.T) {
	var l int
	var c string
	if l, c = LastChar(""); l != 0 || c != "" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, c)

	if l, c = LastChar("A"); l != 1 || c != "A" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, c)

	if l, c = LastChar("abccc d"); l != 7 || c != "d" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, c)
}

func TestRuneWrp(t *testing.T) {
	s := "abcde   "
	rw := GetRuneWrp(s)

	if rw.StrAt(0) != "a" {
		t.Error()
		return
	}

	if rw.StrAt(2) != "c" {
		t.Error()
		return
	}

	if rw.Len() != 8 {
		t.Error()
		return
	}
}
