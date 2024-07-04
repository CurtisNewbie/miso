package util

import (
	"fmt"
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

func TestIsBlankStr(t *testing.T) {
	if !IsBlankStr("") {
		t.Fatal()
	}
	if !IsBlankStr(" ") {
		t.Fatal()
	}
	if IsBlankStr("a") {
		t.Fatal()
	}
	if IsBlankStr(" a ") {
		t.Fatal()
	}
}

func BenchmarkIsBlankStr(b *testing.B) {
	// before: 2.061 ns/op           0 B/op          0 allocs/op
	// after: 0.2890 ns/op          0 B/op          0 allocs/op
	var r bool
	for i := 0; i < b.N; i++ {
		r = IsBlankStr("")
	}
	b.ReportAllocs()
	if !r {
		b.Fatal()
	}
}

func TestNamedSprintf(t *testing.T) {
	s := NamedSprintf("{{.brand}} Yes!", map[string]any{"brand": "AMD"})
	if s != "AMD Yes!" {
		t.Fatal(s)
	}
	t.Log(s)
}

func BenchmarkNamedSprintf(b *testing.B) {
	p := map[string]any{"brand": "AMD"}
	pat := NamedFmt("{{.brand}} Yes!")

	// 587.2 ns/op           248 B/op         11 allocs/op
	b.Run("NamedFmt", func(b *testing.B) {
		var s string
		for i := 0; i < b.N; i++ {
			s = pat.Sprintf(p)
		}
		b.Log(s)
	})

	// 42.79 ns/op            8 B/op          1 allocs/op
	b.Run("fmt", func(b *testing.B) {
		var s string
		for i := 0; i < b.N; i++ {
			s = fmt.Sprintf("%v Yes!", "AMD")
		}
		b.Log(s)
	})
}

func TestFmtFloat(t *testing.T) {
	tab := [][]int{
		{0, 0},
		{1, 0},
		{2, 0},
		{3, 0},
		{4, 0},
		{4, 1},
		{4, 2},
		{5, 2},
		{5, 3},
		{5, 4},
		{-5, 4},
		{-6, 4},
		{-7, 4},
	}
	f := 12.333
	for _, v := range tab {
		t.Logf("'%s'", FmtFloat(f, v[0], v[1]))
	}
}

func TestPadSpace(t *testing.T) {
	s := "yes"
	tab := [][]any{
		{0, true},
		{1, true},
		{2, true},
		{3, true},
		{4, true},
		{5, true},
		{6, true},
		{6, false},
	}
	for _, v := range tab {
		t.Logf("'%s'", PadSpace(v[0].(int), s, v[1].(bool)))
	}
}
