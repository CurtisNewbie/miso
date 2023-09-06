package miso

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

type RabbitDummy struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

func msgHandler(rail Rail, payload string) error {
	rail.Infof("Received message %s", payload)
	// return errors.New("nack intentionally")
	return nil
}

func jsonMsgHandler(rail Rail, payload RabbitDummy) error {
	rail.Infof("Received message %s", payload)
	return nil
}

func TestInitClient(t *testing.T) {
	c := EmptyRail()
	LoadConfigFromFile("../app-conf-dev.yml", c)
	SetProp(PROP_RABBITMQ_USERNAME, "guest")
	SetProp(PROP_RABBITMQ_PASSWORD, "guest")

	AddListener(JsonMsgListener[RabbitDummy]{QueueName: "dummy-queue", Handler: jsonMsgHandler})
	AddListener(MsgListener{QueueName: "my-first-queue", Handler: msgHandler})

	RegisterQueue(QueueRegistration{Name: "my-first-queue", Durable: true})
	RegisterQueue(QueueRegistration{Name: "my-second-queue", Durable: true})
	RegisterQueue(QueueRegistration{Name: "dummy-queue", Durable: true})

	RegisterExchange(ExchangeRegistration{Name: "my-exchange-one", Kind: "direct", Durable: true})
	RegisterExchange(ExchangeRegistration{Name: "my-exchange-two", Kind: "direct", Durable: true})
	RegisterExchange(ExchangeRegistration{Name: "dummy-exchange", Kind: "direct", Durable: true})

	RegisterBinding(BindingRegistration{Queue: "dummy-queue", RoutingKey: "#", Exchange: "dummy-exchange"})
	RegisterBinding(BindingRegistration{Queue: "my-first-queue", RoutingKey: "myKey1", Exchange: "my-exchange-one"})
	RegisterBinding(BindingRegistration{Queue: "my-second-queue", RoutingKey: "myKey2", Exchange: "my-exchange-two"})

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
	c := EmptyRail()
	LoadConfigFromFile("../app-conf-dev.yml", c)
	SetProp(PROP_RABBITMQ_USERNAME, "guest")
	SetProp(PROP_RABBITMQ_PASSWORD, "guest")
	logrus.SetLevel(logrus.DebugLevel)

	RegisterQueue(QueueRegistration{Name: "my-first-queue", Durable: true})
	RegisterQueue(QueueRegistration{Name: "my-second-queue", Durable: true})
	RegisterQueue(QueueRegistration{Name: "dummy-queue", Durable: true})

	RegisterExchange(ExchangeRegistration{Name: "my-exchange-one", Kind: "direct", Durable: true})
	RegisterExchange(ExchangeRegistration{Name: "my-exchange-two", Kind: "direct", Durable: true})
	RegisterExchange(ExchangeRegistration{Name: "dummy-exchange", Kind: "direct", Durable: true})

	RegisterBinding(BindingRegistration{Queue: "dummy-queue", RoutingKey: "#", Exchange: "dummy-exchange"})
	RegisterBinding(BindingRegistration{Queue: "my-first-queue", RoutingKey: "myKey1", Exchange: "my-exchange-one"})
	RegisterBinding(BindingRegistration{Queue: "my-second-queue", RoutingKey: "myKey2", Exchange: "my-exchange-two"})

	ctx, cancel := context.WithCancel(context.Background())
	e := StartRabbitMqClient(ctx)
	if e != nil {
		t.Error(e)
	}
	defer cancel()

	time.Sleep(time.Second * 1)

	for i := 0; i < 10; i++ {
		e = PublishText(c, "yo check me out", "my-exchange-one", "myKey1")
		if e != nil {
			t.Error(e)
		}
	}
	time.Sleep(time.Second * 3)
}

func TestPublishJsonMessage(t *testing.T) {
	c := EmptyRail()
	LoadConfigFromFile("../app-conf-dev.yml", c)
	SetProp(PROP_RABBITMQ_USERNAME, "guest")
	SetProp(PROP_RABBITMQ_PASSWORD, "guest")
	logrus.SetLevel(logrus.DebugLevel)

	ctx, cancel := context.WithCancel(context.Background())
	e := StartRabbitMqClient(ctx)
	if e != nil {
		t.Error(e)
	}
	defer cancel()

	time.Sleep(time.Second * 1)

	for i := 0; i < 50; i++ {
		j := i
		dummy := RabbitDummy{Name: fmt.Sprintf("dummy no.%v", j), Desc: "dummy with all the love"}
		e = PublishJson(c, dummy, "dummy-exchange", "#")
		if e != nil {
			t.Error(e)
		}
	}
	time.Sleep(time.Second * 3)
}
