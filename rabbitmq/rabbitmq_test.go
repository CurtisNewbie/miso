package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/sirupsen/logrus"
)

func TestInitConnection(t *testing.T) {
	common.LoadConfigFromFile("../app-conf-dev.json")
	common.SetProp(common.PROP_RABBITMQ_USERNAME, "guest")
	common.SetProp(common.PROP_RABBITMQ_PASSWORD, "guest")

	AddListener(MsgListener{QueueName: "my-first-queue", Handler: func(payload []byte, contentType, messageId string) error {
		logrus.Infof("Received message %s, content-type: %s, messageId: %s", string(payload), contentType, messageId)	
		return nil
	}})

	ctx, cancel := context.WithCancel(context.Background())
	_, e := initClient(ctx)
	if e != nil {
		t.Error(e)
	}

	// make sure that the consumer is created before we cancel the context
	time.Sleep(time.Second * 3) 
	cancel()
	logrus.Info("Cancelling backgroun context")
	time.Sleep(time.Second * 3)
}
