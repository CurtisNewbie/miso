package miso

import (
	"errors"
	"fmt"
)

var (
	// Error that represents None or Nil.
	//
	// Use miso.IsNoneErr(err) to check if an error represents None.
	NoneErr *MisoErr = NewErrf("none")
)

// Check if the error represents None
func IsNoneErr(err error) bool {
	return errors.Is(err, NoneErr)
}

// Miso Error.
//
//	Use NewErrf(...) to instantiate.
type MisoErr struct {
	Code        string // error code.
	Msg         string // error message returned to the client requested to the endpoint.
	InternalMsg string // internal message that is only logged on server.
}

func (e *MisoErr) Error() string {
	return e.Msg
}

func (e *MisoErr) HasCode() bool {
	return !IsBlankStr(e.Code)
}

func (e *MisoErr) WithCode(code string) *MisoErr {
	e.Code = code
	return e
}

// Implements *MisoErr Is check.
//
// Returns true, if both are *MisoErr and the code matches.
//
// WithInternalMsg always create new error, so we can basically
// reuse the same error created using 'miso.NewErrf(...).WithCode(...)'
//
//	var ErrIllegalArgument = miso.NewErrf(...).WithCode(...)
//
//	var e1 = ErrIllegalArgument.WithInternalMsg(...)
//	var e2 = ErrIllegalArgument.WithInternalMsg(...)
//
//	errors.Is(e1, ErrIllegalArgument)
//	errors.Is(e2, ErrIllegalArgument)
func (e *MisoErr) Is(target error) bool {
	if tme, ok := target.(*MisoErr); ok && e.Code != "" && e.Code == tme.Code {
		return true
	}
	return false
}

func (e *MisoErr) WithInternalMsg(msg string, args ...any) *MisoErr {
	ne := new(MisoErr)
	ne.Code = e.Code
	ne.Msg = e.Msg
	if len(args) > 0 {
		ne.InternalMsg = fmt.Sprintf(msg, args...)
	} else {
		ne.InternalMsg = msg
	}
	return ne
}

// Create new MisoErr with message.
func NewErrf(msg string, args ...any) *MisoErr {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return &MisoErr{Msg: msg, InternalMsg: ""}
}
