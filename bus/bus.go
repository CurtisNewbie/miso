package bus

import (
	"errors"

	"github.com/curtisnewbie/gocommon/rabbitmq"
)

const (
	BUS_ROUTING_KEY   = "#"
	BUS_EXCHANGE_KIND = "direct"
)

var (
	errBusNameEmpty = errors.New("bus name cannot be empty")
)

// Send msg to event bus
//
// Internally, it serialize eventObject to a json string and dispatch the message to the exchange that is identified by the bus name
func SendToEventBus(eventObject any, bus string) error {
	if bus == "" {
		return errBusNameEmpty
	}
	busName := busName(bus)
	return rabbitmq.PublishJson(eventObject, busName, BUS_ROUTING_KEY)
}

// Declare event bus
//
// Inernally, it creates the RabbitMQ queue, binding, and exchange that are uniformally identified the same bus name
func DeclareEventBus(bus string) error {
	if bus == "" {
		return errBusNameEmpty
	}
	busName := busName(bus)
	rabbitmq.RegisterQueue(rabbitmq.QueueRegistration{Name: busName, Durable: true})
	rabbitmq.RegisterBinding(rabbitmq.BindingRegistration{Queue: busName, RoutingKey: BUS_ROUTING_KEY, Exchange: busName})
	rabbitmq.RegisterExchange(rabbitmq.ExchangeRegistration{Name: busName, Durable: true, Kind: BUS_EXCHANGE_KIND})
	return nil
}

// Subscribe to event bus
//
// Internally, it registers a listener for the queue identified by the bus name
func SubscribeEventBus[T any](bus string, concurrency int, listener func(t T) error) error {
	if bus == "" {
		return errBusNameEmpty
	}
	if concurrency < 1 {
		concurrency = 1
	}
	rabbitmq.AddListener(rabbitmq.JsonMsgListener[T]{QueueName: busName(bus), Handler: listener, NumOfRoutines: concurrency})
	return nil
}

func busName(bus string) string {
	return "event.bus." + bus
}