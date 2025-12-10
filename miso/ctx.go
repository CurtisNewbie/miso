package miso

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

var (
	_ context.Context = Rail{}
)

var (
	loggerDebugMap = sync.Map{}
)

func ConfigDebugLogToInfo(loggerName string, doRewrite bool) {
	if doRewrite {
		loggerDebugMap.Store(loggerName, struct{}{})
	} else {
		loggerDebugMap.Delete(loggerName)
	}
}

func rewriteDebugLevel(name string) bool {
	if name == "" {
		return false
	}
	_, ok := loggerDebugMap.Load(name)
	return ok
}

// Rail, an object that carries trace infromation along with the execution.
//
// It's essentially a thin wrapper of [context.Context].
type Rail struct {
	name string
	ctx  context.Context
	upN  int
}

func (r Rail) Deadline() (deadline time.Time, ok bool) {
	return r.ctx.Deadline()
}

func (r Rail) Err() error {
	return r.ctx.Err()
}

func (r Rail) Value(key any) any {
	return r.ctx.Value(key)
}

func (r Rail) WithName(name string) Rail {
	r.name = name
	return r
}

func (r Rail) ZeroTrace() Rail {
	return r.WithTraceId("").WithSpanId("")
}

func (r Rail) WithGetCallFnUpN(upN int) Rail {
	r.upN = upN
	if r.upN < 0 {
		r.upN = 0
	}
	return r
}

func (r Rail) SetGetCallFnUpN(upN int) Rail {
	return r.WithGetCallFnUpN(upN)
}

func (r Rail) ErrorIf(err error, op string, args ...any) {
	if err != nil {
		r.Errorf(fmt.Sprintf("%v - %v, %v", GetCallerFnUpN(r.upN), op, err), args...)
	}
}

func (r Rail) WarnIf(err error, op string, args ...any) {
	if err != nil {
		r.Warnf(fmt.Sprintf("%v - %v, %v", GetCallerFnUpN(r.upN), op, err), args...)
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

func (r Rail) WithTraceId(id string) Rail {
	return r.WithCtxVal(XTraceId, id)
}

func (r Rail) Username() string {
	return r.CtxValStr(XUsername)
}

func (r Rail) WithUsername(v string) Rail {
	return r.WithCtxVal(XUsername, v)
}

func (r Rail) SpanId() string {
	return r.CtxValStr(XSpanId)
}

func (r Rail) WithSpanId(id string) Rail {
	return r.WithCtxVal(XSpanId, id)
}

func (r Rail) Tracef(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.TraceLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Tracef(format, args...)
}

func (r Rail) Debugf(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.DebugLevel) {
		if rewriteDebugLevel(r.name) {
			r.WithGetCallFnUpN(1).Infof(format, args...)
		}
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Debugf(format, args...)
}

func (r Rail) Infof(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.InfoLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Infof(format, args...)
}

func appendErrStack(dofmt bool, format string, args ...any) string {
	if dofmt && format != "" && len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
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
	format = appendErrStack(true, format, args...)
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Warn(format)
}

func (r Rail) Errorf(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.ErrorLevel) {
		return
	}
	format = appendErrStack(true, format, args...)
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Error(format)
}

func (r Rail) Fatalf(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Fatalf(format, args...)
}

func (r Rail) Panicf(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Panicf(format, args...)
}

func (r Rail) Printf(format string, args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Printf(format, args...)
}

func (r Rail) Debug(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.DebugLevel) {
		if rewriteDebugLevel(r.name) {
			r.WithGetCallFnUpN(1).Info(args...)
		}
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Debug(args...)
}

func (r Rail) Info(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.InfoLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Info(args...)
}

func (r Rail) Warn(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.WarnLevel) {
		return
	}
	if len(args) == 1 {
		if v, ok := args[0].(*MisoErr); ok && v != nil {
			msgWithStack := appendErrStack(false, v.Error(), v)
			logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
				Warn(msgWithStack)
			return
		}
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Warn(args...)
}

func (r Rail) Error(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.ErrorLevel) {
		return
	}
	if len(args) == 1 {
		if v, ok := args[0].(*MisoErr); ok && v != nil {
			msgWithStack := appendErrStack(false, v.Error(), v)
			logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
				Error(msgWithStack)
			return
		}
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Error(args...)
}

func (r Rail) Fatal(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Fatal(args...)
}

func (r Rail) Panic(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Panic(args...)
}

func (r Rail) Print(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Print(args...)
}

func (r Rail) Debugln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.DebugLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Debugln(args...)
}

