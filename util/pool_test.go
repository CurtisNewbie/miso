package util

import "testing"

func TestFixedPool(t *testing.T) {
	p := NewFixedPool[int](10)
	for i := 0; i < 10; i++ {
		p.Push(i)
	}
	for i := 0; i < 10; i++ {
		v, ok := p.Pop()
		if !ok {
			t.Fatalf("not okay, i: %v", i)
		}
		t.Logf("%v", v)
	}
}
