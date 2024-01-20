package miso

import (
	"errors"
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

func TestRunAsyncPool(t *testing.T) {
	SetLogLevel("debug")
	cnt := 10000
	pool := NewAsyncPool(cnt+1, 100)
	start := time.Now()
	var futures []Future[int]

	for i := 1; i < cnt+1; i++ {
		j := i
		futures = append(futures, RunAsyncPool(pool, func() (int, error) {
			time.Sleep(5 * time.Millisecond)
			Infof("%v is done", j)
			return j, nil
		}))
	}

	var sum int
	for i, fut := range futures {
		res, err := fut.Get()
		if err != nil {
			t.Fatal(err)
		}
		sum += res
		Infof("Get future %d", i)
	}
	expected := (cnt * (cnt + 1)) / 2
	if sum != expected {
		t.Fatalf("expected: %v, actual: %v", expected, sum)
	}
	Infof("sum: %v, time: %v", sum, time.Since(start))
}

func TestRunAsyncWithPanic(t *testing.T) {
	SetLogLevel("debug")

	future := RunAsync[struct{}](panicFunc)
	_, err := future.Get()
	if err == nil {
		t.Fatal("should return err")
	}
	t.Log(err)

	predefinedErr := errors.New("predefined panic error")
	future = RunAsync[struct{}](func() (struct{}, error) {
		t.Log("about to panic")
		panic(predefinedErr)
	})
	_, err = future.Get()
	if err == nil {
		t.Fatal("should return err")
	}
	if !errors.Is(err, predefinedErr) {
		t.Fatalf("wrong error, %v", err)
	}
	t.Log(err)
}

func panicFunc() (struct{}, error) {
	Info("about to panic")
	panic("panic func panicked")
}
