package util

import (
	"testing"
	"unicode/utf8"
)

func TestERand(t *testing.T) {
	var s string

	s = ERand(0)
	if s != "" {
		t.Fatalf("Generate random string should be '', actual: %s", s)
	}

	l := 10
	s = ERand(l)

	rc := utf8.RuneCountInString(s)
	if rc != l {
		t.Fatalf("Expected len: %d, actual len: %d (%s)", l, rc, s)
	}
	t.Log(s)
}

var v string
var v2 string

func BenchmarkRandLowerAlphaNumeric(b *testing.B) {
	b.Run("fast", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v = RandLowerAlphaNumeric16()
		}
	})
	b.Run("slow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v2 = RandLowerAlphaNumeric(16)
		}
	})
}
