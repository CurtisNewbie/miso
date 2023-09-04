package core

import (
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

// Web Error
type WebError struct {
	Code        string
	Msg         string
	InternalMsg string
	hasCode     bool
}

func (e *WebError) Error() string {
	return e.Msg
}

// Create new WebError
func NewWebErr(msg string, args ...any) *WebError {
	var im string
	l := len(args)
	if l > 1 {
		im = fmt.Sprintf(fmt.Sprintf("%v", args[0]), args[1:]...)
	} else if l > 0 {
		im = fmt.Sprintf("%v", args[0])
	}
	return &WebError{Msg: msg, hasCode: false, InternalMsg: im}
}

// Create new WebError with code
func NewWebErrCode(code string, msg string, internalMsg ...string) *WebError {
	var im string
	if len(internalMsg) > 0 {
		im = internalMsg[0]
	}
	return &WebError{Msg: msg, Code: code, hasCode: true, InternalMsg: im}
}

// Check if WebError has a specified code
func HasCode(e *WebError) bool {
	return e.hasCode
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
