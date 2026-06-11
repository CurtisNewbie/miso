package retry

import (
	"testing"
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/util/cli"
)

func TestGetOneDyn(t *testing.T) {
	// Scenario 1: succeeds on first call, no retry needed
	{
		result, err := GetOneDyn(func() (int, error) {
			return 42, nil
		}, func(i int, err error) (time.Duration, bool) { return 0, true })
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result != 42 {
			t.Fatalf("expected 42, got %v", result)
		}
		cli.TPrintlnf("scenario 1 passed: result=%v", result)
	}

	// Scenario 2: fails twice then succeeds, verifies gapFunc called with i=0, i=1
	{
		callCount := 0
		gapIs := []int{}
		result, err := GetOneDyn(func() (string, error) {
			callCount++
			if callCount < 3 {
				return "", errs.NewErrf("not yet")
			}
			return "done", nil
		}, func(i int, err error) (time.Duration, bool) {
			gapIs = append(gapIs, i)
			return 0, true
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result != "done" {
			t.Fatalf("expected 'done', got %v", result)
		}
		if len(gapIs) != 2 {
			t.Fatalf("expected gapFunc called 2 times, got %v", len(gapIs))
		}
		if gapIs[0] != 1 || gapIs[1] != 2 {
			t.Fatalf("expected gapFunc i=[1,2], got %v", gapIs)
		}
		cli.TPrintlnf("scenario 2 passed: callCount=%v gapIs=%v", callCount, gapIs)
	}

	// Scenario 3: gapFunc returns doRetry=false on first error, stops immediately
	{
		callCount := 0
		result, err := GetOneDyn(func() (int, error) {
			callCount++
			return 0, errs.NewErrf("stop")
		}, func(i int, err error) (time.Duration, bool) { return 0, false })
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if callCount != 1 {
			t.Fatalf("expected 1 call, got %v", callCount)
		}
		_ = result
		cli.TPrintlnf("scenario 3 passed: callCount=%v err=%v", callCount, err)
	}

	// Scenario 4: gapFunc receives increasing i and the actual error across retries
	{
		callCount := 0
		gapIs := []int{}
		maxRetries := 4
		_, _ = GetOneDyn(func() (int, error) {
			callCount++
			return 0, errs.NewErrf("keep failing")
		}, func(i int, err error) (time.Duration, bool) {
			gapIs = append(gapIs, i)
			return 0, callCount < maxRetries
		})
		for idx, v := range gapIs {
			if v != idx+1 {
				t.Fatalf("expected gapIs[%v]=%v, got %v", idx, idx+1, v)
			}
		}
		cli.TPrintlnf("scenario 4 passed: gapIs=%v", gapIs)
	}
}

func TestGetOneWithBackoff(t *testing.T) {
	backoff := []time.Duration{time.Second, time.Millisecond * 500, time.Second}
	var gap time.Duration
	now := time.Now()
	err := CallWithBackoff(backoff, func() error {
		gap = time.Since(now)
		cli.TPrintlnf("call, gap: %v", gap)
		now = time.Now()
		return errs.NewErrf("no")
	})
	if err == nil {
		t.Fatal("err should not be nil")
	}
}
