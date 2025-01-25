package miso

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestNewErr(t *testing.T) {
	err := NewErrf("unknown error").WithInternalMsg("nope, that is not unknown error, that is %v", "fake error")
	s := fmt.Sprintf("nope, that is not unknown error, that is %v", "fake error")
	if s != err.InternalMsg {
		t.Fatalf("%v != %v", s, err.InternalMsg)
	}
	if err.Error() != "unknown error" {
		t.Fatalf("%v != 'unknown error'", err.Error())
	}

	err = NewErrf("unknown error").WithInternalMsg("nope, that is not unknown error, that is ")
	s = "nope, that is not unknown error, that is "
	if s != err.InternalMsg {
		t.Fatalf("%v != %v", s, err.InternalMsg)
	}
	if err.Error() != "unknown error" {
		t.Fatalf("%v != 'unknown error'", err.Error())
	}
}

func TestErrReuse(t *testing.T) {
	var ErrBase = NewErrf("Base Error").WithCode("BASE_ERROR")
	var e1 = ErrBase.WithInternalMsg("something happens")
	var e2 = ErrBase.WithInternalMsg("system is cracked")
	if !errors.Is(ErrBase, ErrBase) {
		t.Fatal("ErrBase should be ErrBase")
	}
	if !errors.Is(e1, ErrBase) {
		t.Fatal("e1 should be ErrBase")
	}
	if !errors.Is(e2, ErrBase) {
		t.Fatal("e2 should be ErrBase")
	}
	t.Logf("%#v", ErrBase)
	t.Logf("%#v", e1)
	t.Logf("%#v", e2)
}

func TestUnwrapErrStack(t *testing.T) {
	e := fmt.Errorf("something is wrong, %w", testUnwrapErrStack1())
	stack, ok := UnwrapErrStack(e)
	if !ok {
		t.Fatal("not ok")
	}
	if stack == "" {
		t.Fatal("stack is empty")
	}
	t.Log(e)
	t.Log(stack)
}

func testUnwrapErrStack1() error {
	st := NewErrf("oh no")
	return fmt.Errorf("wrapping oh no, %w", st)
}

func TestStack(t *testing.T) {
	s := stack(1)
	t.Log(s)
}

func TestMultiErr(t *testing.T) {
	err1, err2, err3 := errors.New("err1"), errors.New("err2"), errors.New("err3")
	merr := NewErrf("something is wrong, %v, %v", err1, err2)
	t.Logf("merr: %v", merr)
	t.Logf("unwrapped: %v, %v", merr.Unwrap(), reflect.TypeOf(merr.Unwrap()))
	if !errors.Is(merr, err1) {
		t.Fatal("should be err1")
	}
	if !errors.Is(merr, err2) {
		t.Fatal("should be err2")
	}
	if errors.Is(merr, err3) {
		t.Fatal("should not be err3")
	}
}

func TestWrapErr(t *testing.T) {
	ne := errors.New("something is wrong")
	err := NewErrf("operation failed").Wrap(ne)
	t.Logf("%v", err)
	Errorf("%v", err)
}

func TestWrapErrf(t *testing.T) {
	ne := someOp()
	err := WrapErrf("operation failed, %v", "someContext", ne)
	t.Logf("err: %v", err)
	Errorf("%v", err)
	t.Logf("Unwrapped: %v", err.Unwrap())
}

func someOp() error {
	return NewErrf("something is wrong")
}
