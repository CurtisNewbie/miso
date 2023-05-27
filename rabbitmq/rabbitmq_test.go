package rabbitmq

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/sirupsen/logrus"
)

type Dummy struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

func msgHandler(payload string) error {
	logrus.Infof("Received message %s", payload)
	// return errors.New("nack intentionally")
	return nil
}

func jsonMsgHandler(payload Dummy) error {
	logrus.Infof("Received message %s", payload)
	return nil
}

func TestInitClient(t *testing.T) {
	common.LoadConfigFromFile("../app-conf-dev.yml")
	common.SetProp(common.PROP_RABBITMQ_USERNAME, "guest")
	common.SetProp(common.PROP_RABBITMQ_PASSWORD, "guest")
	common.SetProp(common.PROP_RABBITMQ_CONSUMER_PARALLISM, 2)

	AddListener(JsonMsgListener[Dummy]{QueueName: "dummy-queue", Handler: jsonMsgHandler})
	AddListener(MsgListener{QueueName: "my-first-queue", Handler: msgHandler})

	ctx, cancel := context.WithCancel(context.Background())
	_, e := initClient(ctx)
	if e != nil {
		t.Fatal(e)
	}

	// make sure that the consumer is created before we cancel the context
	time.Sleep(time.Second * 1)
	cancel()
	if e := ClientDisconnect(); e != nil {
		t.Fatal(e)
	}

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

	for i := 0; i < 10; i++ {
		e = PublishText("yo check me out", "my-exchange-one", "myKey1")
		if e != nil {
			t.Error(e)
		}
	}
}

func TestPublishJsonMessage(t *testing.T) {
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

	for i := 0; i < 10; i++ {
		dummy := Dummy{Name: fmt.Sprintf("dummy no.%v", i), Desc: "dummy with all the love"}
		e = PublishJson(dummy, "dummy-exchange", "#")
		if e != nil {
			t.Error(e)
		}
	}
}
