package common

import (
	"testing"
	"time"
)

func TestRunAsync(t *testing.T) {
	start := time.Now()
	var futures []Future[int]

	for i := 0; i < 1000; i++ {
		futures = append(futures, RunAsync(func() FutureResult[int] {
			time.Sleep(5 * time.Second)
			return FutureResult[int]{Result: 1, Err: nil}
		}))
	}

	for _, fut := range futures {
		res, err := fut.Get()
		if err != nil {
			t.Fatal(err)
		}
		if res != 1 {
			t.Fatal("not 1")
		}
	}
	t.Log(time.Since(start))
}
