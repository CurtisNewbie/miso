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

func TestEphPoolFiltersDroppedItems(t *testing.T) {
	p := NewEphPool[int](func(t int) (dropped bool) { return t%2 == 0 })
	for i := 0; i < 10; i++ {
		p.Push(i)
	}

	// PushFront + Pop from Back, so items are popped in insertion order
	for i := 1; i < 10; i += 2 {
		v, ok := p.Pop()
		if !ok {
			t.Fatalf("expected live item %v", i)
		}
		if v != i {
			t.Fatalf("expected %v, got %v", i, v)
		}
	}

	if _, ok := p.Pop(); ok {
		t.Fatal("all remaining items are filtered, pool should be empty")
	}
}
