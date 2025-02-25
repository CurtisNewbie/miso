package miso

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewErr(t *testing.T) {
	err := NewErrf("unknown error, %v", "nope").WithInternalMsg("nope, that is not unknown error, that is %v", "fake error")
	s := fmt.Sprintf("nope, that is not unknown error, that is %v", "fake error")
	if s != err.InternalMsg() {
		t.Fatalf("%v != %v", s, err.InternalMsg())
	}
	if err.Error() != "unknown error, nope, nope, that is not unknown error, that is fake error" {
		t.Fatalf("%v", err.Error())
	}

	err = NewErrf("unknown error").WithInternalMsg("nope, that is not unknown error, that is ")
	s = "nope, that is not unknown error, that is "
	if s != err.InternalMsg() {
		t.Fatalf("%v != %v", s, err.InternalMsg())
	}
	if err.Error() != "unknown error, nope, that is not unknown error, that is " {
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

func TestWrapErr(t *testing.T) {
	ne := errors.New("something is wrong")
	err := NewErrf("operation failed").Wrap(ne)
	t.Logf("%v", err)
	Errorf("%v", err)
}

func TestWrapErrf(t *testing.T) {
	ne := someOp()
	err := WrapErrf(ne, "operation failed, %v", "someContext")
	t.Logf("err: %v", err)
	Errorf("%v", err)
	t.Logf("Unwrapped: %v", errors.Unwrap(err))
	Errorf("wrap again: %v", WrapErrf(err, "warping err"))
}

func someOp() error {
	return NewErrf("something is wrong")
}

func TestDirectWrapErr(t *testing.T) {
	ne := someOp()
	err := WrapErr(ne)
	t.Logf("err: %v", err)
	Errorf("%v", err)
	t.Logf("Unwrapped: %v", errors.Unwrap(err))
}
