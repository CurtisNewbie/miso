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

func (e *MisoErr) WithInternalMsg(msg string, args ...any) *MisoErr {
	if len(args) > 0 {
		e.InternalMsg = fmt.Sprintf(msg, args...)
	} else {
		e.InternalMsg = msg
	}
	return e
}

// Create new MisoErr with message.
func NewErrf(msg string, args ...any) *MisoErr {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return &MisoErr{Msg: msg, InternalMsg: ""}
}
