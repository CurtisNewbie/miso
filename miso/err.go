package miso

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/curtisnewbie/miso/util"
)

var (
	// Error that represents None or Nil.
	//
	// Use miso.IsNoneErr(err) to check if an error represents None.
	NoneErr *MisoErr = NewErrf("none")
)

var (
	ErrCodeGeneric         string = "XXXX"
	ErrCodeUnknownError    string = "UNKNOWN_ERROR"
	ErrCodeNotPermitted    string = "NOT_PERMITTED"
	ErrCodeIllegalArgument string = "ILLEGAL_ARGUMENT"

	ErrUnknownError    *MisoErr = NewErrf("Unknown Error").WithCode(ErrCodeUnknownError)
	ErrNotPermitted    *MisoErr = NewErrf("Not Permitted").WithCode(ErrCodeNotPermitted)
	ErrIllegalArgument *MisoErr = NewErrf("Illegal Argument").WithCode(ErrCodeIllegalArgument)
)

// Check if the error represents None
func IsNoneErr(err error) bool {
	return errors.Is(err, NoneErr)
}

// Miso Error.
//
//	Use NewErrf(...) to instantiate.
type MisoErr struct {
	code        string // error code.
	msg         string // error message returned to the client requested to the endpoint.
	internalMsg string // internal message that is only logged on server.
	stack       string
	err         error
}

func (e *MisoErr) Cause() error {
	return e.err
}

func (e *MisoErr) InternalMsg() string {
	return e.internalMsg
}

func (e *MisoErr) Msg() string {
	return e.msg
}

func (e *MisoErr) Code() string {
	return e.code
}

func (e *MisoErr) StackTrace() string {
	return e.stack
}

// Create new *MisoErr to wrap the cause error
func (e *MisoErr) Wrap(cause error) *MisoErr {
	n := e.copyNew()
	n.withStack()
	return n
}

// Create new *MisoErr to wrap the cause error
func (e *MisoErr) Wrapf(cause error, internalMsg string, args ...any) *MisoErr {
	n := e.copyNew()
	n.withStack()
	if len(args) > 0 {
		n.internalMsg = fmt.Sprintf(internalMsg, args...)
	} else {
		n.internalMsg = internalMsg
	}
	return n
}

func (e *MisoErr) copyNew() *MisoErr {
	n := new(MisoErr)
	n.code = e.code
	n.msg = e.msg
	n.internalMsg = e.internalMsg
	n.stack = e.stack
	n.err = e.err
	return n
}

func (e *MisoErr) Error() string {
	uw := e.Unwrap()
	if uw == nil {
		return e.msg
	}
	if e.msg == "" {
		return uw.Error()
	}
	return e.msg + ", " + uw.Error()
}

func (e *MisoErr) HasCode() bool {
	return !util.IsBlankStr(e.code)
}

func (e *MisoErr) WithCode(code string) *MisoErr {
	n := e.copyNew()
	n.code = code
	return n
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
	if tme, ok := target.(*MisoErr); ok && e.code != "" && e.code == tme.code {
		return true
	}
	return false
}

func (e *MisoErr) WithInternalMsg(msg string, args ...any) *MisoErr {
	ne := e.copyNew()
	if len(args) > 0 {
		ne.internalMsg = fmt.Sprintf(msg, args...)
	} else {
		ne.internalMsg = msg
	}
	return ne
}

func (e *MisoErr) withStack() *MisoErr {
	e.stack = stack(3)
	return e
}

func (e *MisoErr) Unwrap() error {
	return e.err
}

var NewErrf = Errf

// Create new MisoErr with message.
func Errf(msg string, args ...any) *MisoErr {
	me := &MisoErr{msg: msg, internalMsg: "", err: nil}
	me.withStack()
	return me
}

// Create new MisoErr with message and error code.
func ErrfCode(code string, msg string, args ...any) *MisoErr {
	me := &MisoErr{msg: msg, internalMsg: "", err: nil, code: code}
	me.withStack()
	return me
}

// Wrap an error to create new MisoErr without any extra context.
//
// This is equivalent to ErrUnknownError.Wrap(err)
func WrapErr(err error) *MisoErr {
	return ErrUnknownError.Wrap(err)
}

// Wrap an error to create new MisoErr with message.
func WrapErrf(err error, msg string, args ...any) *MisoErr {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	me := &MisoErr{msg: msg, internalMsg: "", err: err}
	me.withStack()
	return me
}

// Wrap an error to create new MisoErr with message.
//
// If the wrapped err is nil, nil is returned.
func WrapErrfCode(err error, code string, msg string, args ...any) *MisoErr {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	me := &MisoErr{msg: msg, internalMsg: "", err: err, code: code}
	me.withStack()
	return me
}

func UnwrapErrStack(err error) (string, bool) {
	var stack string
	var ue error = err
	for {
		if me, ok := ue.(*MisoErr); ok {
			if me != nil {
				stack = me.stack
			}
		}
		u := errors.Unwrap(ue)
		if u == nil {
			break
		}
		ue = u
	}

	return stack, stack != ""
}

var stackPool = sync.Pool{
	New: func() any {
		var v []uintptr = make([]uintptr, 50)
		return &v
	},
}

func stack(n int) string {
	stack := stackPool.Get().(*[]uintptr)
	defer func() {
		clear(*stack)
		stackPool.Put(stack)
	}()

	length := runtime.Callers(n, *stack)
	frames := runtime.CallersFrames((*stack)[:length])
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
