package miso

import (
	"testing"
	"unicode/utf8"
)

func TestERand(t *testing.T) {
	var s string
	var err error

	s = ERand(0)
	if s != "" {
		t.Fatalf("Generate random string should be '', actual: %s", s)
	}

	l := 10
	s = ERand(l)
	if err != nil {
		t.Fatalf("Failed to generate random str, %v", err)
	}

	rc := utf8.RuneCountInString(s)
	if rc != l {
		t.Fatalf("Expected len: %d, actual len: %d (%s)", l, rc, s)
	}
	t.Log(s)
}
