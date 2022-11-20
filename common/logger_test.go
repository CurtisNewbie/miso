package common

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestFormatter(t *testing.T) {
	logrus.Info("test message")
	someSecretFunc()
}

func someSecretFunc() {
	logrus.Info("Whispering")
}