package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/sirupsen/logrus"
)

func TestInitClient(t *testing.T) {
	common.LoadConfigFromFile("../app-conf-dev.yml")
	common.SetProp(common.PROP_RABBITMQ_USERNAME, "guest")
	common.SetProp(common.PROP_RABBITMQ_PASSWORD, "guest")

	AddListener(MsgListener{QueueName: "my-first-queue", Handler: func(payload string) error {
		logrus.Infof("Received message %s", payload)
		// return errors.New("nack intentionally") 
		return nil
	}})

	ctx, cancel := context.WithCancel(context.Background())
	_, e := initClient(ctx)
	if e != nil {
		t.Error(e)
	}

	// make sure that the consumer is created before we cancel the context
	time.Sleep(time.Second * 1)
	cancel()
	logrus.Info("Cancelling background context")
	time.Sleep(time.Second * 3)
}

func TestPublishMessage(t *testing.T) {
	common.LoadConfigFromFile("../app-conf-dev.yml")
	common.SetProp(common.PROP_RABBITMQ_USERNAME, "guest")
	common.SetProp(common.PROP_RABBITMQ_PASSWORD, "guest")

	ctx, cancel := context.WithCancel(context.Background())
	_, e := initClient(ctx)
	if e != nil {
		t.Error(e)
	}
	defer cancel()

	time.Sleep(time.Second * 1)

	for i := 0; i < 10; i ++ {
		e = PublishMsg("yo check me out", "my-exchange-one", "myKey1")
		if e != nil {
			t.Error(e)
		}
	}
}
