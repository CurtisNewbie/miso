package bus

import (
	"context"
	"testing"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/rabbitmq"
	"github.com/sirupsen/logrus"
)

func preTest() {
	common.SetProp(common.PROP_RABBITMQ_USERNAME, "guest")
	common.SetProp(common.PROP_RABBITMQ_PASSWORD, "guest")
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

	if e := SendToEventBus(common.EmptyRail(), &Dummy{Name: "apple", Age: 1}, "test-bus"); e != nil {
		t.Fatal(e)
	}

	cancel()
	time.Sleep(time.Second * 3)
}

func TestSubscribeEventBus(t *testing.T) {
	preTest()

	SubscribeEventBus("test-bus", 1, func(t Dummy) error {
		logrus.Infof("received dummy: %+v", t)
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
