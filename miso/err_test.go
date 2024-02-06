package miso

import (
	"fmt"
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
