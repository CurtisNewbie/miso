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
	var fn string = ""

	if entry.HasCaller() {
		clr := entry.Caller
		fn = " " + getShortFnName(clr.Function)
	}

	var traceId any
	var spanId any
	var username any

	fields := entry.Data
	if fields != nil {
		traceId = fields[X_TRACEID]
		spanId = fields[X_SPANID]
		username = fields[X_USERNAME]
	}
	if traceId == nil {
		traceId = ""
	}
	if spanId == nil {
		spanId = ""
	}
	if username == nil {
		username = ""
	}

	s := fmt.Sprintf("%s %s [%-16v,%-16v,%v]%s - %s\n", entry.Time.Format("2006-01-02 15:04:05.000"), toLevelStr(entry.Level), traceId, spanId, username, fn, entry.Message)
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
	return logrus.WithFields(logrus.Fields{X_SPANID: ctx.Value(X_SPANID), X_TRACEID: ctx.Value(X_TRACEID), X_USERNAME: ctx.Value(X_USERNAME)})
}
