package dbquery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/curtisnewbie/miso/miso"
	lg "gorm.io/gorm/logger"
)

func NewGormLogger(config lg.Config) *gormLogger {
	l := &gormLogger{}
	l.UpdateConfig(config)
	return l
}

type gormLogger struct {
	lg.Config
	traceStr, traceErrStr, traceWarnStr string
}

func (l *gormLogger) UpdateConfig(config lg.Config) {
	var (
		traceStr     = "[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s [%.3fms] [rows:%v] %s"
		traceErrStr  = "[%.3fms] [rows:%v] %s\n\t%v"
	)

	if config.Colorful {
		traceStr = lg.Yellow + "[%.3fms] " + lg.BlueBold + "[rows:%v]" + lg.Reset + " %s" + lg.Reset
		traceWarnStr = lg.Yellow + "%s " + lg.RedBold + "[%.3fms] " + lg.Yellow + "[rows:%v]" + lg.Magenta + " %s" + lg.Reset
		traceErrStr = lg.Yellow + "[%.3fms] " + lg.BlueBold + "[rows:%v]" + lg.Reset + " %s\n\t" + lg.RedBold + "%s" + lg.Reset
	}
	l.Config = config
	l.traceStr = traceStr
	l.traceWarnStr = traceWarnStr
	l.traceErrStr = traceErrStr
}

// LogMode log mode
func (l *gormLogger) LogMode(level lg.LogLevel) lg.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info print info
func (l gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= lg.Info {
		miso.NewRail(ctx).Infof(msg, data...)
	}
}

// Warn print warn messages
func (l gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= lg.Warn {
		miso.NewRail(ctx).Warnf(msg, data...)
	}
}

// Error print error messages
func (l gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= lg.Error {
		miso.NewRail(ctx).Errorf(msg, data...)
	}
}

// Trace print sql message
func (l gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	ll := l.LogLevel
	forceLog := ctx.Value(contextKeyLogSQL) != nil
	if forceLog {
		ll = lg.Info
	}
	if ll <= lg.Silent {
		return
	}
	if !forceLog && ctx.Value(contextKeyNotLogSQL) != nil {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && ll >= lg.Error && (!errors.Is(err, lg.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			miso.NewRail(ctx).Errorf(l.traceErrStr, float64(elapsed.Nanoseconds())/1e6, "-", sql, err)
		} else {
			miso.NewRail(ctx).Errorf(l.traceErrStr, float64(elapsed.Nanoseconds())/1e6, rows, sql, err)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && ll >= lg.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			miso.NewRail(ctx).Warnf(l.traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			miso.NewRail(ctx).Warnf(l.traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case ll == lg.Info:
		sql, rows := fc()
		if rows == -1 {
			miso.NewRail(ctx).Infof(l.traceStr, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			miso.NewRail(ctx).Infof(l.traceStr, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}
