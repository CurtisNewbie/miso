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
