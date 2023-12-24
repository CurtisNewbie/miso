package miso

import (
	"errors"
	"fmt"
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
func NewEventBus(name string) error {
	if name == "" {
		return errBusNameEmpty
	}
	if _, ok := declaredBus.Load(name); ok {
		return nil // race condition is harmless, don't worry
	}

	// already connected
	if RabbitConnected() {
		ch, err := NewRabbitChan()
		if err != nil {
			return fmt.Errorf("failed to obtain channel for event bus declaration, %w", err)
		}
		defer ch.Close()
		if err := DeclareRabbitQueue(ch, QueueRegistration{Name: name, Durable: true}); err != nil {
			return err
		}
		if err := DeclareRabbitBinding(ch, BindingRegistration{Queue: name, RoutingKey: BusRoutingKey, Exchange: name}); err != nil {
			return err
		}
		if err := DeclareRabbitExchange(ch, ExchangeRegistration{Name: name, Durable: true, Kind: BusExchangeKind}); err != nil {
			return err
		}
		declaredBus.Store(name, true)
		return nil
	}

	// not connected yet, prepare the registration instead
	RegisterRabbitQueue(QueueRegistration{Name: name, Durable: true})
	RegisterRabbitBinding(BindingRegistration{Queue: name, RoutingKey: BusRoutingKey, Exchange: name})
	RegisterRabbitExchange(ExchangeRegistration{Name: name, Durable: true, Kind: BusExchangeKind})
	declaredBus.Store(name, true)
	return nil
}

// Subscribe to event bus.
//
// Internally, it registers a listener for the queue identified by the bus name.
func SubEventBus[T any](name string, concurrency int, listener func(rail Rail, t T) error) error {
	if name == "" {
		return errBusNameEmpty
	}
	if concurrency < 1 {
		concurrency = 1
	}
	AddRabbitListener(JsonMsgListener[T]{QueueName: name, Handler: listener, NumOfRoutines: concurrency})
	return nil
}
