package core

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
