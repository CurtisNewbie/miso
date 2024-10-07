package util

import (
	"errors"
	"fmt"
	"sync"
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
	cnt := 1000
	pool := NewAsyncPool(cnt+1, 100)
	start := time.Now()
	var futures []Future[int]

	for i := 1; i < cnt+1; i++ {
		j := i
		futures = append(futures, SubmitAsync(pool, func() (int, error) {
			time.Sleep(5 * time.Millisecond)
			fmt.Printf("%v is done\n", j)
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
		t.Logf("Get future %d", i)
	}
	expected := (cnt * (cnt + 1)) / 2
	if sum != expected {
		t.Fatalf("expected: %v, actual: %v", expected, sum)
	}
	t.Logf("sum: %v, time: %v", sum, time.Since(start))
}

func TestRunAsyncWithPanic(t *testing.T) {
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
	Printlnf("about to panic")
	panic("panic func panicked")
}

func TestAwaitFutures(t *testing.T) {
	cnt := 1000
	pool := NewAsyncPool(cnt+1, 100)
	awaitFutures := NewAwaitFutures[int](pool)
	start := time.Now()

	for i := 1; i < cnt+1; i++ {
		j := i
		awaitFutures.SubmitAsync(func() (int, error) {
			time.Sleep(5 * time.Millisecond)
			return j, nil
		})
	}

	var futures []Future[int] = awaitFutures.Await()
	var sum int
	for _, fut := range futures {
		res, err := fut.Get()
		if err != nil {
			t.Fatal(err)
		}
		sum += res
	}
	expected := (cnt * (cnt + 1)) / 2
	if sum != expected {
		t.Fatalf("expected: %v, actual: %v", expected, sum)
	}
	t.Logf("sum: %v, time: %v", sum, time.Since(start))
}

func TestPoolPanic(t *testing.T) {
	pool := NewAsyncPool(1, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	pool.Go(func() {
		defer wg.Done()
		panic("oops")
	})

	wg.Add(1)
	pool.Go(func() {
		defer wg.Done()
		panic("oops")
	})

	wg.Add(1)
	pool.Go(func() {
		defer wg.Done()
		panic("oops")
	})
	wg.Wait()
}

func TestAsyncPoolStop(t *testing.T) {
	pool := NewAsyncPool(1, 10)
	for i := 0; i < 10; i++ {
		v := i
		pool.Go(func() {
			time.Sleep(time.Second)
			Printlnf("v: %v", v)
		})
	}
	pool.StopAndWait()
}

func TestAsyncOnce(t *testing.T) {
	f := RunAsync(func() (int, error) {
		Printlnf("async ran")
		return 1, nil
	})
	r, err := f.Get()
	t.Logf("1. r: %v, err: %v", r, err)

	r, err = f.Get()
	t.Logf("2. r: %v, err: %v", r, err)

	r, err = f.TimedGet(100)
	t.Logf("3. r: %v, err: %v", r, err)
}
