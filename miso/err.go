package miso

import (
	"errors"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	gopath = build.Default.GOPATH
	src    = filepath.Join(gopath, "src")
)

var (
	NoneErr = NewErr("Miso: None")
)

// Check if the error represents None
func IsNoneErr(err error) bool {
	return errors.Is(err, NoneErr)
}

// Miso Error
type MisoErr struct {
	Code        string // error code
	Msg         string // error message returned to web client
	InternalMsg string // internal message only logged on server
}

func (e *MisoErr) Error() string {
	return e.Msg
}

// Create new MisoErr
func NewErr(msg string, args ...any) *MisoErr {
	var im string
	l := len(args)
	if l > 1 {
		im = fmt.Sprintf(fmt.Sprintf("%v", args[0]), args[1:]...)
	} else if l > 0 {
		im = fmt.Sprintf("%v", args[0])
	}
	return &MisoErr{Msg: msg, InternalMsg: im}
}

// Create new MisoErr with code
func NewErrCode(code string, msg string, args ...any) *MisoErr {
	err := NewErr(msg, args...)
	err.Code = code
	return err
}

// Check if MisoErr has a specified code
func HasCode(e *MisoErr) bool {
	return !IsBlankStr(e.Code)
}

func srcPath(filename string) string {
	return strings.TrimPrefix(filename, fmt.Sprintf("%s%s", src, string(os.PathSeparator)))
}

type TraceableError struct {
	cause error
	file  string
	line  int
	msg   string
}

func (e *TraceableError) FormatedError() string {
	if e.cause != nil {
		if tr, ok := e.cause.(*TraceableError); ok {
			return fmt.Sprintf("%v:%v %v\n\t%v", e.file, e.line, e.msg, tr.FormatedError())
		}
		return fmt.Sprintf("%v:%v %v, \n\t%v", e.file, e.line, e.msg, e.cause)
	}
	return fmt.Sprintf("%v:%v %v", e.file, e.line, e.msg)
}

func (e *TraceableError) Error() string {
	if e.cause == nil {
		return e.msg
	}

	if tr, ok := e.cause.(*TraceableError); ok {
		return fmt.Sprintf("%v\n\t%v", e.msg, tr.FormatedError())
	}
	return fmt.Sprintf("%v, %v", e.msg, e.cause)
}

func TraceErrf(err error, msg string, param ...any) error {
	t := new(TraceableError)
	t.cause = err
	_, file, line, _ := runtime.Caller(1)
	t.file = srcPath(file)
	t.line = line
	t.msg = fmt.Sprintf(msg, param...)
	return t
}

func NewTraceErrf(msg string, param ...any) error {
	t := new(TraceableError)
	_, file, line, _ := runtime.Caller(1)
	t.file = srcPath(file)
	t.line = line
	t.msg = fmt.Sprintf(msg, param...)
	return t
}
