package miso

import (
	"github.com/curtisnewbie/miso/util/err"
)

var (
	// Error that represents None or Nil.
	//
	// Use miso.IsNoneErr(err) to check if an error represents None.
	NoneErr = err.NoneErr
)

const (
	ErrCodeGeneric            string = err.ErrCodeGeneric
	ErrCodeUnknownError       string = err.ErrCodeUnknownError
	ErrCodeNotPermitted       string = err.ErrCodeNotPermitted
	ErrCodeIllegalArgument    string = err.ErrCodeIllegalArgument
	ErrCodeServerShuttingDown string = err.ErrCodeServerShuttingDown
)

var (
	ErrUnknownError       = err.ErrUnknownError
	ErrNotPermitted       = err.ErrNotPermitted
	ErrIllegalArgument    = err.ErrIllegalArgument
	ErrServerShuttingDown = err.ErrServerShuttingDown
)

var (
	// Deprecated: use [err.NewErrf] instead.
	Errf            = err.NewErrf
	NewErrf         = err.NewErrf
	IsNoneErr       = err.IsNoneErr
	NewErrfCode     = err.NewErrfCode
	ErrfCode        = err.ErrfCode
	UnknownErr      = err.UnknownErr
	WrapErr         = err.WrapErr
	WrapErrMulti    = err.WrapErrMulti
	UnknownErrf     = err.UnknownErrf
	UnknownErrMsgf  = err.UnknownErrMsgf
	WrapErrf        = err.WrapErrf
	WrapErrfCode    = err.WrapErrfCode
	UnwrapErrStack  = err.UnwrapErrStack
	ErrorStackTrace = err.ErrorStackTrace
)

type MisoErr = err.MisoErr
