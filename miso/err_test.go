package miso

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

func TestNewErr(t *testing.T) {
	err := NewErr("unknown error", "nope, that is not unknown error, that is %v", "fake error")
	s := fmt.Sprintf("nope, that is not unknown error, that is %v", "fake error")
	if s != err.InternalMsg {
		t.Logf("%v != %v", s, err.InternalMsg)
	}
	if err.Error() != "unknown error" {
		t.Logf("%v != 'unknown error'", err.Error())
	}

	err = NewErr("unknown error", "nope, that is not unknown error, that is %v")

	s = "nope, that is not unknown error, that is %v"
	if s != err.InternalMsg {
		t.Logf("%v != %v", s, err.InternalMsg)
	}
	if err.Error() != "unknown error" {
		t.Logf("%v != 'unknown error'", err.Error())
	}
}
