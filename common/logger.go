package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetReportCaller(true)
	logrus.SetFormatter(CustomFormatter())
}

type CTFormatter struct {
}

func (c *CTFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	clr := entry.Caller
	var fn string = ""

	if entry.HasCaller() {
		fn = " " + getShortFnName(clr.Function)
	}

	var traceId any
	var spanId any

	fields := entry.Data
	if fields != nil {
		traceId = fields[X_B3_TRACEID]
		spanId = fields[X_B3_SPANID]
	}
	if traceId == nil {
		traceId = ""
	}
	if spanId == nil {
		spanId = ""
	}

	s := fmt.Sprintf("[%s] [%v,%v] %s%s - %s\n", toLevelStr(entry.Level), traceId, spanId, entry.Time.Format("2006-01-02 15:04:05"), fn, entry.Message)
	return []byte(s), nil
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

func getShortFnName(fn string) string {
	i := strings.LastIndex(fn, ".")
	if i < 0 {
		return fn
	}
	rw := GetRuneWrp(fn)
	return rw.SubstrFrom(i + 1)
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
func WithTrace(ctx context.Context) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{X_B3_SPANID: ctx.Value(X_B3_SPANID), X_B3_TRACEID: ctx.Value(X_B3_TRACEID)})
}

// logrus.Fatal with tracing information 
func Fatal(ctx context.Context, args ...interface{}) {
	WithTrace(ctx).Fatal(args...)
}

// logrus.Fatalf with tracing information 
func Fatalf(ctx context.Context, fmtstr string, args ...interface{}) {
	WithTrace(ctx).Fatalf(fmtstr, args...)
}

// logrus.Error with tracing information 
func Error(ctx context.Context, args ...interface{}) {
	WithTrace(ctx).Error(args...)
}

// logrus.Errorf with tracing information 
func Errorf(ctx context.Context, fmtstr string, args ...interface{}) {
	WithTrace(ctx).Errorf(fmtstr, args...)
}

// logrus.Warn with tracing information 
func Warn(ctx context.Context, args ...interface{}) {
	WithTrace(ctx).Warn(args...)
}

// logrus.Warnf with tracing information 
func Warnf(ctx context.Context, fmtstr string, args ...interface{}) {
	WithTrace(ctx).Warnf(fmtstr, args...)
}

// logrus.Info with tracing information 
func Info(ctx context.Context, args ...interface{}) {
	WithTrace(ctx).Info(args...)
}

// logrus.Infof with tracing information 
func Infof(ctx context.Context, fmtstr string, args ...interface{}) {
	WithTrace(ctx).Infof(fmtstr, args...)
}
