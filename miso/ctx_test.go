package miso

import (
	"errors"
	"fmt"
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

func TestRailErrorIf(t *testing.T) {
	r := EmptyRail()
	r.ErrorIf("Create file", nil)
	r.ErrorIf("Delete file, file_id: %v", errors.New("file not found"), "ABC123")
}

func TestRailWarnIf(t *testing.T) {
	r := EmptyRail()
	r.WarnIf("Create file", nil)
	r.WarnIf("Delete file failed, file_id: %v", errors.New("file not found"), "ABC123")
	r.WarnIf("Delete file failed, file_id: %v", fmt.Errorf("failed to delete file, %w", errors.New("file not found")), "ABC123")
}
