package bus

import (
	"errors"
	"fmt"
	"sync"

	"github.com/curtisnewbie/miso/core"
	"github.com/curtisnewbie/miso/rabbitmq"
)

const (
	BUS_ROUTING_KEY   = "#"
	BUS_EXCHANGE_KIND = "direct"
)

var (
	errBusNameEmpty = errors.New("bus name cannot be empty")
	declaredBus     sync.Map
)

// Send msg to event bus.
//
// Internally, it serialize eventObject to a json string and dispatch the message to the exchange that is identified by the bus name.
//
// Before calling this method, the DeclareEventBus(...) should be called at least once to create the necessary components.
func SendToEventBus(c core.Rail, eventObject any, bus string) error {
	if bus == "" {
		return errBusNameEmpty
	}
	DeclareEventBus(bus)
	busName := busName(bus)
	return rabbitmq.PublishJson(c, eventObject, busName, BUS_ROUTING_KEY)
}

// Declare event bus.
//
// Internally, it creates the RabbitMQ queue, binding, and exchange that are uniformally identified by the same bus name.
func DeclareEventBus(bus string) error {
	if bus == "" {
		panic(errBusNameEmpty)
	}
	busName := busName(bus)
	if _, ok := declaredBus.Load(busName); ok {
		return nil // race condition is harmless, don't worry
	}

	// already connected
	if rabbitmq.Connected() {
		ch, err := rabbitmq.NewChan()
		if err != nil {
			return fmt.Errorf("failed to obtain channel for event bus declaration, %w", err)
		}
		defer ch.Close()
		if err := rabbitmq.DeclareQueue(ch, rabbitmq.QueueRegistration{Name: busName, Durable: true}); err != nil {
			return err
		}
		if err := rabbitmq.DeclareBinding(ch, rabbitmq.BindingRegistration{Queue: busName, RoutingKey: BUS_ROUTING_KEY, Exchange: busName}); err != nil {
			return err
		}
		if err := rabbitmq.DeclareExchange(ch, rabbitmq.ExchangeRegistration{Name: busName, Durable: true, Kind: BUS_EXCHANGE_KIND}); err != nil {
			return err
		}
		declaredBus.Store(busName, true)
		return nil
	}

	// not connected yet, prepare the registration instead
	rabbitmq.RegisterQueue(rabbitmq.QueueRegistration{Name: busName, Durable: true})
	rabbitmq.RegisterBinding(rabbitmq.BindingRegistration{Queue: busName, RoutingKey: BUS_ROUTING_KEY, Exchange: busName})
	rabbitmq.RegisterExchange(rabbitmq.ExchangeRegistration{Name: busName, Durable: true, Kind: BUS_EXCHANGE_KIND})
	declaredBus.Store(busName, true)
	return nil
}

// Subscribe to event bus.
//
// Internally, it registers a listener for the queue identified by the bus name.
//
// It also calls DeclareEventBus(...) automatically before it registers the listeners.
func SubscribeEventBus[T any](bus string, concurrency int, listener func(rail core.Rail, t T) error) {
	if bus == "" {
		panic(errBusNameEmpty)
	}

	DeclareEventBus(bus)

	if concurrency < 1 {
		concurrency = 1
	}
	rabbitmq.AddListener(rabbitmq.JsonMsgListener[T]{QueueName: busName(bus), Handler: listener, NumOfRoutines: concurrency})
}

func busName(bus string) string {
	return "event.bus." + bus
}
