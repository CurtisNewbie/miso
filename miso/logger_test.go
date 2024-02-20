package miso

import (
	"testing"
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

func BenchmarkInfo(b *testing.B) {
	rail := EmptyRail()
	rail.Info("abc")
	b.ResetTimer()

	// 1. original, sprintf version
	// 1806 B/op         23 allocs/op

	// 2. bytes.Buffer handwrote formatting + buffer pooling
	// 1587 B/op         16 allocs/op
	for i := 0; i < b.N; i++ {
		rail.Info("abc")
	}
}
