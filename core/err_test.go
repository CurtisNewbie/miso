package core

import (
	"errors"
	"fmt"
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

func TestNewWebErr(t *testing.T) {
	err := NewWebErr("unknown error", "nope, that is not unknown error, that is %v", "fake error")
	TestEqual(t, fmt.Sprintf("nope, that is not unknown error, that is %v", "fake error"), err.InternalMsg)
	TestEqual(t, "unknown error", err.Error())

	err = NewWebErr("unknown error", "nope, that is not unknown error, that is %v")
	TestEqual(t, "nope, that is not unknown error, that is %v", err.InternalMsg)
	TestEqual(t, "unknown error", err.Error())
}
