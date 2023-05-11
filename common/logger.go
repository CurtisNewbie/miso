package common

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

type CTFormatter struct {
}

func init() {
	// for connvenience
	logrus.SetReportCaller(true)
	logrus.SetFormatter(CustomFormatter())
}

func (c *CTFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var fn string = ""

	if entry.HasCaller() {
		clr := entry.Caller
		fn = " " + getShortFnName(clr.Function)
	}

	var traceId any
	var spanId any

	fields := entry.Data
	if fields != nil {
		traceId = fields[X_TRACEID]
		spanId = fields[X_SPANID]
	}
	if traceId == nil {
		traceId = ""
	}
	if spanId == nil {
		spanId = ""
	}

	s := fmt.Sprintf("%s %-5s [%-16v,%-16v]%-35s : %s\n", entry.Time.Format("2006-01-02 15:04:05.000"), toLevelStr(entry.Level), traceId, spanId, fn, entry.Message)
	return []byte(s), nil
}

// Create rolling file based logger
func BuildRollingLogFileWriter(logFile string) io.Writer {
	return &lumberjack.Logger{
		Filename:  logFile,
		MaxSize:   100, // megabytes
		MaxAge:    15,  //days
		LocalTime: true,
		Compress:  false,
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

func getShortFnName(fn string) string {
	j := strings.LastIndex(fn, "/")
	if j < 0 {
		return fn
	}
	return GetRuneWrp(fn).SubstrFrom(j + 1)
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
	return logrus.WithFields(logrus.Fields{X_SPANID: ctx.Value(X_SPANID), X_TRACEID: ctx.Value(X_TRACEID), X_USERNAME: ctx.Value(X_USERNAME)})
}
