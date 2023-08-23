package core

import (
	"errors"
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
	yetAnotherErr := deepestStuff()
	return TraceErrf(yetAnotherErr, "I am ok but I will still return err anyway")
}

func deepestStuff() error {
	return errors.New("the deepest mistake")
}

func TestTraceableError(t *testing.T) {
	err := TraceErrf(doSomething(), "doSomething failed, %v", ":(")
	logrus.Infof("%v", err)
}
