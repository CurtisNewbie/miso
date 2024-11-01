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

func TestRandOp(t *testing.T) {
	m := map[string]int{}
	for i := 0; i < 1000; i++ {
		RandOp(func() {
			m["1"]++
		}, func() {
			m["2"]++
		}, func() {
			m["3"]++
		})
	}
	t.Log(m)
}

func TestWeightedRandPick(t *testing.T) {
	arr := []WeightedItem[string]{{"apple", 10}, {"juice", 10}, {"orange", 10}}
	for j := 0; j < 3; j++ {
		m := map[string]int{}
		for i := 0; i < 100_000; i++ {
			p := WeightedRandPick(arr)
			m[p.Value]++
		}
		t.Log(m)
	}
}
