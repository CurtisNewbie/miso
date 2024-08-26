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
	s := NamedSprintf("${brand} Yes!", map[string]any{"brand": "AMD"})
	if s != "AMD Yes!" {
		t.Fatal(s)
	}
	t.Log(s)

	s = NamedSprintf("{brand} Yes!", map[string]any{"brand": "AMD"})
	if s != "{brand} Yes!" {
		t.Fatal(s)
	}
	t.Log(s)

	s = NamedSprintf("${brand} Yes!", map[string]any{"brand1": "AMD"})
	if s != " Yes!" {
		t.Fatal(s)
	}
	t.Log(s)
}

func BenchmarkNamedSprintf(b *testing.B) {
	p := map[string]any{"brand": "AMD"}
	pat := "${brand} Yes!"

	// 188.9 ns/op            32 B/op          3 allocs/op
	// 236.9 ns/op            32 B/op          3 allocs/op
	b.Run("NamedFmt", func(b *testing.B) {
		var s string
		for i := 0; i < b.N; i++ {
			s = NamedSprintf(pat, p)
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
		{0},
		{-1},
		{-2},
		{-3},
		{-4},
		{-5},
		{-6},
		{6},
	}
	for _, v := range tab {
		t.Logf("'%s'", PadSpace(v[0].(int), s))
	}
	t.Logf("'%-6s'", s)
	t.Logf("'%6s'", s)
}

func TestSplitKV(t *testing.T) {
	logkv := func(k, v string, ok bool) { t.Logf("k: '%v', v: '%v', ok: %v", k, v, ok) }
	k, v, ok := SplitKV("k : v", ":")
	if !ok {
		t.FailNow()
	}
	logkv(k, v, ok)

	k, v, ok = SplitKV("k : ", ":")
	if !ok {
		t.FailNow()
	}
	logkv(k, v, ok)

	k, v, ok = SplitKV(": v", ":")
	if ok {
		t.FailNow()
	}
	logkv(k, v, ok)

	k, v, ok = SplitKV(": ", ":")
	if ok {
		t.FailNow()
	}
	logkv(k, v, ok)

	k, v, ok = SplitKV("k : v1:v2", ":")
	if !ok || v != "v1:v2" {
		t.FailNow()
	}
	logkv(k, v, ok)
}
