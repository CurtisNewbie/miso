package rabbit

import (
	"errors"
	"sync"

	"github.com/curtisnewbie/miso/miso"
)

const (
	BusRoutingKey   = "#"
	BusExchangeKind = "direct"
)

var (
	errBusNameEmpty = errors.New("bus name cannot be empty")
)

var busModule = miso.InitAppModuleFunc(func() *eventBusModule {
	return &eventBusModule{
		declaredBus: &sync.Map{},
	}
})

type eventBusModule struct {
	declaredBus *sync.Map
}

// Send msg to event bus with "miso-rabbitmq-max-retry" header.
//
// Notice that redelivery mechanism is implemented on the consumer side (by miso).
//
// It's identical to sending a message to an exchange identified by the name using routing key '#'.
//
// Before calling this method, the NewEventBus(...) should be called at least once to create the necessary components.
func PubRetryEventBus(rail miso.Rail, eventObject any, name string, retry int) error {
	return PubEventBusHeaders(rail, eventObject, name, map[string]any{HeaderRabbitMaxRetry: retry})
}

// Send msg to event bus.
//
// It's identical to sending a message to an exchange identified by the name using routing key '#'.
//
// Before calling this method, the NewEventBus(...) should be called at least once to create the necessary components.
func PubEventBus(rail miso.Rail, eventObject any, name string) error {
	return PubEventBusHeaders(rail, eventObject, name, nil)
}

// Send msg to event bus.
//
// It's identical to sending a message to an exchange identified by the name using routing key '#'.
//
// Before calling this method, the NewEventBus(...) should be called at least once to create the necessary components.
func PubEventBusHeaders(rail miso.Rail, eventObject any, name string, headers map[string]any) error {
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
	m := busModule()
	if _, ok := m.declaredBus.Load(name); ok {
		return // race condition is harmless, don't worry
	}

	// not connected yet, prepare the registration instead
	RegisterRabbitQueue(QueueRegistration{Name: name, Durable: true})
	RegisterRabbitBinding(BindingRegistration{Queue: name, RoutingKey: BusRoutingKey, Exchange: name})
	RegisterRabbitExchange(ExchangeRegistration{Name: name, Durable: true, Kind: BusExchangeKind})
	m.declaredBus.Store(name, true)
}

// Subscribe to event bus.
//
// Internally, it calls NewEventBus(...) and registers a listener for the queue identified by the bus name.
func SubEventBus[T any](name string, concurrency int, listener func(rail miso.Rail, t T) error) {
	SubEventBusQos[T](name, concurrency, 0, listener)
}

// Subscribe to event bus.
//
// Internally, it calls NewEventBus(...) and registers a listener for the queue identified by the bus name.
func SubEventBusQos[T any](name string, concurrency int, qos int, listener func(rail miso.Rail, t T) error) {
	if name == "" {
		panic("event bus name is empty")
	}
	if concurrency < 1 {
		concurrency = 1
	}
	NewEventBus(name)
	AddRabbitListener(JsonMsgListener[T]{QueueName: name, Handler: listener, NumOfRoutines: concurrency, Qos: qos})
}

// EventPipeline is a thin wrapper of NewEventBus, SubEventBus and PubEventBus.
// It's used to make things easier and more consistent.
//
// Use NewEventPipeline to instantiate.
type EventPipeline[T any] struct {
	name      string
	logPaylod bool
	maxRetry  int
	qos       int
}

// Name of the pipeline.
func (ep *EventPipeline[T]) Name() string {
	return ep.name
}

// Log payload in message consumer.
func (ep *EventPipeline[T]) LogPayload() *EventPipeline[T] {
	ep.logPaylod = true
	return ep
}

// Document EventPipline in the generated apidoc.
func (ep *EventPipeline[T]) Document(name string, desc string, provider string) *EventPipeline[T] {
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
func (ep *EventPipeline[T]) Send(rail miso.Rail, event T) error {
	if ep.maxRetry > -1 {
		return PubRetryEventBus(rail, event, ep.name, ep.maxRetry)
	}
	return PubEventBus(rail, event, ep.name)
}

// Specify QOS for listener, should be called before #Listen func.
func (ep *EventPipeline[T]) ListenerQos(v int) *EventPipeline[T] {
	if v < 0 {
		return ep
	}
	ep.qos = v
	return ep
}

// Call SubEventBus.
func (ep *EventPipeline[T]) Listen(concurrency int, listener func(rail miso.Rail, t T) error) *EventPipeline[T] {
	SubEventBusQos(ep.name, concurrency, ep.qos, func(rail miso.Rail, t T) error {
		if ep.logPaylod {
			rail.Infof("Pipeline %s receive %+v", ep.name, t)
		} else {
			rail.Infof("Pipeline %s receive event", ep.name)
		}
		return listener(rail, t)
	})
	return ep
}

// Create new EventPipeline. NewEventBus is internally called as well.
func NewEventPipeline[T any](name string) *EventPipeline[T] {
	NewEventBus(name)
	return &EventPipeline[T]{
		name:     name,
		maxRetry: -1,
	}
}
