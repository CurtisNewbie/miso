package miso

import (
	"fmt"
	"testing"
	"time"
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
	rail, cancel := EmptyRail().WithCancel()
	LoadConfigFromFile("../conf_dev.yml", rail)
	SetProp(PropRabbitMqUsername, "guest")
	SetProp(PropRabbitMqPassword, "guest")

	AddRabbitListener(JsonMsgListener[RabbitDummy]{QueueName: "dummy-queue", Handler: jsonMsgHandler})
	AddRabbitListener(MsgListener{QueueName: "my-first-queue", Handler: msgHandler})

	RegisterRabbitQueue(QueueRegistration{Name: "my-first-queue", Durable: true})
	RegisterRabbitQueue(QueueRegistration{Name: "my-second-queue", Durable: true})
	RegisterRabbitQueue(QueueRegistration{Name: "dummy-queue", Durable: true})

	RegisterRabbitExchange(ExchangeRegistration{Name: "my-exchange-one", Kind: "direct", Durable: true})
	RegisterRabbitExchange(ExchangeRegistration{Name: "my-exchange-two", Kind: "direct", Durable: true})
	RegisterRabbitExchange(ExchangeRegistration{Name: "dummy-exchange", Kind: "direct", Durable: true})

	RegisterRabbitBinding(BindingRegistration{Queue: "dummy-queue", RoutingKey: "#", Exchange: "dummy-exchange"})
	RegisterRabbitBinding(BindingRegistration{Queue: "my-first-queue", RoutingKey: "myKey1", Exchange: "my-exchange-one"})
	RegisterRabbitBinding(BindingRegistration{Queue: "my-second-queue", RoutingKey: "myKey2", Exchange: "my-exchange-two"})

	_, e := initRabbitClient(rail)
	if e != nil {
		t.Fatal(e)
	}

	// make sure that the consumer is created before we cancel the context
	time.Sleep(time.Second * 1)
	cancel()
	if e := RabbitDisconnect(rail); e != nil {
		t.Fatal(e)
	}

	rail.Info("Cancelling background context")
	time.Sleep(time.Second * 3)
}

func TestPublishMessage(t *testing.T) {
	rail, cancel := EmptyRail().WithCancel()
	LoadConfigFromFile("../conf_dev.yml", rail)
	SetProp(PropRabbitMqUsername, "guest")
	SetProp(PropRabbitMqPassword, "guest")
	SetLogLevel("debug")

	RegisterRabbitQueue(QueueRegistration{Name: "my-first-queue", Durable: true})
	RegisterRabbitQueue(QueueRegistration{Name: "my-second-queue", Durable: true})
	RegisterRabbitQueue(QueueRegistration{Name: "dummy-queue", Durable: true})

	RegisterRabbitExchange(ExchangeRegistration{Name: "my-exchange-one", Kind: "direct", Durable: true})
	RegisterRabbitExchange(ExchangeRegistration{Name: "my-exchange-two", Kind: "direct", Durable: true})
	RegisterRabbitExchange(ExchangeRegistration{Name: "dummy-exchange", Kind: "direct", Durable: true})

	RegisterRabbitBinding(BindingRegistration{Queue: "dummy-queue", RoutingKey: "#", Exchange: "dummy-exchange"})
	RegisterRabbitBinding(BindingRegistration{Queue: "my-first-queue", RoutingKey: "myKey1", Exchange: "my-exchange-one"})
	RegisterRabbitBinding(BindingRegistration{Queue: "my-second-queue", RoutingKey: "myKey2", Exchange: "my-exchange-two"})

	e := StartRabbitMqClient(rail)
	if e != nil {
		t.Error(e)
	}
	defer cancel()

	time.Sleep(time.Second * 1)

	for i := 0; i < 10; i++ {
		e = PublishText(rail, "yo check me out", "my-exchange-one", "myKey1")
		if e != nil {
			t.Fatal(e)
		}
	}
	time.Sleep(time.Second * 3)
}

func TestPublishJsonMessage(t *testing.T) {
	rail, cancel := EmptyRail().WithCancel()
	LoadConfigFromFile("../conf_dev.yml", rail)
	SetProp(PropRabbitMqUsername, "guest")
	SetProp(PropRabbitMqPassword, "guest")
	SetLogLevel("debug")

	e := StartRabbitMqClient(rail)
	if e != nil {
		t.Error(e)
	}
	defer cancel()

	time.Sleep(time.Second * 1)

	for i := 0; i < 50; i++ {
		j := i
		dummy := RabbitDummy{Name: fmt.Sprintf("dummy no.%v", j), Desc: "dummy with all the love"}
		e = PublishJson(rail, dummy, "dummy-exchange", "#")
		if e != nil {
			t.Error(e)
		}
	}
	time.Sleep(time.Second * 3)
}
