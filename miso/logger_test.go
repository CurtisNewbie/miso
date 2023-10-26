package miso

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestFormatter(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	logrus.Info("test message")
	determineIdealMethodName()
}

func determineIdealMethodName() {
	logrus.Info("Whispering")
	logrus.Debug("Whispering ???? :D")
}
