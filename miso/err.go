package miso

import (
	"github.com/curtisnewbie/miso/util/errs"
)

var (
	// Error that represents None or Nil.
	//
	// Use miso.IsNoneErr(err) to check if an error represents None.
	NoneErr = errs.NoneErr
)

const (
	ErrCodeGeneric            string = errs.ErrCodeGeneric
	ErrCodeUnknownError       string = errs.ErrCodeUnknownError
	ErrCodeNotPermitted       string = errs.ErrCodeNotPermitted
	ErrCodeIllegalArgument    string = errs.ErrCodeIllegalArgument
	ErrCodeServerShuttingDown string = errs.ErrCodeServerShuttingDown
)

var (
	ErrUnknownError       = errs.ErrUnknownError
	ErrNotPermitted       = errs.ErrNotPermitted
	ErrIllegalArgument    = errs.ErrIllegalArgument
	ErrServerShuttingDown = errs.ErrServerShuttingDown
)

var (
	// Deprecated: use [errs.NewErrf] instead.
	Errf            = errs.NewErrf
	NewErrf         = errs.NewErrf
	IsNoneErr       = errs.IsNoneErr
	NewErrfCode     = errs.NewErrfCode
	ErrfCode        = errs.NewErrfCode
	UnknownErr      = errs.UnknownErr
	WrapErr         = errs.WrapErr
	WrapErrMulti    = errs.WrapErrMulti
	UnknownErrf     = errs.UnknownErrf
	UnknownErrMsgf  = errs.UnknownErrMsgf
	WrapErrf        = errs.WrapErrf
	WrapErrfCode    = errs.WrapErrfCode
	UnwrapErrStack  = errs.UnwrapErrStack
	ErrorStackTrace = errs.ErrorStackTrace
)

type MisoErr = errs.MisoErr
