package bus

import (
	"errors"
	"sync"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/rabbitmq"
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
// Before calling this method, the DeclareEventBus(...) must be called once to create the necessary components.
func SendToEventBus(c common.Rail, eventObject any, bus string) error {
	if bus == "" {
		return errBusNameEmpty
	}
	busName := busName(bus)
	return rabbitmq.PublishJson(c, eventObject, busName, BUS_ROUTING_KEY)
}

// Declare event bus.
//
// Internally, it creates the RabbitMQ queue, binding, and exchange that are uniformally identified by the same bus name.
func DeclareEventBus(bus string) {
	if bus == "" {
		panic(errBusNameEmpty)
	}
	busName := busName(bus)
	if _, ok := declaredBus.Load(busName); ok {
		return // race condition is harmless, don't worry
	}

	rabbitmq.RegisterQueue(rabbitmq.QueueRegistration{Name: busName, Durable: true})
	rabbitmq.RegisterBinding(rabbitmq.BindingRegistration{Queue: busName, RoutingKey: BUS_ROUTING_KEY, Exchange: busName})
	rabbitmq.RegisterExchange(rabbitmq.ExchangeRegistration{Name: busName, Durable: true, Kind: BUS_EXCHANGE_KIND})
	declaredBus.Store(busName, true)
}

// Subscribe to event bus.
//
// Internally, it registers a listener for the queue identified by the bus name.
//
// It also calls DeclareEventBus(...) automatically before it registers the listeners.
func SubscribeEventBus[T any](bus string, concurrency int, listener func(rail common.Rail, t T) error) {
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
