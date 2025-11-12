package pool

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

func TestEphPool(t *testing.T) {
	p := NewEphPool[int](func(t int) (dropped bool) { return false })
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

	_, ok := p.Pop()
	if ok {
		t.Fatalf("okay")
	}

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
