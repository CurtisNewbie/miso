package rabbit

import (
	"errors"
	"testing"
	"time"

	"github.com/curtisnewbie/miso/miso"
)

func preTest() {
	miso.SetProp(PropRabbitMqUsername, "guest")
	miso.SetProp(PropRabbitMqPassword, "guest")
}

func TestDeclareEventBus(t *testing.T) {
	preTest()
	NewEventBus("test-bus")

	rail, cancel := miso.EmptyRail().WithCancel()
	if e := StartRabbitMqClient(rail); e != nil {
		t.Fatal(e)
	}

	cancel()
	time.Sleep(time.Second * 3)
}

func TestSendToEventBus(t *testing.T) {
	preTest()

	type dummy struct {
		Name string
		Age  int
	}

	rail, cancel := miso.EmptyRail().WithCancel()
	if e := StartRabbitMqClient(rail); e != nil {
		t.Fatal(e)
	}

	if e := PubEventBus(miso.EmptyRail(), &dummy{Name: "apple", Age: 1}, "test-bus"); e != nil {
		t.Fatal(e)
	}

	cancel()
	time.Sleep(time.Second * 3)
}

func TestSubscribeEventBus(t *testing.T) {
	preTest()

	type dummy struct {
		Name string
		Age  int
	}

	SubEventBus("test-bus", 1, func(rail miso.Rail, t dummy) error {
		rail.Infof("received dummy: %+v", t)
		return nil
	})

	rail, cancel := miso.EmptyRail().WithCancel()
	if e := StartRabbitMqClient(rail); e != nil {
		t.Fatal(e)
	}

	time.Sleep(time.Second * 3)

	cancel()
	time.Sleep(time.Second * 3)
}

func TestNewPipeline(t *testing.T) {
	preTest()

	type dummy struct{}
	NewEventPipeline[dummy]("test-pipe")

	rail, cancel := miso.EmptyRail().WithCancel()
	if e := StartRabbitMqClient(rail); e != nil {
		t.Fatal(e)
	}

	cancel()
	time.Sleep(time.Second * 3)
}

func TestPipeSend(t *testing.T) {
	preTest()

	type dummy struct {
		Attr string
	}

	pipe := NewEventPipeline[dummy]("test-pipe")
	pipe.MaxRetry(2)

	rail, cancel := miso.EmptyRail().WithCancel()
	if e := StartRabbitMqClient(rail); e != nil {
		t.Fatal(e)
	}

	if err := pipe.Send(miso.EmptyRail(), dummy{Attr: "yes"}); err != nil {
		t.Fatal(err)
	}

	cancel()
	time.Sleep(time.Second * 3)
}

func TestPipeListen(t *testing.T) {
	preTest()

	type dummy struct {
		Attr string
	}
	pipe := NewEventPipeline[dummy]("test-pipe")

	pipe.Listen(1, func(rail miso.Rail, t dummy) error {
		rail.Infof("received dummy: %+v", t)
		return nil
	})

	rail, cancel := miso.EmptyRail().WithCancel()
	if e := StartRabbitMqClient(rail); e != nil {
		t.Fatal(e)
	}

	cancel()
	time.Sleep(time.Second * 3)
}

func TestPipeListenRetry(t *testing.T) {
	preTest()

	type dummy struct {
		Attr string
	}
	pipe := NewEventPipeline[dummy]("test-pipe")

	pipe.Listen(1, func(rail miso.Rail, t dummy) error {
		rail.Infof("received dummy: %+v", t)
		return errors.New("nope")
	})

	rail, cancel := miso.EmptyRail().WithCancel()
	if e := StartRabbitMqClient(rail); e != nil {
		t.Fatal(e)
	}

	time.Sleep(time.Second * 17)

	cancel()
	time.Sleep(time.Second * 3)
}
