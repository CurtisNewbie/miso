package miso

import (
	"testing"
	"time"
)

func preTest() {
	SetProp(PropRabbitMqUsername, "guest")
	SetProp(PropRabbitMqPassword, "guest")
}

func TestDeclareEventBus(t *testing.T) {
	preTest()
	NewEventBus("test-bus")

	rail, cancel := EmptyRail().WithCancel()
	if e := StartRabbitMqClient(rail); e != nil {
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

	rail, cancel := EmptyRail().WithCancel()
	if e := StartRabbitMqClient(rail); e != nil {
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

	rail, cancel := EmptyRail().WithCancel()
	if e := StartRabbitMqClient(rail); e != nil {
		t.Fatal(e)
	}

	time.Sleep(time.Second * 3)

	cancel()
	time.Sleep(time.Second * 3)
}
