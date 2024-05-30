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
	return PubEventBusHeaders(rail, eventObject, name, nil)
}

// Send msg to event bus.
//
// It's identical to sending a message to an exchange identified by the name using routing key '#'.
//
// Before calling this method, the NewEventBus(...) should be called at least once to create the necessary components.
func PubEventBusHeaders(rail Rail, eventObject any, name string, headers map[string]any) error {
	if name == "" {
		return errBusNameEmpty
	}
	return PublishJsonHeaders(rail, eventObject, name, BusRoutingKey, headers)
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

// EventPipeline is a thin wrapper of NewEventBus, SubEventBus and PubEventBus.
// It's used to make things easier and more consistent.
//
// Use NewEventPipeline to instantiate.
type EventPipeline[T any] struct {
	name      string
	logPaylod bool
	maxRetry  int
}

func (ep *EventPipeline[T]) Name() string {
	return ep.name
}

func (ep *EventPipeline[T]) LogPayload() *EventPipeline[T] {
	ep.logPaylod = true
	return ep
}

// Specify max retry times, by default, it's -1, meaning that the message will be redelivered forever until it's successfully consumed.
//
// The message redelivery mechanism is implemented in miso's message consumer not publisher.
func (ep *EventPipeline[T]) MaxRetry(n int) *EventPipeline[T] {
	ep.maxRetry = n
	return ep
}

// Call PubEventBus.
func (ep *EventPipeline[T]) Send(rail Rail, event T) error {
	if ep.maxRetry > -1 {
		return PubEventBusHeaders(rail, event, ep.name, map[string]any{HeaderRabbitMaxRetry: ep.maxRetry})
	}
	return PubEventBus(rail, event, ep.name)
}

// Call SubEventBus.
func (ep *EventPipeline[T]) Listen(concurrency int, listener func(rail Rail, t T) error) {
	SubEventBus[T](ep.name, concurrency, func(rail Rail, t T) error {
		if ep.logPaylod {
			rail.Infof("Pipeline %s receive %#v", ep.name, t)
		}
		return listener(rail, t)
	})
}

// Create new EventPipeline. NewEventBus is internally called as well.
func NewEventPipeline[T any](name string) *EventPipeline[T] {
	NewEventBus(name)
	return &EventPipeline[T]{
		name:     name,
		maxRetry: -1,
	}
}
