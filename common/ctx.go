package common

import (
	"context"
	"runtime"

	"github.com/sirupsen/logrus"
)

var (
	nilUser = User{}
)

// Prepared execution context, namly the rail
type Rail struct {
	Ctx context.Context // request context
	log *logrus.Entry   // logger with tracing info
}

func (r Rail) Debugf(format string, args ...interface{}) {
	r.log.Caller = getCaller()
	r.log.Debugf(format, args)
}

func (r Rail) Infof(format string, args ...interface{}) {
	r.log.Caller = getCaller()
	r.log.Infof(format, args)
}

func (r Rail) Warnf(format string, args ...interface{}) {
	r.log.Caller = getCaller()
	r.log.Warnf(format, args)
}

func (r Rail) Errorf(format string, args ...interface{}) {
	r.log.Caller = getCaller()
	r.log.Errorf(format, args)
}

func (r Rail) Fatalf(format string, args ...interface{}) {
	r.log.Caller = getCaller()
	r.log.Fatalf(format, args)
}

func (r Rail) Debug(msg string) {
	r.log.Caller = getCaller()
	r.log.Debug(msg)
}

func (r Rail) Info(msg string) {
	r.log.Caller = getCaller()
	r.log.Info(msg)
}

func (r Rail) Warn(msg string) {
	r.log.Caller = getCaller()
	r.log.Warn(msg)
}

func (r Rail) Error(msg string) {
	r.log.Caller = getCaller()
	r.log.Error(msg)
}

func (r Rail) Fatal(msg string) {
	r.log.Caller = getCaller()
	r.log.Fatal(msg)
}

// Create a new ExecContext with a new SpanId
func (c *Rail) NextSpan() Rail {
	// X_TRACE_ID is propagated as parent context, we only need to create a new X_SPAN_ID
	ctx := context.WithValue(c.Ctx, X_SPANID, RandLowerAlphaNumeric(16)) //lint:ignore SA1029 keys must be exposed for user to use
	return NewExecContext(ctx)
}

func getCaller() *runtime.Frame {
	pcs := make([]uintptr, 2)
	depth := runtime.Callers(1, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	i := 0
	for f, next := frames.Next(); next && i < 2; {
		return &f //nolint:scopelint
	}
	return nil
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
