package rabbitmq

import (
	"fmt"
	"sync"

	"github.com/curtisnewbie/gocommon/common"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

var (
	// TODO Only one connection and channel for now
	conn *amqp.Connection
	mu   sync.Mutex
)

func init() {
	common.SetDefProp(common.PROP_RABBITMQ_HOST, "localhost")
	common.SetDefProp(common.PROP_RABBITMQ_PORT, 5672)
	common.SetDefProp(common.PROP_RABBITMQ_USERNAME, "")
	common.SetDefProp(common.PROP_RABBITMQ_PASSWORD, "")
	common.SetDefProp(common.PROP_RABBITMQ_VHOST, "")
}

/*
	Declare durable queues

	It looks for PROP:

		PROP_RABBITMQ_DEC_QUEUE
*/
func declareQueues(ch *amqp.Channel) error {
	common.NonNil(ch, "channel is nil")
	if !common.ContainsProp(common.PROP_RABBITMQ_DEC_QUEUE) {
		return nil
	}

	directQueues := common.GetPropStringSlice(common.PROP_RABBITMQ_DEC_QUEUE)
	for _, queue := range directQueues {
		dqueue, e := ch.QueueDeclare(queue, true, false, false, false, nil)
		if e != nil {
			return e
		}
		logrus.Infof("Declared queue '%s'", dqueue.Name)
	}
	return nil
}

/*
	Declare bindings

	It looks for PROP:

		PROP_RABBITMQ_DEC_QUEUE
		PROP_RABBITMQ_DEC_BINDING + "." + queueName + ".key"
		PROP_RABBITMQ_DEC_BINDING + "." + queueName + ".exchange"
*/
func declareBindings(ch *amqp.Channel) error {
	common.NonNil(ch, "channel is nil")
	if !common.ContainsProp(common.PROP_RABBITMQ_DEC_QUEUE) {
		return nil
	}

	directQueues := common.GetPropStringSlice(common.PROP_RABBITMQ_DEC_QUEUE)
	for _, queue := range directQueues {
		routingKeyPropKey := bindRoutingKeyProp(queue)
		exchangePropKey := bindExchangeProp(queue)

		if !common.ContainsProp(exchangePropKey) || !common.ContainsProp(routingKeyPropKey) {
			continue
		}

		routingKey := common.GetPropStr(routingKeyPropKey)
		exchange := common.GetPropStr(exchangePropKey)
		e := ch.QueueBind(queue, routingKey, exchange, false, nil)
		if e != nil {
			return e
		}
		logrus.Infof("Declared binding for queue '%s' to exchange '%s' using routingKey '%s'", queue, exchange, routingKey)
	}
	return nil
}

func bindRoutingKeyProp(queue string) (propKey string) {
	propKey = common.PROP_RABBITMQ_DEC_BINDING + "." + queue + ".key"
	return
}

func bindExchangeProp(queue string) (propKey string) {
	propKey = common.PROP_RABBITMQ_DEC_BINDING + "." + queue + ".exchange"
	return
}

/*
	Declare exchanges

	It looks for PROP:

		PROP_RABBITMQ_DEC_EXCHANGE
*/
func declareExchanges(ch *amqp.Channel) error {
	common.NonNil(ch, "channel is nil")
	if !common.ContainsProp(common.PROP_RABBITMQ_DEC_EXCHANGE) {
		return nil
	}

	directExchange := common.GetPropStringSlice(common.PROP_RABBITMQ_DEC_EXCHANGE)
	for _, v := range directExchange {
		// exchange type, only direct exchange supported for now
		exg_type := "direct" 
		e := ch.ExchangeDeclare(v, exg_type, true, false, false, false, nil)
		if e != nil {
			return e
		}
		logrus.Infof("Declared %s exchange '%s'", exg_type, v)
	}
	return nil
}

/*
	Init RabbitMQ Connection
*/
func InitConnection() error {
	mu.Lock()
	defer mu.Unlock()
	if conn != nil {
		return nil
	}

	c := amqp.Config{}
	username := common.GetPropStr(common.PROP_RABBITMQ_USERNAME)
	password := common.GetPropStr(common.PROP_RABBITMQ_PASSWORD)
	vhost := common.GetPropStr(common.PROP_RABBITMQ_VHOST)
	host := common.GetPropStr(common.PROP_RABBITMQ_HOST)
	port := common.GetPropInt(common.PROP_RABBITMQ_PORT)
	dialUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", username, password, host, port, vhost)
	conn, e := amqp.DialConfig(dialUrl, c)
	if e != nil {
		return e
	}

	ch, e := conn.Channel()
	if e != nil {
		return e
	}

	declareQueues(ch)
	declareExchanges(ch)
	declareBindings(ch)
	return nil
}
