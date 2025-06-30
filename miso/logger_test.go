package miso

import (
	"sync/atomic"
	"testing"
	"time"
)

func determineIdealMethodName() {
	Info("Whispering")
	Debug("Whispering ???? :D")
}

func TestGetCallerFn(t *testing.T) {
	Info("yo")
	determineIdealMethodName()

	EmptyRail().Info("oops")
}

func BenchmarkDebugf(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Debugf("abc, %v", 1)
	}
}

func TestUnsafeGetShortFnName(t *testing.T) {
	if v := unsafeGetShortFnName(""); v != "" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("shortFunc"); v != "shortFunc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("pck.shortFunc"); v != "pck.shortFunc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("vvvv/pck.shortFunc"); v != "pck.shortFunc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("gggg/vv漢字vv/pck.(*abc).shortFunc"); v != "(*abc).shortFunc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("gggg/vv漢字vv/pck.(*abc).shortFunc.func2"); v != "(*abc).shortFunc.func2" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("gggg/vv漢字vv/(*abc).shortFunc.func2"); v != "(*abc).shortFunc.func2" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("gggg/vv漢字vv/pck.(*abc).shortFunc.func2.func"); v != "(*abc).shortFunc.func2.func" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("gggg/vv漢字vv/pck.(*abc).shortFunc.funcA.funcB.do"); v != "funcA.funcB.do" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("[...]abc"); v != "[...]abc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("[..]abc"); v != "[..]abc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("..]abc"); v != "..]abc" {
		t.Fatal(v)
	}

}

func BenchmarkUnsafeGetShortFnName(b *testing.B) {
	/*
		goos: darwin
		goarch: arm64
		pkg: github.com/curtisnewbie/miso/miso
		cpu: Apple M3 Pro
		=== RUN   BenchmarkUnsafeGetShortFnName
		BenchmarkUnsafeGetShortFnName
		BenchmarkUnsafeGetShortFnName-11        203311693                5.792 ns/op           0 B/op          0 allocs/op
	*/
	for range b.N {
		unsafeGetShortFnName("gggg/vvvv/pck.(*abc).shortFunc")
	}
}

func BenchmarkInfo(b *testing.B) {
	rail := EmptyRail()
	rail.Info("abc")
	b.ResetTimer()

	// 1. original, sprintf version
	// 1806 B/op         23 allocs/op
	//
	// 2. bytes.Buffer handwrote formatting + buffer pooling
	// 1587 B/op         16 allocs/op
	//
	// 3. getCallerFn, getShortFnName optimization
	// 1227 B/op         12 allocs/op
	for i := 0; i < b.N; i++ {
		rail.Info("abc")
	}
}

func BenchmarkError(b *testing.B) {
	rail := EmptyRail()

	var n int64
	if ok := SetErrLogHandler(func(el *ErrorLog) {
		atomic.AddInt64(&n, 1)
	}); !ok {
		b.Fatal("not ok")
	}

	b.Run("yes", func(b *testing.B) {
		// 1253 B/op         13 allocs/op
		for i := 0; i < b.N; i++ {
			rail.Error("abc")
		}
	})
	time.Sleep(1 * time.Second)
	b.Logf("n: %d", atomic.LoadInt64(&n))
}
