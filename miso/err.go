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
	// Unknown Error
	ErrUnknownError *MisoErr = NewErrf("Unknown Error")

	// Not Permitted
	ErrNotPermitted *MisoErr = NewErrf("Not Permitted")
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
	errs        []error
}

func (e *MisoErr) Error() string {
	return e.Msg
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
	l := len(e.errs)
	if l < 1 {
		return nil
	}
	if l == 1 {
		return e.errs[0]
	}
	return &joinError{errs: util.SliceCopy(e.errs)}
}

type joinError struct {
	errs []error
}

func (e *joinError) Error() string {
	s := make([]string, 0, len(e.errs))
	for _, er := range e.errs {
		s = append(s, er.Error())
	}
	return strings.Join(s, ", ")
}

func (e *joinError) Unwrap() []error {
	return e.errs
}

// Create new MisoErr with message.
func NewErrf(msg string, args ...any) *MisoErr {
	errs := []error{}
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
		for _, ar := range args {
			if ae, ok := ar.(error); ok {
				errs = append(errs, ae)
			}
		}
	}
	me := &MisoErr{Msg: msg, InternalMsg: "", errs: errs}
	me.withStack()
	return me
}

func UnwrapErrStack(err error) (string, bool) {
	var ue error = err
	for {
		u := errors.Unwrap(ue)
		if u == nil {
			break
		}
		ue = u
	}
	if me, ok := ue.(*MisoErr); ok {
		return me.stack, me.stack != ""
	}
	return "", false
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
