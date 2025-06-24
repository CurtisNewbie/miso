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
	t.Log(ErrorStackTrace(err))
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
	t.Logf("%#v\n%v", ErrBase, ErrorStackTrace(ErrBase))
	t.Logf("%#v\n%v", e1, ErrorStackTrace(e1))
	t.Logf("%#v\n%v", e2, ErrorStackTrace(e2))
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
	t.Log(ErrorStackTrace(e))
}

func testUnwrapErrStack1() error {
	st := NewErrf("oh no")
	return fmt.Errorf("wrapping oh no, %w", st)
}

func TestStack(t *testing.T) {
	s := stack(1)
	t.Log(s)
}

func TestUnknownErr(t *testing.T) {
	ne := errors.New("something is wrong")
	err := NewErrf("operation failed").Wrap(ne)
	t.Logf("%v", err)
	Errorf("%v", err)
}

func TestUnknownErrf(t *testing.T) {
	ne := someOp()
	err := UnknownErrf(ne, "operation failed, %v", "someContext")
	t.Logf("err: %v", err)
	Errorf("%v", err)
	t.Logf("Unwrapped: %v", errors.Unwrap(err))
	Errorf("wrap again: %v", UnknownErrf(err, "warping err"))
}

func someOp() error {
	return NewErrf("something is wrong")
}

func TestDirectUnknownErr(t *testing.T) {
	ne := someOp()
	err := UnknownErr(ne)
	t.Logf("err: %v", err)
	Errorf("%v", err)
	t.Logf("Unwrapped: %v", errors.Unwrap(err))
}

func TestWrapNilErr(t *testing.T) {
	wrp := UnknownErr(nil)
	if wrp != nil {
		t.Fatal("wrp != nil")
	}
}

func TestWrapMisoErr(t *testing.T) {
	me := ErrfCode("xxxx", "something is wrong")
	wrp := UnknownErr(me)
	if wrp == nil {
		t.Fatal("wrp == nil")
	}
	Errorf("me: %v", me)
	Errorf("wrp: %v", wrp)
	v, ok := wrp.(*MisoErr)
	if !ok {
		t.Fatal("wrp is not MisoErr")
	}
	if v.Code() != "xxxx" {
		t.Fatal("wrp is code is different")
	}

	wrp2 := UnknownErr(errors.New("oh no"))
	Errorf("wrp2: %v", wrp2)
}

func TestWrapErr(t *testing.T) {
	me := ErrfCode("xxxx", "something is wrong")
	wrp := WrapErr(me)
	if wrp == nil {
		t.Fatal("wrp == nil")
	}
	Errorf("me: %v", me)
	Errorf("wrp: %v", wrp)
	v, ok := wrp.(*MisoErr)
	if !ok {
		t.Fatal("wrp is not MisoErr")
	}
	if v.Code() != "xxxx" {
		t.Fatal("wrp is code is different")
	}

	wrp2 := WrapErr(errors.New("oh no"))
	Errorf("wrp2: %v", wrp2)

	wrp3 := UnknownErr(errors.New("oh no"))
	Errorf("wrp3: %v", wrp3)
}
