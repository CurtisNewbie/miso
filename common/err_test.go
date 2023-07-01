package common

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func doSomething() error {
	return doSomethingElse()
}


func doSomethingElse() error {
	return TraceErr("doSomethingDeeper", doSomethingDeeper())
}

func doSomethingDeeper() error {
	// return errors.New("I am not feeling good")
	return NewTraceErr("I am ok but I will still return err anyway")
}


func TestTraceableError(t *testing.T) {
	err := TraceErr("doSomething failed", doSomething())
	logrus.Infof("%v", err)
}
