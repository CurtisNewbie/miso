package miso

import (
	"context"
	"fmt"
	"strconv"

	"github.com/curtisnewbie/miso/util"
	"github.com/sirupsen/logrus"
)

// Rail, an object that carries trace infromation along with the execution.
type Rail struct {
	ctx context.Context
}

func (r Rail) ErrorIf(op string, err error, args ...any) {
	if err != nil {
		r.Errorf(fmt.Sprintf("%v - %v, %v", getCallerFn(), op, err), args...)
	}
}

func (r Rail) WarnIf(op string, err error, args ...any) {
	if err != nil {
		r.Warnf(fmt.Sprintf("%v - %v, %v", getCallerFn(), op, err), args...)
	}
}

func (r Rail) IsDone() bool {
	return r.ctx.Err() != nil
}

func (r Rail) Context() context.Context {
	return r.ctx
}

func (r Rail) Done() <-chan struct{} {
	return r.ctx.Done()
}

func (r Rail) CtxValue(key string) any {
	return r.ctx.Value(key)
}

func (r Rail) CtxValStr(key string) string {
	if s, ok := GetCtxStr(r.ctx, key); ok {
		return s
	}
	return ""
}

func (r Rail) CtxValInt(key string) int {
	if d, ok := GetCtxInt(r.ctx, key); ok {
		return d
	}
	return 0
}

func (r Rail) TraceId() string {
	return r.CtxValStr(XTraceId)
}

func (r Rail) SpanId() string {
	return r.CtxValStr(XSpanId)
}

func (r Rail) Tracef(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.TraceLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Tracef(format, args...)
}

func (r Rail) Debugf(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.DebugLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Debugf(format, args...)
}

func (r Rail) Infof(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.InfoLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Infof(format, args...)
}

func appendErrStack(format string, args ...any) string {
	var err error = nil
	for i := len(args) - 1; i > -1; i-- {
		ar := args[i]
		if er, ok := ar.(error); ok {
			err = er
			break
		}
	}
	if err != nil {
		stackTrace, withStack := UnwrapErrStack(err)
		if withStack {
			format += stackTrace
		}
	}
	return format
}

func (r Rail) Warnf(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.WarnLevel) {
		return
	}
	format = appendErrStack(format, args...)
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Warnf(format, args...)
}

func (r Rail) Errorf(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.ErrorLevel) {
		return
	}
	format = appendErrStack(format, args...)
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Errorf(format, args...)
}

func (r Rail) Fatalf(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Fatalf(format, args...)
}

func (r Rail) Panicf(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Panicf(format, args...)
}

func (r Rail) Printf(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Printf(format, args...)
}

func (r Rail) Debug(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.DebugLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Debug(args...)
}

func (r Rail) Info(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.InfoLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Info(args...)
}

func (r Rail) Warn(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.WarnLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Warn(args...)
}

func (r Rail) Error(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.ErrorLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Error(args...)
}

func (r Rail) Fatal(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Fatal(args...)
}

func (r Rail) Panic(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Panic(args...)
}

func (r Rail) Print(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Print(args...)
}

func (r Rail) Debugln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.DebugLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Debugln(args...)
}

func (r Rail) Infoln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.InfoLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Infoln(args...)
}

func (r Rail) Warnln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.WarnLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Warnln(args...)
}

func (r Rail) Errorln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.ErrorLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Errorln(args...)
}

func (r Rail) Fatalln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Fatalln(args...)
}

func (r Rail) Panicln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Panicln(args...)
}

func (r Rail) Println(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: getCallerFn()}).
		Println(args...)
}

func (r Rail) WithCtxVal(key string, val any) Rail {
	ctx := context.WithValue(r.ctx, key, val) //lint:ignore SA1029 keys must be exposed for user to use
	return NewRail(ctx)
}

// Create a new Rail with a new SpanId
func (r Rail) NextSpan() Rail {
	// X_TRACE_ID is propagated as parent context, we only need to create a new X_SPAN_ID
	return r.WithCtxVal(XSpanId, util.RandLowerAlphaNumeric16())
}

// Create new Rail with context's CancelFunc
func (r Rail) WithCancel() (Rail, context.CancelFunc) {
	cc, cancel := context.WithCancel(r.ctx)
	return NewRail(cc), cancel
}

// Create empty Rail.
func EmptyRail() Rail {
	return NewRail(context.Background())
}

// Create new Rail from context.
func NewRail(ctx context.Context) Rail {
	if ctx.Value(XSpanId) == nil {
		ctx = context.WithValue(ctx, XSpanId, util.RandLowerAlphaNumeric16()) //lint:ignore SA1029 keys must be exposed for user to use
	}
	if ctx.Value(XTraceId) == nil {
		ctx = context.WithValue(ctx, XTraceId, util.RandLowerAlphaNumeric16()) //lint:ignore SA1029 keys must be exposed for user to use
	}
	return Rail{ctx: ctx}
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