func (r Rail) Infoln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.InfoLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Infoln(args...)
}

func (r Rail) Warnln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.WarnLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Warnln(args...)
}

func (r Rail) Errorln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.ErrorLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Errorln(args...)
}

func (r Rail) Fatalln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Fatalln(args...)
}

func (r Rail) Panicln(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Panicln(args...)
}

func (r Rail) Println(args ...interface{}) {
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logger.WithFields(logrus.Fields{XSpanId: r.ctx.Value(XSpanId), XTraceId: r.ctx.Value(XTraceId), callerField: GetCallerFnUpN(r.upN)}).
		Println(args...)
}

func (r Rail) WithCtxVal(key string, val any) Rail {
	ctx := context.WithValue(r.ctx, key, val) //lint:ignore SA1029 keys must be exposed for user to use
	return Rail{ctx: ctx}
}

func (r Rail) NewTrace() Rail {
	tid := NewTraceId()
	return r.NewCtx().WithTraceId(tid).WithSpanId(tid)
}

// Create a new Rail with a new SpanId and a new Context
func (r Rail) NextSpan() Rail {
	return r.NewCtx().WithSpanId(NewSpanId())
}

// Create a new Rail with a new Context
func (r Rail) NewCtx() Rail {
	prev := r.ctx
	r.ctx = context.Background() // avoid using the cancelled context in a new goroutine

	// copy values from previous context
	for _, k := range GetPropagationKeys() {
		r = r.WithCtxVal(k, prev.Value(k))
	}
	return r
}

// Create new Rail with context's CancelFunc
func (r Rail) WithCancel() (Rail, context.CancelFunc) {
	cc, cancel := context.WithCancel(r.ctx)
	return Rail{ctx: cc}, cancel
}

// Create new Rail with timeout and context's CancelFunc
func (r Rail) WithTimeout(timeout time.Duration) (Rail, context.CancelFunc) {
	cc, cancel := context.WithTimeout(r.ctx, timeout)
	return Rail{ctx: cc}, cancel
}

// Create empty Rail.
func EmptyRail() Rail {
	return NewRail(context.Background())
}

// Create new TraceId.
func NewTraceId() string {

	// in latest implementation, it's [16]byte{}
	/*
		t := [16]byte{}
		binary.NativeEndian.PutUint64(t[:8], rand.Uint64())
		binary.NativeEndian.PutUint64(t[8:], rand.Uint64())
		return hex.EncodeToString(t[:])
	*/

	t := [8]byte{} // in latest implementation, it's [16]byte{}
	binary.NativeEndian.PutUint64(t[:], rand.Uint64())
	return hex.EncodeToString(t[:])
}

// Create new SpanId.
func NewSpanId() string {
	s := [8]byte{}
	binary.NativeEndian.PutUint64(s[:], rand.Uint64())
	return hex.EncodeToString(s[:])
}

// Create new Rail from context.
func NewRail(ctx context.Context) Rail {
	var tid string
	if ctx.Value(XTraceId) == nil {
		tid = NewTraceId()
		ctx = context.WithValue(ctx, XTraceId, tid) //lint:ignore SA1029 keys must be exposed for user to use
	}

	if ctx.Value(XSpanId) == nil {
		var sid string
		if tid != "" {
			sid = tid
		} else {
			sid = NewSpanId()
		}
		ctx = context.WithValue(ctx, XSpanId, sid) //lint:ignore SA1029 keys must be exposed for user to use
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
	return cast.ToString(v), true
}

// Get value from context as an int.
//
// string is also formatted as int if possible.
func GetCtxInt(ctx context.Context, key string) (int, bool) {
	v := ctx.Value(key)
	if v == nil {
		return 0, false
	}
	n, err := cast.ToIntE(v)
	return n, err == nil
}
