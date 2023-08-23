package bus

import (
	"context"
	"testing"
	"time"

	"github.com/curtisnewbie/miso/core"
	"github.com/curtisnewbie/miso/rabbitmq"
)

func preTest() {
	core.SetProp(core.PROP_RABBITMQ_USERNAME, "guest")
	core.SetProp(core.PROP_RABBITMQ_PASSWORD, "guest")
}

func TestDeclareEventBus(t *testing.T) {
	preTest()
	DeclareEventBus("test-bus")

	ctx, cancel := context.WithCancel(context.Background())
	if e := rabbitmq.StartRabbitMqClient(ctx); e != nil {
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
	if e := rabbitmq.StartRabbitMqClient(ctx); e != nil {
		t.Fatal(e)
	}

	if e := SendToEventBus(core.EmptyRail(), &Dummy{Name: "apple", Age: 1}, "test-bus"); e != nil {
		t.Fatal(e)
	}

	cancel()
	time.Sleep(time.Second * 3)
}

func TestSubscribeEventBus(t *testing.T) {
	preTest()

	SubscribeEventBus("test-bus", 1, func(rail core.Rail, t Dummy) error {
		rail.Infof("received dummy: %+v", t)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	if e := rabbitmq.StartRabbitMqClient(ctx); e != nil {
		t.Fatal(e)
	}

	time.Sleep(time.Second * 3)

	cancel()
	time.Sleep(time.Second * 3)
}
