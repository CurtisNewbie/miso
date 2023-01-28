package common

import (
	"testing"
)

func TestFirstChar(t *testing.T) {
	var l int
	var c string
	var cp *string = &c
	if l, *cp = FirstChar(""); l != 0 || *cp != "" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, *cp)

	if l, *cp = FirstChar("A"); l != 1 || *cp != "A" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, *cp)

	if l, *cp = FirstChar("abccc d"); l != 7 || *cp != "a" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, *cp)
}

func TestLastChar(t *testing.T) {
	var l int
	var c string
	var cp *string = &c
	if l, *cp = LastChar(""); l != 0 || *cp != "" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, *cp)

	if l, *cp = LastChar("A"); l != 1 || *cp != "A" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, *cp)

	if l, *cp = LastChar("abccc d"); l != 7 || *cp != "d" {
		t.Error()
		return
	}
	t.Logf("l: %d, c: '%s'", l, *cp)
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
