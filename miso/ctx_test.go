package miso

import (
	"sync"
	"testing"
)

func TestNewSpan(t *testing.T) {
	ec := EmptyRail()
	ec.Infof("Parent Span")

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		cc := ec.NextSpan()
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			cc.Infof("Child Span, j: %v", j)
		}(i)
	}

	wg.Wait()
}

func TestRailIsDone(t *testing.T) {
	r := EmptyRail()
	if r.IsDone() {
		t.Fatal("isDone")
	}

	r, c := r.WithCancel()
	if r.IsDone() {
		t.Fatal("isDone")
	}
	c()
	if !r.IsDone() {
		t.Fatal("not isDone")
	}
}
