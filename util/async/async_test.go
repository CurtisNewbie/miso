package async

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/curtisnewbie/miso/util/utillog"
)

func TestRunAsync(t *testing.T) {
	start := time.Now()
	var futures []Future[int]

	n := 1000
	for i := 1; i < n+1; i++ {
		j := i
		futures = append(futures, Run(func() (int, error) {
			// time.Sleep(50 * time.Millisecond)
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
	expected := (n * (n + 1)) / 2
	if sum != expected {
		t.Fatalf("expected: %v, actual: %v", expected, sum)
	}
	t.Logf("sum: %v, time: %v", sum, time.Since(start))
}

func TestRunAsyncPool(t *testing.T) {
	cnt := 10000
	pool := NewBoundedAsyncPool(300, 150)
	// pool := NewAntsAsyncPool(150)
	start := time.Now()
	var futures []Future[int]

	for i := 1; i < cnt+1; i++ {
		j := i
		futures = append(futures, Submit(pool, func() (int, error) {
			time.Sleep(5 * time.Millisecond)
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
	expected := (cnt * (cnt + 1)) / 2
	if sum != expected {
		t.Fatalf("expected: %v, actual: %v", expected, sum)
	}
	t.Logf("sum: %v, time: %v", sum, time.Since(start))
}

func TestRunAsyncWithPanic(t *testing.T) {
	future := Run[struct{}](panicFunc)
	_, err := future.Get()
	if err == nil {
		t.Fatal("should return err")
	}
	t.Log(err)

	predefinedErr := errors.New("predefined panic error")
	future = Run[struct{}](func() (struct{}, error) {
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
	fmt.Println("about to panic")
	panic("panic func panicked")
}

func TestAwaitFutures(t *testing.T) {
	cnt := 1000
	pool := NewBoundedAsyncPool(cnt+1, 100)
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

func TestAwaitResultAnyErr(t *testing.T) {
	cnt := 1000
	pool := NewBoundedAsyncPool(cnt+1, 100)
	awaitFutures := NewAwaitFutures[int](pool)
	start := time.Now()

	for i := 1; i < cnt+1; i++ {
		j := i
		awaitFutures.SubmitAsync(func() (int, error) {
			time.Sleep(5 * time.Millisecond)
			return j, nil
		})
	}

	res, err := awaitFutures.AwaitResultAnyErr()
	if err != nil {
		t.Fatal(err)
	}
	var sum int
	for _, v := range res {
		sum += v
	}
	expected := (cnt * (cnt + 1)) / 2
	if sum != expected {
		t.Fatalf("expected: %v, actual: %v", expected, sum)
	}
	t.Logf("sum: %v, time: %v", sum, time.Since(start))
}

func TestPoolPanic(t *testing.T) {
	pool := NewBoundedAsyncPool(1, 10)
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
	pool := NewBoundedAsyncPool(1, 10)
	for i := 0; i < 10; i++ {
		v := i
		pool.Go(func() {
			start := time.Now().UnixMilli()
			time.Sleep(time.Second)
			fmt.Printf(time.Now().Format("2006-01-02 15:04:05.000")+" start: %v, v: %v\n", start, v)
		})
	}
	fmt.Println("Test pre stop")
	pool.StopAndWait()
	fmt.Println("Test post stop")

	for i := 10; i < 15; i++ {
		v := i
		pool.Go(func() {
			start := time.Now().UnixMilli()
			time.Sleep(time.Second)
			fmt.Printf(time.Now().Format("2006-01-02 15:04:05.000")+" start: %v, v: %v\n", start, v)
		})
	}

	t.Logf("Test end")
}

func TestAsyncOnce(t *testing.T) {
	f := Run(func() (int, error) {
		fmt.Println("async ran")
		time.Sleep(time.Millisecond * 500)
		return 1, nil
	})

	r, err := f.TimedGet(100)
	t.Logf("1. r: %v, err: %v", r, err)
	if err == nil {
		t.Fatal("should timeout")
	}

	r, err = f.Get()
	t.Logf("2. r: %v, err: %v", r, err)
	if err != nil {
		t.Fatal("should not err")
	}
	if r != 1 {
		t.Fatal("should be 1")
	}

	r, err = f.Get()
	t.Logf("3. r: %v, err: %v", r, err)
	if err != nil {
		t.Fatal("should not err")
	}
	if r != 1 {
		t.Fatal("should be 1")
	}
}

func TestFutureBeforeThen(t *testing.T) {
	f := Run(func() (int, error) {
		t.Logf("async ran")
		return 1, nil
	})
	time.Sleep(time.Millisecond * 100)
	var cnt int32 = 0

	f.Then(func(i int, err error) {
		atomic.AddInt32(&cnt, 1)
		t.Logf("1. r: %v, err: %v", i, err)
	})

	if atomic.LoadInt32(&cnt) < 1 {
		t.Fatalf("cnt should be 1, then callback not invoked")
	}
}

func TestFutureAfterThen(t *testing.T) {
	var cnt int32 = 0
	f := Run(func() (int, error) {
		t.Logf("async ran start")
		time.Sleep(time.Millisecond * 100)
		t.Logf("async ran end")
		return 1, nil
	})
	time.Sleep(time.Millisecond * 50)

	f.Then(func(i int, err error) {
		atomic.AddInt32(&cnt, 1)
		t.Logf("1. r: %v, err: %v", i, err)
	})
	t.Log("added Then")

	time.Sleep(time.Millisecond * 200)

	if atomic.LoadInt32(&cnt) < 1 {
		t.Fatalf("cnt should be 1, then callback not invoked")
	}
}

func TestFutureThenAndGet(t *testing.T) {
	var cnt int32 = 0
	f := Run(func() (int, error) {
		t.Logf("async ran")
		return 1, nil
	})

	f.Then(func(i int, err error) {
		atomic.AddInt32(&cnt, 1)
		t.Logf("1. r: %v, err: %v", i, err)
	})

	time.Sleep(time.Millisecond * 50)

	i, err := f.Get()
	t.Logf("2. r: %v, err: %v", i, err)

	if atomic.LoadInt32(&cnt) < 1 {
		t.Fatalf("cnt should be 1, then callback not invoked")
	}
}

func TestFutureGetAndThen(t *testing.T) {
	var cnt int32 = 0
	f := Run(func() (int, error) {
		t.Logf("async ran")
		return 1, nil
	})

	i, err := f.Get()
	t.Logf("1. r: %v, err: %v", i, err)

	f.Then(func(i int, err error) {
		atomic.AddInt32(&cnt, 1)
		t.Logf("2. r: %v, err: %v", i, err)
	})

	time.Sleep(time.Millisecond * 50)

	if atomic.LoadInt32(&cnt) < 1 {
		t.Fatalf("cnt should be 1, then callback not invoked")
	}
}

func TestFutureThenPanic(t *testing.T) {
	var cnt int32 = 0
	f := Run(func() (int, error) {
		t.Logf("async ran")
		return 1, nil
	})

	i, err := f.Get()
	t.Logf("1. r: %v, err: %v", i, err)

	f.Then(func(i int, err error) {
		atomic.AddInt32(&cnt, 1)
		t.Logf("2. r: %v, err: %v", i, err)

		panic("no no no")
	})

	time.Sleep(time.Millisecond * 50)

	if atomic.LoadInt32(&cnt) < 1 {
		t.Fatalf("cnt should be 1, then callback not invoked")
	}
}

func TestBatchTask(t *testing.T) {
	var mu sync.Mutex
	var sum int = 0

	n := 50
	batchTask := NewBatchTask[int, int](10, 10, func(v int) (int, error) {
		t.Logf("%v, running -> %v", time.Now().UnixMilli(), v)
		defer func() {
			t.Logf("%v, finished -> %v", time.Now().UnixMilli(), v)
		}()
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		sum += v
		return v, nil
	})

	shouldBe := n * (n + 1) / 2
	for i := 0; i <= n; i++ {
		batchTask.Generate(i)
	}
	t.Logf("waiting batchTask")

	result := batchTask.Wait()
	if sum != shouldBe {
		t.Fatalf("should be %v but %v", shouldBe, sum)
	}
	t.Logf("sum: %v", sum)

	resultSum := 0
	for _, v := range result {
		resultSum += v.Result
	}
	if resultSum != shouldBe {
		t.Fatalf("should be %v but %v", shouldBe, resultSum)
	}
	t.Logf("resultSum: %v", resultSum)
}

func TestAyncPoolFull(t *testing.T) {
	utillog.DebugLog = func(pat string, args ...any) { t.Logf(pat, args...) }
	ap := NewBoundedAsyncPool(0, 0, FallbackCallerRun())
	v := &atomic.Int32{}
	ap.Go(func() {
		t.Log("task 1")
		v.Add(1)
	})
	ap.Go(func() {
		t.Log("task 2")
		v.Add(1)
	})
	if av := v.Load(); av != 2 {
		t.Fatalf("task 1 and task 2 should run by caller, %v", av)
	}
	ap.StopAndWait()
	t.Log("---")

	ap = NewBoundedAsyncPool(0, 1, FallbackDropTask())
	v.Store(0)
	ap.Go(func() {
		t.Log("2 - task 1")
		time.Sleep(time.Second)
		v.Add(1)
	})
	ap.Go(func() {
		t.Log("2 - task 2")
		time.Sleep(time.Second)
		v.Add(1)
	})

	if v.Load() == 2 {
		t.Fatal("either task 1 or task 2 should be dropped")
	}
	ap.StopAndWait()
}

func TestAntsAyncPoolFull(t *testing.T) {
	utillog.DebugLog = func(pat string, args ...any) { t.Logf(pat, args...) }
	ap := NewAntsAsyncPool(1, FallbackCallerRun())
	v := &atomic.Int32{}
	var cnt int32 = 10
	for i := range cnt {
		j := i
		ap.Go(func() {
			t.Logf("task %v", j)
			time.Sleep(time.Second)
			v.Add(1)
		})
	}
	if av := v.Load(); av != cnt {
		t.Fatalf("task 1 and task 2 should be run, %v", av)
	}
	ap.StopAndWait()
}
