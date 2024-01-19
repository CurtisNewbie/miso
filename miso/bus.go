package miso

import (
	"errors"
	"sync"
)

const (
	BusRoutingKey   = "#"
	BusExchangeKind = "direct"
)

var (
	errBusNameEmpty = errors.New("bus name cannot be empty")
	declaredBus     sync.Map
)

// Send msg to event bus.
//
// It's identical to sending a message to an exchange identified by the name using routing key '#'.
//
// Before calling this method, the NewEventBus(...) should be called at least once to create the necessary components.
func PubEventBus(rail Rail, eventObject any, name string) error {
	if name == "" {
		return errBusNameEmpty
	}
	return PublishJson(rail, eventObject, name, BusRoutingKey)
}

// Declare event bus.
//
// It basically is to create an direct exchange and a queue identified by the name, and bind them using routing key '#'.
func NewEventBus(name string) {
	if name == "" {
		panic("event bus name is empty")
	}
	if _, ok := declaredBus.Load(name); ok {
		return // race condition is harmless, don't worry
	}

	// not connected yet, prepare the registration instead
	RegisterRabbitQueue(QueueRegistration{Name: name, Durable: true})
	RegisterRabbitBinding(BindingRegistration{Queue: name, RoutingKey: BusRoutingKey, Exchange: name})
	RegisterRabbitExchange(ExchangeRegistration{Name: name, Durable: true, Kind: BusExchangeKind})
	declaredBus.Store(name, true)
}

// Subscribe to event bus.
//
// Internally, it calls NewEventBus(...) and registers a listener for the queue identified by the bus name.
func SubEventBus[T any](name string, concurrency int, listener func(rail Rail, t T) error) {
	if name == "" {
		panic("event bus name is empty")
	}
	if concurrency < 1 {
		concurrency = 1
	}
	NewEventBus(name)
	AddRabbitListener(JsonMsgListener[T]{QueueName: name, Handler: listener, NumOfRoutines: concurrency})
}
