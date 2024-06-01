package miso

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

const (
	callerField = "caller"
)

func init() {
	logrus.SetReportCaller(false) // it's now set manually using Rail

	// for convenience
	logrus.SetFormatter(CustomFormatter())
}

const (
	traceSpanIdWidth = 16
	fnWidth          = 30
	levelWidth       = 5
)

var (
	logBufPool = sync.Pool{
		New: func() any {
			return &bytes.Buffer{}
		},
	}
)

var (
	errLogHandlerOnce sync.Once
	errLogPipe        chan *ErrorLog = nil
	errLogPool                       = sync.Pool{
		New: func() any { return new(ErrorLog) },
	}
	errLogRoutineCancel func() = nil
)

type ErrorLog struct {
	Time     time.Time
	TraceId  string
	SpanId   string
	FuncName string
	Message  string
}

type ErrorLogHandler func(el *ErrorLog)

type CTFormatter struct {
}

func (c *CTFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var fn string
	caller, ok := entry.Data[callerField]
	if ok {
		fn = caller.(string)
	}

	var traceId string
	var spanId string
	fields := entry.Data
	if fields != nil {
		if v, ok := fields[XTraceId].(string); ok {
			traceId = v
		}
		if v, ok := fields[XSpanId].(string); ok {
			spanId = v
		}
	}

	levelstr := toLevelStr(entry.Level)

	b := logBufPool.Get().(*bytes.Buffer)
	defer putLogBuf(b)

	b.WriteString(entry.Time.Format("2006-01-02 15:04:05.000"))
	b.WriteByte(' ')
	b.WriteString(levelstr)

	if len(levelstr) < levelWidth {
		b.WriteString(Spaces(levelWidth - len(levelstr)))
	}

	b.WriteString(" [")
	b.WriteString(traceId)

	if len(traceId) < traceSpanIdWidth {
		b.WriteString(Spaces(traceSpanIdWidth - len(traceId)))
	}

	b.WriteByte(',')
	b.WriteString(spanId)

	if len(spanId) < traceSpanIdWidth {
		b.WriteString(Spaces(traceSpanIdWidth - len(spanId)))
	}

	b.WriteString("]  ")
	b.WriteString(fn)

	if len(fn) < fnWidth {
		b.WriteString(Spaces(fnWidth - len(fn)))
	}

	b.WriteString(" : ")
	b.WriteString(entry.Message)
	b.WriteByte('\n')

	if entry.Level == logrus.ErrorLevel && errLogPipe != nil {
		el := errLogPool.Get().(*ErrorLog)
		el.Time = entry.Time
		el.FuncName = fn
		el.Message = entry.Message
		el.SpanId = spanId
		el.TraceId = traceId

		select {
		case errLogPipe <- el:
		default: // just in case the pipe is blocked
		}
	}

	return b.Bytes(), nil
}

func putLogBuf(b *bytes.Buffer) {
	b.Reset()
	logBufPool.Put(b)
}

type NewRollingLogFileParam struct {
	Filename   string // filename
	MaxSize    int    // max file size in mb
	MaxAge     int    // max age in day
	MaxBackups int    // max number of files
}

// Create rolling file based logger
func BuildRollingLogFileWriter(p NewRollingLogFileParam) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   p.Filename,
		MaxSize:    p.MaxSize,    // megabytes
		MaxAge:     p.MaxAge,     // days
		MaxBackups: p.MaxBackups, // num of files
		LocalTime:  true,
		Compress:   false,
	}
}

func toLevelStr(level logrus.Level) string {
	switch level {
	case logrus.TraceLevel:
		return "TRACE"
	case logrus.DebugLevel:
		return "DEBUG"
	case logrus.InfoLevel:
		return "INFO"
	case logrus.WarnLevel:
		return "WARN"
	case logrus.ErrorLevel:
		return "ERROR"
	case logrus.FatalLevel:
		return "FATAL"
	case logrus.PanicLevel:
		return "PANIC"
	}
	return "UNKNOWN"
}

// Get custom formatter logrus
func CustomFormatter() logrus.Formatter {
	return &CTFormatter{}
}

// Get pre-configured TextFormatter for logrus
func PreConfiguredFormatter() logrus.Formatter {
	return &logrus.TextFormatter{
		FullTimestamp: true,
	}
}

// Return logger with tracing infomation
func TraceLogger(ctx context.Context) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{XSpanId: ctx.Value(XSpanId), XTraceId: ctx.Value(XTraceId)})
}

// Check whether current log level is DEBUG
func IsDebugLevel() bool {
	return logrus.GetLevel() == logrus.DebugLevel
}

// Parse log level
func ParseLogLevel(logLevel string) (logrus.Level, bool) {
	logLevel = strings.ToUpper(logLevel)
	switch logLevel {
	case "INFO":
		return logrus.InfoLevel, true
	case "DEBUG":
		return logrus.DebugLevel, true
	case "WARN":
		return logrus.WarnLevel, true
	case "ERROR":
		return logrus.ErrorLevel, true
	case "TRACE":
		return logrus.TraceLevel, true
	case "FATAL":
		return logrus.FatalLevel, true
	case "PANIC":
		return logrus.PanicLevel, true
	}
	return logrus.InfoLevel, false
}

