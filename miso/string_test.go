package miso

import (
	"strings"
	"testing"
)

func TestPadNum(t *testing.T) {
	var res string
	var expected string

	res = PadNum(11, 4)
	expected = "0011"
	if res != expected {
		t.Fatalf("actual: %v, expected: %v", res, expected)
	}

	res = PadNum(0, 4)
	expected = "0000"
	if res != expected {
		t.Fatalf("actual: %v, expected: %v", res, expected)
	}

	res = PadNum(12345, 4)
	expected = "12345"
	if res != expected {
		t.Fatalf("actual: %v, expected: %v", res, expected)
	}
}

func TestMaxLenStr(t *testing.T) {
	s := "123456"
	ml := MaxLenStr(s, 3)
	if ml != "123" {
		t.Logf("%v != '123'", ml)
		t.FailNow()
	}

	s = "12"
	ml = MaxLenStr(s, 3)
	if ml != "12" {
		t.Logf("%v != '12'", ml)
		t.FailNow()
	}
}

func TestHasPrefixIgnoreCase(t *testing.T) {
	// s, p, matched
	tests := [][3]any{
		{"abc", "abc", true},
		{"abc", "Abc", true},
		{"abcd", "Abc", true},
		{"abcd", "AbC", true},
		{"abc", "Abcd", false},
		{"abc", "", true},
		{"abc", "a", true},
		{"abc", "ab", true},
		{"Abc", "abc", true},
		{"Abc", "Abc", true},
		{"Abcd", "Abc", true},
		{"Abcd", "AbC", true},
		{"Abc", "Abcd", false},
		{"Abc", "", true},
		{"Abc", "a", true},
		{"Abc", "ab", true},
	}

	for i := range tests {
		te := tests[i]
		s := te[0].(string)
		p := te[1].(string)
		m := te[2].(bool)
		if HasPrefixIgnoreCase(s, p) != m {
			t.Logf("s: %v, p: %v, m: %v", s, p, m)
			t.FailNow()
		}
	}
}

func TestHasSuffixIgnoreCaseFuzz(t *testing.T) {
	for i := 0; i < 500; i++ {
		s := strings.ToLower(RandStr(30))
		p := strings.ToLower(RandStr(3))
		if strings.HasSuffix(s, p) != HasSuffixIgnoreCase(s, p) {
			t.Logf("s: %v, p: %v", s, p)
			t.FailNow()
		}
	}
}

func TestHasPrefixIgnoreCaseFuzz(t *testing.T) {
	for i := 0; i < 500; i++ {
		s := strings.ToLower(RandStr(30))
		p := strings.ToLower(RandStr(3))
		if strings.HasPrefix(s, p) != HasPrefixIgnoreCase(s, p) {
			t.Logf("s: %v, p: %v", s, p)
			t.FailNow()
		}
	}
}

func BenchmarkHasPrefix(b *testing.B) {
	b.Run("HasPrefix", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if !strings.HasPrefix("abCde", "abC") {
				b.FailNow()
			}
		}
	})

	b.Run("HasPrefixIgnoreCase", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if !HasPrefixIgnoreCase("abCde", "abC") {
				b.FailNow()
			}
		}
	})
}
