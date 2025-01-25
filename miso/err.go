package miso

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/curtisnewbie/miso/util"
)

var (
	// Error that represents None or Nil.
	//
	// Use miso.IsNoneErr(err) to check if an error represents None.
	NoneErr *MisoErr = NewErrf("none")

	disableErrStack = atomic.Bool{}
)

var (
	ErrUnknownError    *MisoErr = NewErrf("Unknown Error")
	ErrNotPermitted    *MisoErr = NewErrf("Not Permitted")
	ErrIllegalArgument *MisoErr = NewErrf("Illegal Argument")
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
	stack       string
	err         error
}

func (e *MisoErr) StackTrace() string {
	return e.stack
}

func (e *MisoErr) Wrap(cause error) *MisoErr {
	e.err = cause
	e.withStack()
	return e
}

func (e *MisoErr) WrapNew(cause error) *MisoErr {
	n := new(MisoErr)
	n.Code = e.Code
	n.Msg = e.Msg
	n.InternalMsg = e.InternalMsg
	n.err = cause
	n.withStack()
	return n
}

func (e *MisoErr) Error() string {
	uw := e.Unwrap()
	if uw == nil {
		return e.Msg
	}
	return e.Msg + ", " + uw.Error()
}

func (e *MisoErr) HasCode() bool {
	return !util.IsBlankStr(e.Code)
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

func (e *MisoErr) withStack() *MisoErr {
	if !disableErrStack.Load() {
		e.stack = stack(3)
	}
	return e
}

func (e *MisoErr) Unwrap() error {
	return e.err
}

var NewErrf = Errf

// Create new MisoErr with message.
func Errf(msg string, args ...any) *MisoErr {
	me := &MisoErr{Msg: msg, InternalMsg: "", err: nil}
	me.withStack()
	return me
}

// Wrap an error to create new MisoErr with message.
//
// Last argument must be error, and the error will wrapped internally by MisoErr. Call to MisoErr.Unwrap() will unwrap this error argument.
//
// The last argument (i.e., the wrapped error) will not be used as the argument for message formatting.
//
// E.g.,
//
//	var myId string = "123" // some context information
//	var rootCauseErr error = someOp(myId)
//	if rootCauseErr != nil {
//		return miso.WrapErrf("someOp failed, id: %v", myId, rootCauseErr)
//	}
func WrapErrf(msg string, args ...any) *MisoErr {
	var werr error = nil
	if len(args) > 0 {
		if lerr, ok := args[len(args)-1].(error); ok {
			werr = lerr
			msg = fmt.Sprintf(msg, args[:len(args)-1]...)
		} else {
			msg = fmt.Sprintf(msg, args...)
		}
	}
	me := &MisoErr{Msg: msg, InternalMsg: "", err: werr}
	me.withStack()
	return me
}

func UnwrapErrStack(err error) (string, bool) {
	var stack string
	var ue error = err
	for {
		if me, ok := ue.(*MisoErr); ok {
			stack = me.stack
		}
		u := errors.Unwrap(ue)
		if u == nil {
			break
		}
		ue = u
	}

	return stack, stack != ""
}

func DisableErrStack() {
	disableErrStack.Store(true)
}

func stack(n int) string {
	stack := make([]uintptr, 50)
	length := runtime.Callers(n, stack)
	frames := runtime.CallersFrames(stack[:length])
	b := strings.Builder{}

	for {
		f, next := frames.Next()
		if !next {
			break
		}
		b.WriteString(fmt.Sprintf("\n\t%v\n\t\t%v:%v", f.Function, f.File, f.Line))
	}
	return b.String()
}
