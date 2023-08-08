package common

import (
	"context"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// Prepared execution context, namly the rail
type Rail struct {
	Ctx context.Context // request context
	log *logrus.Entry   // logger with tracing info
}

func (r Rail) Debugf(format string, args ...interface{}) {
	r.log.WithField(callerField, getCallerFn()).Debugf(format, args...)
}

func (r Rail) Infof(format string, args ...interface{}) {
	r.log.WithField(callerField, getCallerFn()).Infof(format, args...)
}

func (r Rail) Warnf(format string, args ...interface{}) {
	r.log.WithField(callerField, getCallerFn()).Warnf(format, args...)
}

func (r Rail) Errorf(format string, args ...interface{}) {
	r.log.WithField(callerField, getCallerFn()).Errorf(format, args...)
}

func (r Rail) Fatalf(format string, args ...interface{}) {
	r.log.WithField(callerField, getCallerFn()).Fatalf(format, args...)
}

func (r Rail) Debug(msg string) {
	r.log.WithField(callerField, getCallerFn()).Debug(msg)
}

func (r Rail) Info(msg string) {
	r.log.WithField(callerField, getCallerFn()).Info(msg)
}

func (r Rail) Warn(msg string) {
	r.log.WithField(callerField, getCallerFn()).Warn(msg)
}

func (r Rail) Error(msg string) {
	r.log.WithField(callerField, getCallerFn()).Error(msg)
}

func (r Rail) Fatal(msg string) {
	r.log.WithField(callerField, getCallerFn()).Fatal(msg)
}

// Create a new ExecContext with a new SpanId
func (c *Rail) NextSpan() Rail {
	// X_TRACE_ID is propagated as parent context, we only need to create a new X_SPAN_ID
	ctx := context.WithValue(c.Ctx, X_SPANID, RandLowerAlphaNumeric(16)) //lint:ignore SA1029 keys must be exposed for user to use
	return NewExecContext(ctx)
}

func getCaller(level int) *runtime.Frame {
	pcs := make([]uintptr, 25) // logrus also use 25 :D
	depth := runtime.Callers(level, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, next := frames.Next(); next; {
		return &f //nolint:scopelint
	}
	return nil
}

func getCallerFn() string {
	clr := getCaller(4)
	if clr == nil {
		return ""
	}
	return getShortFnName(clr.Function)
}

func getShortFnName(fn string) string {
	j := strings.LastIndex(fn, "/")
	if j < 0 {
		return fn
	}
	return GetRuneWrp(fn).SubstrFrom(j + 1)
}

// Create empty ExecContext
func EmptyExecContext() Rail {
	ctx := context.Background()

	if ctx.Value(X_SPANID) == nil {
		ctx = context.WithValue(ctx, X_SPANID, RandLowerAlphaNumeric(16)) //lint:ignore SA1029 keys must be exposed for user to use
	}

	if ctx.Value(X_TRACEID) == nil {
		ctx = context.WithValue(ctx, X_TRACEID, RandLowerAlphaNumeric(16)) //lint:ignore SA1029 keys must be exposed for user to use
	}

	return NewExecContext(ctx)
}

// Create new ExecContext
func NewExecContext(ctx context.Context) Rail {
	if ctx.Value(X_SPANID) == nil {
		ctx = context.WithValue(ctx, X_SPANID, RandLowerAlphaNumeric(16)) //lint:ignore SA1029 keys must be exposed for user to use
	}

	if ctx.Value(X_TRACEID) == nil {
		ctx = context.WithValue(ctx, X_TRACEID, RandLowerAlphaNumeric(16)) //lint:ignore SA1029 keys must be exposed for user to use
	}

	return Rail{Ctx: ctx, log: TraceLogger(ctx)}
}

func GetCtxStr(ctx context.Context, key string) string {
	v := ctx.Value(key)
	if v == nil {
		return ""
	}
	sv, ok := v.(string)
	if ok {
		return sv
	}
	return ""
}

func SelectExecContext(cs ...Rail) Rail {
	if len(cs) > 0 {
		return cs[0]
	} else {
		return EmptyExecContext()
	}
}
