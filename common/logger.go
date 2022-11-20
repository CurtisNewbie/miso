package common

import (
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
	s := fmt.Sprintf("[%s] %s%s - %s\n", toLevelStr(entry.Level), entry.Time.Format("2006-01-02 15:04:05"), fn, entry.Message)
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

	return "UNOWN"
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
