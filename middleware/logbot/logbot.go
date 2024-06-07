package logbot

import (
	"errors"
	"fmt"

	"github.com/curtisnewbie/miso/middleware/rabbit"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
)

type errorLog struct {
	Node     string
	App      string
	Time     util.ETime
	TraceId  string
	SpanId   string
	FuncName string
	Message  string
}

var (
	reportLogPipeline = rabbit.NewEventPipeline[errorLog]("logbot:error-log:report:pipeline").
		MaxRetry(3)
)

func EnableLogbotErrLogReport() {
	miso.PreServerBootstrap(func(rail miso.Rail) error {
		app := miso.GetPropStr(miso.PropAppName)
		node := fmt.Sprintf("%v-%v", app, miso.GetLocalIPV4())
		ok := miso.SetErrLogHandler(func(el *miso.ErrorLog) {
			reportLogPipeline.Send(rail, errorLog{
				Node:     node,
				App:      app,
				Time:     util.ETime(el.Time),
				TraceId:  el.TraceId,
				SpanId:   el.SpanId,
				FuncName: el.FuncName,
				Message:  el.Message,
			})
		})
		if !ok {
			return errors.New("failed to setup miso ErrorLogHandler, other components may have setup handler already, please resolve conflict before using logbot middleware")
		}
		return nil
	})
}
