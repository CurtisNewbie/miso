package miso

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/util/pool"
	"github.com/curtisnewbie/miso/util/src"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

const (
	callerField = "caller"
)

const (
	traceSpanIdWidth = 16
	fnWidth          = 30
	levelWidth       = 5
)

var (
	GetCallerFn    = src.GetCallerFn
	GetCallerFnUpN = src.GetCallerFnUpN
)

var (
	zRail  = EmptyRail().ZeroTrace()
	Infof  = zRail.Infof
	Tracef = zRail.Tracef
	Debugf = zRail.Debugf
	Warnf  = zRail.Warnf
	Errorf = zRail.Errorf
	Fatalf = zRail.Fatalf
	Debug  = zRail.Debug
	Info   = zRail.Info
	Warn   = zRail.Warn
	Error  = zRail.Error
	Fatal  = zRail.Fatal
)

var (
	logBufPool = pool.NewByteBufferPool(128)
)

var (
	errLogHandlerOnce sync.Once
	errLogPipe        chan *ErrorLog = nil
	errLogPool                       = sync.Pool{
		New: func() any { return new(ErrorLog) },
	}
	errLogRoutineCancel func()         = nil
	logger              *logrus.Logger = logrus.New()
)

func init() {
	logger.SetReportCaller(false) // caller is handled by CTFormatter
	logger.SetFormatter(CustomFormatter())
}

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

	b := logBufPool.Get()
	defer logBufPool.Put(b)

	b.WriteString(entry.Time.Format("2006-01-02 15:04:05.000"))
	b.WriteByte(' ')
	b.WriteString(levelstr)

	if len(levelstr) < levelWidth {
		b.WriteString(strutil.Spaces(levelWidth - len(levelstr)))
	}

	b.WriteString(" [")
	b.WriteString(traceId)

	if len(traceId) < traceSpanIdWidth {
		b.WriteString(strutil.Spaces(traceSpanIdWidth - len(traceId)))
	}

	b.WriteByte(',')
	b.WriteString(spanId)

	if len(spanId) < traceSpanIdWidth {
		b.WriteString(strutil.Spaces(traceSpanIdWidth - len(spanId)))
	}

	b.WriteString("]  ")
	b.WriteString(fn)

	if len(fn) < fnWidth {
		b.WriteString(strutil.Spaces(fnWidth - len(fn)))
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
	return logger.WithFields(logrus.Fields{XSpanId: ctx.Value(XSpanId), XTraceId: ctx.Value(XTraceId)})
}

// Check whether current log level is DEBUG
func IsDebugLevel() bool {
	return logger.GetLevel() == logrus.DebugLevel
}

func IsTraceLevel() bool {
	return logger.GetLevel() == logrus.TraceLevel
}

func IsLogLevel(logLevel string) bool {
	v, ok := ParseLogLevel(logLevel)
	if !ok {
		return false
	}
	return logger.GetLevel() == v
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

func SetLogLevel(level string) {
	ll, ok := ParseLogLevel(level)
	if !ok {
		return
	}
	logger.SetLevel(ll)
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

func SetLogOutput(out io.Writer) {
	logger.SetOutput(out)
}

func GetLogrusLogger() *logrus.Logger {
	return logger
}

type PlainStrFormatter struct {
}

func (p PlainStrFormatter) Format(e *logrus.Entry) ([]byte, error) {
	return []byte(e.Message), nil
}
