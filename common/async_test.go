package common

import (
	"testing"
	"time"
)

func TestRunAsync(t *testing.T) {
	start := time.Now()
	var futures []Future[int]

	for i := 1; i < 1001; i++ {
		j := i
		futures = append(futures, RunAsync(func() (int, error) {
			time.Sleep(50 * time.Millisecond)
			t.Logf("%v is done", j)
			return j, nil
		}))
	}

	var sum int
	for _, fut := range futures {
		res, err := fut.Get()
		if err != nil {
			t.Fatal(err)
		}
		sum += res
	}
	expected := (1000 * (1000 + 1)) / 2
	if sum != expected {
		t.Fatalf("expected: %v, actual: %v", expected, sum)
	}
	t.Logf("sum: %v, time: %v", sum, time.Since(start))
}
