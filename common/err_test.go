package common

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func doSomething() error {
	return doSomethingElse()
}

func doSomethingElse() error {
	return TraceErrf(doSomethingDeeper(), "doSomethingDeeper")
}

func doSomethingDeeper() error {
	// return errors.New("I am not feeling good")
	return NewTraceErrf("I am ok but I will still return err anyway")
}

func TestTraceableError(t *testing.T) {
	err := TraceErrf(doSomething(), "doSomething failed, %v", ":(")
	logrus.Infof("%v", err)
}
