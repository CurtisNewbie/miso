package util

import "testing"

func BenchmarkUnsafeStrConvert(b *testing.B) {
	var s string
	byt := []byte("oops")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s = UnsafeByt2Str(byt)
		byt = UnsafeStr2Byt(s)
	}
	b.StopTimer()
	b.ReportAllocs()

	if string(byt) != "oops" {
		b.Fatal(string(byt))
	}
}

func TestUnsafeStrConvert(t *testing.T) {
	s := "abc"
	byt := UnsafeStr2Byt(s)
	t.Logf("1. s: %v, byt: %s, len(byt): %v", s, byt, len(byt))
	// byt[0] = 'd' // will panic

	byt = []byte("abc") // one alloc for "abc" -> []byte
	s = UnsafeByt2Str(byt)
	t.Logf("2. s: %v, byt: %s", s, byt)

	byt[0] = 'd' // modified in place at 0.
	t.Logf("3. s: %v, byt: %s", s, byt)

	byt = append(byt, 'e') // expanded, byt is not the original one anymore
	t.Logf("4. s: %v, byt: %s", s, byt)
}
