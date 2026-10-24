package async

import (
	"strings"
	"sync"
	"testing"
)

func TestAwaitReturnsAfterFuturesComplete(t *testing.T) {
	group := NewAwaitFutures[int](nil)
	const n = 50
	for i := 0; i < n; i++ {
		v := i
		group.SubmitAsync(func() (int, error) { return v, nil })
	}

	futures := group.Await()
	for i, f := range futures {
		fired := false
		f.Then(func(int, error) { fired = true })
		if !fired {
			t.Fatalf("future %v should already be completed when Await returns", i)
		}
	}
}

func TestAwaitEmptyGroup(t *testing.T) {
	group := NewAwaitFutures[int](nil)

	futures := group.Await()
	if len(futures) != 0 {
		t.Fatalf("expected an empty group, got %v futures", len(futures))
	}
	if again := group.Await(); len(again) != 0 {
		t.Fatalf("repeated Await should return the empty sealed set, got %v futures", len(again))
	}
}

func TestSubmitAsyncAfterAwaitPanics(t *testing.T) {
	group := NewAwaitFutures[int](nil)
	group.SubmitAsync(func() (int, error) { return 1, nil })
	group.Await()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("SubmitAsync after Await should panic")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected a string panic message, got %T", r)
		}
		for _, want := range []string{"SubmitAsync", "sealed"} {
			if !strings.Contains(msg, want) {
				t.Fatalf("panic message %q should mention %q", msg, want)
			}
		}
	}()

	group.SubmitAsync(func() (int, error) { return 2, nil })
}

func TestAwaitReturnsAnIndependentCopy(t *testing.T) {
	group := NewAwaitFutures[int](nil)
	group.SubmitAsync(func() (int, error) { return 1, nil })
	group.SubmitAsync(func() (int, error) { return 2, nil })

	first := group.Await()
	if len(first) != 2 {
		t.Fatalf("expected 2 futures, got %v", len(first))
	}

	first[0] = nil
	first = append(first, nil)

	second := group.Await()
	if len(second) != 2 {
		t.Fatalf("mutating the returned slice changed the group size, got %v", len(second))
	}
	if second[0] == nil {
		t.Fatal("mutating the returned slice changed the group's futures")
	}
	if v, err := second[0].Get(); err != nil || v != 1 {
		t.Fatalf("expected the first future to survive, got (%v, %v)", v, err)
	}
}

func TestAwaitFuturesConcurrentSubmit(t *testing.T) {
	group := NewAwaitFutures[int](nil)
	const n = 50

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			group.SubmitAsync(func() (int, error) {
				return v, nil
			})
		}(i)
	}
	wg.Wait()

	futures := group.Await()
	if len(futures) != n {
		t.Fatalf("expected %v futures, got %v", n, len(futures))
	}

	sum := 0
	for _, f := range futures {
		v, err := f.Get()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		sum += v
	}
	expected := n * (n - 1) / 2
	if sum != expected {
		t.Fatalf("expected sum %v, got %v", expected, sum)
	}
}
