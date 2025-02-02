package logbot

import (
	"errors"
	"fmt"
	"strings"

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

// Deprecated: this is a very bad practice.
// The error logs may not be sent to the pipeline when the app fails to
// connect to rabbitmq broker, but the broker can be running just fine.
//
// Correct solution should be an agent tailing and parsing the logs produced by the app (i.e., logbot itself).
func EnableLogbotErrLogReport() {
	miso.PreServerBootstrap(func(rail miso.Rail) error {
		app := miso.GetPropStr(miso.PropAppName)
		node := fmt.Sprintf("%v-%v", app, util.GetLocalIPV4())

		ok := miso.SetErrLogHandler(func(el *miso.ErrorLog) {

			if strings.HasPrefix(el.FuncName, "rabbit.") || strings.HasPrefix(el.FuncName, "logbot.") {
				// exclude error logs from middleware/rabbit
				return
			}
			sendErrLog(rail, node, app, el)
		})
		if !ok {
			return errors.New("failed to setup miso ErrorLogHandler, other components may have setup handler already, please resolve conflict before using logbot middleware")
		}
		return nil
	})
}

func sendErrLog(rail miso.Rail, node string, app string, el *miso.ErrorLog) {
	err := reportLogPipeline.Send(rail, errorLog{
		Node:     node,
		App:      app,
		Time:     util.ToETime(el.Time),
		TraceId:  el.TraceId,
		SpanId:   el.SpanId,
		FuncName: el.FuncName,
		Message:  el.Message,
	})
	if err != nil {
		rail.Errorf("logbot reportLogPipeline.Send failed, %v", err)
	}
}
