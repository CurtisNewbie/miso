package miso

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
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

func (r Rail) CtxValue(key string) any {
	return r.Ctx.Value(key)
}

func (r Rail) CtxValStr(key string) string {
	if s, ok := GetCtxStr(r.Ctx, key); ok {
		return s
	}
	return ""
}

func (r Rail) CtxValInt(key string) int {
	if d, ok := GetCtxInt(r.Ctx, key); ok {
		return d
	}
	return 0
}

func (r Rail) TraceId() string {
	return r.CtxValStr(X_TRACEID)
}

func (r Rail) SpanId() string {
	return r.CtxValStr(X_SPANID)
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

func (r Rail) IsDebugLogEnabled() bool {
	return r.log.Logger.IsLevelEnabled(logrus.DebugLevel)
}

func (r Rail) IsLogLevelEnabled(level string) bool {
	ll, ok := ParseLogLevel(level)
	if !ok {
		return false
	}
	return r.log.Logger.IsLevelEnabled(ll)
}

func (r Rail) SetLogLevel(level string) {
	ll, ok := ParseLogLevel(level)
	if !ok {
		return
	}
	r.log.Logger.SetLevel(ll)
}

func (r Rail) WithCtxVal(key string, val any) Rail {
	ctx := context.WithValue(r.Ctx, key, val) //lint:ignore SA1029 keys must be exposed for user to use
	return NewRail(ctx)
}

// Create a new Rail with a new SpanId
func (r Rail) NextSpan() Rail {
	// X_TRACE_ID is propagated as parent context, we only need to create a new X_SPAN_ID
	return r.WithCtxVal(X_SPANID, RandLowerAlphaNumeric(16))
}

// Create new Rail with context's CancelFunc
func (r Rail) WithCancel() (Rail, context.CancelFunc) {
	cc, cancel := context.WithCancel(r.Ctx)
	return NewRail(cc), cancel
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
	j := strings.LastIndexAny(fn, "./")
	if j < 0 {
		return fn
	}
	return string([]rune(fn)[j+1:])
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

// Get value from context as a string
//
// int*, unit*, float* types are formatted as string, other types are returned as empty string
func GetCtxStr(ctx context.Context, key string) (string, bool) {
	v := ctx.Value(key)
	if v == nil {
		return "", false
	}
	switch tv := v.(type) {
	case string:
		return tv, true
	case int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", v), true
	default:
		return "", false
	}
}

// Get value from context as an int.
//
// string is also formatted as int if possible.
func GetCtxInt(ctx context.Context, key string) (int, bool) {
	v := ctx.Value(key)
	if v == nil {
		return 0, false
	}
	switch tv := v.(type) {
	case int:
		return tv, true
	case string:
		i, e := strconv.Atoi(tv)
		return i, e == nil
	default:
		return 0, false
	}
}
