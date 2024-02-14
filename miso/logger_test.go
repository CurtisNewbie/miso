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
