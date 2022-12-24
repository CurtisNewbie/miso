package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

var (
	conn         *amqp.Connection
	msgListeners []MsgListener
	mu           sync.Mutex
	qos          = 250
)

func init() {
	common.SetDefProp(common.PROP_RABBITMQ_HOST, "localhost")
	common.SetDefProp(common.PROP_RABBITMQ_PORT, 5672)
	common.SetDefProp(common.PROP_RABBITMQ_USERNAME, "")
	common.SetDefProp(common.PROP_RABBITMQ_PASSWORD, "")
	common.SetDefProp(common.PROP_RABBITMQ_VHOST, "")
}

/*
	Message Listener for Queue
*/
type MsgListener struct {
	/* Name of the queue */
	QueueName string
	/* Handler of message */
	Handler   func(payload []byte, contentType string, messageId string) error
}

/*
	Add message Listener

	Listeners will be registered in StartRabbitMqClient func when the connection to broker is established.
*/
func AddListener(listener MsgListener) {
	mu.Lock()
	defer mu.Unlock()
	msgListeners = append(msgListeners, listener)
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

/*
	Get prop key for routing key of queue

		PROP_RABBITMQ_DEC_BINDING + "." + queueName + ".key"
*/
func bindRoutingKeyProp(queue string) (propKey string) {
	propKey = common.PROP_RABBITMQ_DEC_BINDING + "." + queue + ".key"
	return
}

/*
	Get prop key for exchange name of queue

		PROP_RABBITMQ_DEC_BINDING + "." + queueName + ".exchange"
*/
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
	Start RabbitMQ Client (Asynchronous)

	This func will attempt to establish connection to broker, declare queues, exchanges and bindings. 

	Listeners are also created once the intial setup is done. 

	When connection is lost, it will attmpt to reconnect to recover, unless the given context is done.

	To register listener, please use 'AddListener' func 

*/
func StartRabbitMqClient(ctx context.Context) {
	go func() {
		for {
			notifyCloseChan, err := initConnection(ctx)
			if err != nil {
				logrus.Infof("Error connecting to RabbitMQ: %v", err)
				time.Sleep(time.Second * 5)
				continue
			}
			select {
			// block until connection is closed, then reconnect, thus continue
			case <-notifyCloseChan:
				continue
			// context is done, close the connection, and exit 
			case <-ctx.Done(): 
				if err := closeConnection(); err != nil {
					logrus.Warnf("Failed to close connection to RabbitMQ: %v", err)
				}
				return
			}
		}
	}()
}

/*
	Try to close RabbitMQ Connection
*/
func closeConnection() (error) {
	mu.Lock()
	defer mu.Unlock()
	if conn == nil {
		return nil
	}

	return conn.Close()
}

/*
	Init RabbitMQ Connection
*/
func initConnection(ctx context.Context) (chan *amqp.Error, error) {
	mu.Lock()
	defer mu.Unlock()

	if conn == nil {
		logrus.Info("Connecting to RabbitMQ")
		c := amqp.Config{}
		username := common.GetPropStr(common.PROP_RABBITMQ_USERNAME)
		password := common.GetPropStr(common.PROP_RABBITMQ_PASSWORD)
		vhost := common.GetPropStr(common.PROP_RABBITMQ_VHOST)
		host := common.GetPropStr(common.PROP_RABBITMQ_HOST)
		port := common.GetPropInt(common.PROP_RABBITMQ_PORT)
		dialUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", username, password, host, port, vhost)
		cn, e := amqp.DialConfig(dialUrl, c)
		if e != nil {
			return nil, e
		}
		conn = cn
	}

	notifyCloseChan := make(chan *amqp.Error)
	conn.NotifyClose(notifyCloseChan)

	logrus.Infof("Creating Channel to RabbitMQ")
	ch, e := conn.Channel()
	if e != nil {
		return nil, e
	}

	declareQueues(ch)
	declareExchanges(ch)
	declareBindings(ch)

	e = ch.Qos(qos, 0, false)
	if e != nil {
		return nil, e
	}

	// register consumers for queues
	for _, v := range msgListeners {
		listener := v
		msgs, err := ch.Consume(listener.QueueName, "", false, false, false, false, nil)
		if err != nil {
			log.Fatalf("Failed to listen to '%s', err: %v", listener.QueueName, err)
		}

		// go routine for each queue
		go func() {
			logrus.Infof("Created RabbitMQ Consumer for queue: '%s'", listener.QueueName)
			for msg := range msgs {
				e := listener.Handler(msg.Body, msg.ContentType, msg.MessageId)
				if e != nil {
					logrus.Warnf("Failed to handle message for queue: '%s', err: %v, body: %v, msgId: %s", listener.QueueName, e, msg.Body, msg.MessageId)
					msg.Nack(false, true)
				} else {
					msg.Ack(false)
				}
			}
			logrus.Infof("RabbitMQ Consumer for queue '%s' is closed", listener.QueueName)
		}()
	}

	return notifyCloseChan, nil
}
