package miso

import (
	"errors"
	"fmt"
	"sync"
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
func SendToEventBus(c Rail, eventObject any, bus string) error {
	if bus == "" {
		return errBusNameEmpty
	}
	DeclareEventBus(bus)
	busName := busName(bus)
	return PublishJson(c, eventObject, busName, BUS_ROUTING_KEY)
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
	if Connected() {
		ch, err := NewChan()
		if err != nil {
			return fmt.Errorf("failed to obtain channel for event bus declaration, %w", err)
		}
		defer ch.Close()
		if err := DeclareQueue(ch, QueueRegistration{Name: busName, Durable: true}); err != nil {
			return err
		}
		if err := DeclareBinding(ch, BindingRegistration{Queue: busName, RoutingKey: BUS_ROUTING_KEY, Exchange: busName}); err != nil {
			return err
		}
		if err := DeclareExchange(ch, ExchangeRegistration{Name: busName, Durable: true, Kind: BUS_EXCHANGE_KIND}); err != nil {
			return err
		}
		declaredBus.Store(busName, true)
		return nil
	}

	// not connected yet, prepare the registration instead
	RegisterQueue(QueueRegistration{Name: busName, Durable: true})
	RegisterBinding(BindingRegistration{Queue: busName, RoutingKey: BUS_ROUTING_KEY, Exchange: busName})
	RegisterExchange(ExchangeRegistration{Name: busName, Durable: true, Kind: BUS_EXCHANGE_KIND})
	declaredBus.Store(busName, true)
	return nil
}

// Subscribe to event bus.
//
// Internally, it registers a listener for the queue identified by the bus name.
//
// It also calls DeclareEventBus(...) automatically before it registers the listeners.
func SubscribeEventBus[T any](bus string, concurrency int, listener func(rail Rail, t T) error) {
	if bus == "" {
		panic(errBusNameEmpty)
	}

	DeclareEventBus(bus)

	if concurrency < 1 {
		concurrency = 1
	}
	AddListener(JsonMsgListener[T]{QueueName: busName(bus), Handler: listener, NumOfRoutines: concurrency})
}

func busName(bus string) string {
	return "event.bus." + bus
}
