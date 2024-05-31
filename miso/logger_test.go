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
	if v := unsafeGetShortFnName("shortFunc"); v != "shortFunc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("pck.shortFunc"); v != "pck.shortFunc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("vvvv/pck.shortFunc"); v != "pck.shortFunc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("gggg/vvvv/pck.shortFunc"); v != "pck.shortFunc" {
		t.Fatal(v)
	}
}

func TestGetShortFnName(t *testing.T) {
	if v := getShortFnName("shortFunc"); v != "shortFunc" {
		t.Fatal(v)
	}

	if v := getShortFnName("pck.shortFunc"); v != "pck.shortFunc" {
		t.Fatal(v)
	}

	if v := getShortFnName("vvvv/pck.shortFunc"); v != "pck.shortFunc" {
		t.Fatal(v)
	}

	if v := getShortFnName("gggg/vvvv/pck.shortFunc"); v != "pck.shortFunc" {
		t.Fatal(v)
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
