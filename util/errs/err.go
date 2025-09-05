package errs

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

const (
	ErrCodeGeneric            string = "XXXX"
	ErrCodeUnknownError       string = "UNKNOWN_ERROR"
	ErrCodeNotPermitted       string = "NOT_PERMITTED"
	ErrCodeIllegalArgument    string = "ILLEGAL_ARGUMENT"
	ErrCodeServerShuttingDown string = "SERVER_SHUTTING_DOWN"
)

var (
	ErrUnknownError       *MisoErr = NewErrfCode(ErrCodeUnknownError, "Unknown Error")
	ErrNotPermitted       *MisoErr = NewErrfCode(ErrCodeNotPermitted, "Not Permitted")
	ErrIllegalArgument    *MisoErr = NewErrfCode(ErrCodeIllegalArgument, "Illegal Argument")
	ErrServerShuttingDown *MisoErr = NewErrfCode(ErrCodeServerShuttingDown, "Server Shutting Down")
)

var (
	Errf = NewErrf
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
//
// if cause is nil, nil is returned.
func (e *MisoErr) Wrap(cause error) error {
	if cause == nil {
		return nil
	}
	n := e.copyNew()
	n.err = cause
	n.withStack()
	return n
}

// Create new *MisoErr to wrap the cause error
//
// if cause is nil, nil is returned.
func (e *MisoErr) Wrapf(cause error, internalMsg string, args ...any) error {
	if cause == nil {
		return nil
	}
	n := e.copyNew()
	n.err = cause
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

func (e *MisoErr) New() error {
	n := new(MisoErr)
	n.code = e.code
	n.msg = e.msg
	n.internalMsg = e.internalMsg
	n.err = e.err
	n.withStack()
	return n
}

func (e *MisoErr) Error() string {

	tok := []string{}
	if e.msg != "" {
		tok = append(tok, e.msg)
	}
	if e.internalMsg != "" {
		tok = append(tok, e.internalMsg)
	}
	uw := e.Unwrap()
	if uw != nil {
		tok = append(tok, uw.Error())
	}
	return strings.Join(tok, ", ")
}

func (e *MisoErr) HasCode() bool {
	return !util.IsBlankStr(e.code)
}

func (e *MisoErr) WithCode(code string) *MisoErr {
	n := e.copyNew()
	n.code = code
	return n
}

func (e *MisoErr) WithMsg(msg string) *MisoErr {
	n := e.copyNew()
	n.msg = msg
	n.withStack()
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
	ne.withStack()
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

// Create new *MisoErr with message.
func NewErrf(msg string, args ...any) *MisoErr {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	me := &MisoErr{msg: msg, internalMsg: "", err: nil}
	me.withStack()
	return me
}

// Create new *MisoErr with message and error code.
func NewErrfCode(code string, msg string, args ...any) *MisoErr {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	me := &MisoErr{msg: msg, internalMsg: "", err: nil, code: code}
	me.withStack()
	return me
}

// Deprecated: Use NewErrfCode() instead
func ErrfCode(code string, msg string, args ...any) *MisoErr {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	me := &MisoErr{msg: msg, internalMsg: "", err: nil, code: code}
	me.withStack()
	return me
}

// Wrap an error to create new *MisoErr without any extra context.
//
// This is almost equivalent to ErrUnknownError.Wrap(err)
//
// If err is nil, nil is returned.
func UnknownErr(err error) error {
	return ErrUnknownError.Wrap(err)
}

// Wrap an error to create new *MisoErr with stacktrace.
//
// If err is nil, nil is returned.
//
// If err is *MisoErr, err is returned directly.
func WrapErr(err error) error {
	if err == nil {
		return nil
	}
	if me, ok := err.(*MisoErr); ok {
		return me
	}
	me := &MisoErr{msg: "", internalMsg: "", err: err, code: ""}
	me.withStack()
	return me
}

// Wrap multi errors to create new *MisoErr with stacktrace.
//
// If err is nil, nil is returned.
//
// If err is *MisoErr, err is returned directly.
func WrapErrMulti(errs ...error) error {
	if len(errs) < 1 {
		return nil
	}
	errs = util.Filter(errs, func(err error) bool { return err != nil })
	if len(errs) < 1 {
		return nil
	}
	me := &MisoErr{msg: "", internalMsg: "", err: errors.Join(errs...), code: ""}
	me.withStack()
	return me
}

// Equivalent to ErrUnknownError.Wrapf(..).
//
// If err is nil, nil is returned.
func UnknownErrf(err error, msg string, args ...any) error {
	return ErrUnknownError.Wrapf(err, msg, args...)
}

// Equivalent to ErrUnknownError.WithInternalMsg(msg, args...).
func UnknownErrMsgf(msg string, args ...any) error {
	return ErrUnknownError.WithInternalMsg(msg, args...)
}

// Wrap an error to create new MisoErr with message.
//
// If the wrapped err is nil, nil is returned.
func WrapErrf(err error, msg string, args ...any) error {
	if err == nil {
		return nil
	}
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
func WrapErrfCode(err error, code string, msg string, args ...any) error {
	if err == nil {
		return nil
	}
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

func ErrorStackTrace(err error) string {
	if err == nil {
		return "nil"
	}
	stackTrace, withStack := UnwrapErrStack(err)
	m := err.Error()
	if withStack {
		m += stackTrace
	}
	return m
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
