package miso

import (
	"github.com/curtisnewbie/miso/flow"
	"github.com/curtisnewbie/miso/util/src"
)

const (
	XTraceId  = flow.XTraceId
	XSpanId   = flow.XSpanId
	XUsername = flow.XUsername
)

type (
	Rail                   = flow.Rail
	CTFormatter            = flow.CTFormatter
	NewRollingLogFileParam = flow.NewRollingLogFileParam
	PlainStrFormatter      = flow.PlainStrFormatter
	PropagationKeys        = flow.PropagationKeys
)

var (
	GetCallerFn    = src.GetCallerFn
	GetCallerFnUpN = src.GetCallerFnUpN

	ConfigDebugLogToInfo      = flow.ConfigDebugLogToInfo
	EmptyRail                 = flow.EmptyRail
	NewTraceId                = flow.NewTraceId
	NewSpanId                 = flow.NewSpanId
	NewRail                   = flow.NewRail
	GetCtxStr                 = flow.GetCtxStr
	GetCtxInt                 = flow.GetCtxInt
	BuildRollingLogFileWriter = flow.BuildRollingLogFileWriter
	CustomFormatter           = flow.CustomFormatter
	PreConfiguredFormatter    = flow.PreConfiguredFormatter
	TraceLogger               = flow.TraceLogger
	IsDebugLevel              = flow.IsDebugLevel
	IsTraceLevel              = flow.IsTraceLevel
	IsLogLevel                = flow.IsLogLevel
	ParseLogLevel             = flow.ParseLogLevel
	SetLogLevel               = flow.SetLogLevel
	SetLogOutput              = flow.SetLogOutput
	GetLogrusLogger           = flow.GetLogrusLogger
	Infof                     = flow.Infof
	Tracef                    = flow.Tracef
	Debugf                    = flow.Debugf
	Warnf                     = flow.Warnf
	Errorf                    = flow.Errorf
	Fatalf                    = flow.Fatalf
	Debug                     = flow.Debug
	Info                      = flow.Info
	Warn                      = flow.Warn
	Error                     = flow.Error
	Fatal                     = flow.Fatal
	AddPropagationKeys        = flow.AddPropagationKeys
	AddPropagationKey         = flow.AddPropagationKey
	GetPropagationKeys        = flow.GetPropagationKeys
	UsePropagationKeys        = flow.UsePropagationKeys
	BuildTraceHeadersAny      = flow.BuildTraceHeadersAny
	BuildTraceHeadersStr      = flow.BuildTraceHeadersStr
)
