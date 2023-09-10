package miso

import (
	"context"
	"testing"
	"time"
)

func preTest() {
	SetProp(PROP_RABBITMQ_USERNAME, "guest")
	SetProp(PROP_RABBITMQ_PASSWORD, "guest")
}

func TestDeclareEventBus(t *testing.T) {
	preTest()
	NewEventBus("test-bus")

	ctx, cancel := context.WithCancel(context.Background())
	if e := StartRabbitMqClient(ctx); e != nil {
		t.Fatal(e)
	}

	cancel()
	time.Sleep(time.Second * 3)
}

type Dummy struct {
	Name string
	Age  int
}

func TestSendToEventBus(t *testing.T) {
	preTest()

	ctx, cancel := context.WithCancel(context.Background())
	if e := StartRabbitMqClient(ctx); e != nil {
		t.Fatal(e)
	}

	if e := PubEventBus(EmptyRail(), &Dummy{Name: "apple", Age: 1}, "test-bus"); e != nil {
		t.Fatal(e)
	}

	cancel()
	time.Sleep(time.Second * 3)
}

func TestSubscribeEventBus(t *testing.T) {
	preTest()

	SubEventBus("test-bus", 1, func(rail Rail, t Dummy) error {
		rail.Infof("received dummy: %+v", t)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	if e := StartRabbitMqClient(ctx); e != nil {
		t.Fatal(e)
	}

	time.Sleep(time.Second * 3)

	cancel()
	time.Sleep(time.Second * 3)
}
