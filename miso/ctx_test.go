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
	r.ErrorIf(nil, "Create file")
	r.ErrorIf(errors.New("file not found"), "Delete file, file_id: %v", "ABC123")
}

func TestRailWarnIf(t *testing.T) {
	r := EmptyRail()
	r.WarnIf(nil, "Create file")
	r.WarnIf(errors.New("file not found"), "Delete file failed, file_id: %v", "ABC123")
	r.WarnIf(fmt.Errorf("failed to delete file, %w", errors.New("file not found")), "Delete file failed, file_id: %v", "ABC123")
}

func TestErrorfStackTrace(t *testing.T) {
	r := EmptyRail()
	err := errors.New("local error")
	r.Errorf("TestErrorfStackTrace err: %v, %v", err, testErrorfStackTrace1())
}

func testErrorfStackTrace1() error {
	return NewErrf("NO!!!!!")
}

func TestErrorStackTrace(t *testing.T) {
	EmptyRail().Error(NewErrf("oh no"))
	Error(NewErrf("oh no"))
	EmptyRail().Warn(NewErrf("oh no"))
	Warn(NewErrf("oh no"))
	EmptyRail().Error((*MisoErr)(nil))
	Error((*MisoErr)(nil))
	EmptyRail().Warn((*MisoErr)(nil))
	Warn((*MisoErr)(nil))
}
