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

func (r Rail) Logger() *logrus.Entry {
	return r.log
}

func (r Rail) CtxValue(key string) string {
	v := r.Ctx.Value(key)
	if vs, ok := v.(string); ok {
		return vs
	}
	return ""
}

func (r Rail) TraceId() string {
	return r.CtxValue(X_TRACEID)
}

func (r Rail) SpanId() string {
	return r.CtxValue(X_SPANID)
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

func (r Rail) Debug(args ...interface{}) {
	r.log.WithField(callerField, getCallerFn()).Debug(args...)
}

func (r Rail) Info(args ...interface{}) {
	r.log.WithField(callerField, getCallerFn()).Info(args...)
}

func (r Rail) Warn(args ...interface{}) {
	r.log.WithField(callerField, getCallerFn()).Warn(args...)
}

func (r Rail) Error(args ...interface{}) {
	r.log.WithField(callerField, getCallerFn()).Error(args...)
}

func (r Rail) Fatal(args ...interface{}) {
	r.log.WithField(callerField, getCallerFn()).Fatal(args...)
}

func (r Rail) WithCtxVal(key string, val string) Rail {
	ctx := context.WithValue(r.Ctx, key, val) //lint:ignore SA1029 keys must be exposed for user to use
	return NewRail(ctx)
}

// Create a new Rail with a new SpanId
func (r Rail) NextSpan() Rail {
	// X_TRACE_ID is propagated as parent context, we only need to create a new X_SPAN_ID
	return r.WithCtxVal(X_SPANID, RandLowerAlphaNumeric(16))
}

func getCaller(level int) *runtime.Frame {
	pcs := make([]uintptr, level+1) // we only need the first frame
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

	return RString(fn).SubstrStart(j + 1)
}

// Create empty Rail
func EmptyRail() Rail {
	return NewRail(context.Background())
}

// Create new Rail from context
func NewRail(ctx context.Context) Rail {
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

func AnyRail(cs ...Rail) Rail {
	if len(cs) > 0 {
		return cs[0]
	} else {
		return EmptyRail()
	}
}