func Tracef(format string, args ...interface{}) {
	logrus.WithField(callerField, getCallerFn()).Tracef(format, args...)
}

func Debugf(format string, args ...interface{}) {
	if !logrus.IsLevelEnabled(logrus.DebugLevel) {
		return
	}
	logrus.WithField(callerField, getCallerFn()).Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	if !logrus.IsLevelEnabled(logrus.InfoLevel) {
		return
	}
	logrus.WithField(callerField, getCallerFn()).Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	if !logrus.IsLevelEnabled(logrus.WarnLevel) {
		return
	}
	logrus.WithField(callerField, getCallerFn()).Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	if !logrus.IsLevelEnabled(logrus.ErrorLevel) {
		return
	}
	logrus.WithField(callerField, getCallerFn()).Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	if !logrus.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logrus.WithField(callerField, getCallerFn()).Fatalf(format, args...)
}

func Debug(args ...interface{}) {
	if !logrus.IsLevelEnabled(logrus.DebugLevel) {
		return
	}
	logrus.WithField(callerField, getCallerFn()).Debug(args...)
}

func Info(args ...interface{}) {
	if !logrus.IsLevelEnabled(logrus.InfoLevel) {
		return
	}
	logrus.WithField(callerField, getCallerFn()).Info(args...)
}

func Warn(args ...interface{}) {
	if !logrus.IsLevelEnabled(logrus.WarnLevel) {
		return
	}
	logrus.WithField(callerField, getCallerFn()).Warn(args...)
}

func Error(args ...interface{}) {
	if !logrus.IsLevelEnabled(logrus.ErrorLevel) {
		return
	}
	logrus.WithField(callerField, getCallerFn()).Error(args...)
}

func Fatal(args ...interface{}) {
	if !logrus.IsLevelEnabled(logrus.FatalLevel) {
		return
	}
	logrus.WithField(callerField, getCallerFn()).Fatal(args...)
}

func SetLogLevel(level string) {
	ll, ok := ParseLogLevel(level)
	if !ok {
		return
	}
	logrus.SetLevel(ll)
}

// reduce alloc, logger calls getCallerFn very frequently, we have to optimize it as much as possible.
var callerUintptrPool = sync.Pool{
	New: func() any {
		p := make([]uintptr, 4)
		return &p
	},
}

func getCallerFn() string {
	pcs := callerUintptrPool.Get().(*[]uintptr)
	defer putCallerUintptrPool(pcs)

	depth := runtime.Callers(3, *pcs)
	frames := runtime.CallersFrames((*pcs)[:depth])

	// we only need the first frame
	for f, next := frames.Next(); next; {
		return unsafeGetShortFnName(f.Function)
	}
	return ""
}

func putCallerUintptrPool(pcs *[]uintptr) {
	for i := range *pcs {
		(*pcs)[i] = 0 // zero the values, just in case
	}
	callerUintptrPool.Put(pcs)
}

// func getCaller(level int) *runtime.Frame {
// 	pcs := make([]uintptr, level+1) // we only need the first frame
// 	depth := runtime.Callers(level, pcs)
// 	frames := runtime.CallersFrames(pcs[:depth])

// 	for f, next := frames.Next(); next; {
// 		return &f //nolint:scopelint
// 	}
// 	return nil
// }

func getShortFnName(fn string) string {
	j := strings.LastIndex(fn, "/")
	if j < 0 {
		return fn
	}
	return string([]rune(fn)[j+1:])
}

func unsafeGetShortFnName(fn string) string {
	j := strings.LastIndexByte(fn, '/')
	if j < 0 {
		return fn
	}
	return UnsafeByt2Str(UnsafeStr2Byt(fn)[j+1:])
}

func Printlnf(pat string, args ...any) {
	fmt.Printf(pat+"\n", args...)
}

// Setup error log handler.
//
// ErrorLogHnadler is invoked when ERROR level log is printed, the log messages passed to handler
// are buffered, but handler should never block for a long time (i.e., process as fast as possible).
// If the buffer is full, latest error log messages are simply dropped.
//
// ErrorLogHandler can only be set once.
func SetErrLogHandler(handler ErrorLogHandler) bool {
	set := false
	errLogHandlerOnce.Do(func() {
		errLogPipe = make(chan *ErrorLog, 1024)
		c, cancel := context.WithCancel(context.Background())
		errLogRoutineCancel = cancel
		AddShutdownHook(errLogRoutineCancel)

		go func() {
			for {
				select {
				case <-c.Done():
					return
				case el := <-errLogPipe:
					handler(el)
					errLogPool.Put(el)
				}
			}
		}()
		set = true
	})
	return set
}
