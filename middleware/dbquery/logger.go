package dbquery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/curtisnewbie/miso/miso"
	lg "gorm.io/gorm/logger"
)

func NewGormLogger(config lg.Config) lg.Interface {
	var (
		infoStr      = "[info] "
		warnStr      = "[warn] "
		errStr       = "[error] "
		traceStr     = "[%.3fms] [rows:%v] %s"
		traceWarnStr = "[%.3fms] [rows:%v] %s"
		traceErrStr  = "[%.3fms] [rows:%v] %s"
	)

	if config.Colorful {
		infoStr = lg.Green + "[info] " + lg.Reset
		warnStr = lg.Magenta + "[warn] " + lg.Reset
		errStr = lg.Red + "[error] " + lg.Reset
		traceStr = lg.Yellow + "[%.3fms] " + lg.BlueBold + "[rows:%v]" + lg.Reset + " %s"
		traceWarnStr = lg.RedBold + "[%.3fms] " + lg.Yellow + "[rows:%v]" + lg.Magenta + " %s" + lg.Reset
		traceErrStr = lg.Yellow + "[%.3fms] " + lg.BlueBold + "[rows:%v]" + lg.Reset + " %s" + lg.Reset
	}

	return &gormLogger{
		Config:       config,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
	}
}

type gormLogger struct {
	lg.Config
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
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
		miso.NewRail(ctx).Infof(l.infoStr+msg, data...)
	}
}

// Warn print warn messages
func (l gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= lg.Warn {
		miso.NewRail(ctx).Warnf(l.infoStr+msg, data...)
	}
}

// Error print error messages
func (l gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= lg.Error {
		miso.NewRail(ctx).Errorf(l.infoStr+msg, data...)
	}
}

// Trace print sql message
func (l gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= lg.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= lg.Error && (!errors.Is(err, lg.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			miso.NewRail(ctx).Infof(l.traceErrStr, err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			miso.NewRail(ctx).Infof(l.traceErrStr, err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= lg.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			miso.NewRail(ctx).Infof(l.traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			miso.NewRail(ctx).Infof(l.traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.LogLevel == lg.Info:
		sql, rows := fc()
		if rows == -1 {
			miso.NewRail(ctx).Infof(l.traceStr, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			miso.NewRail(ctx).Infof(l.traceStr, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}
