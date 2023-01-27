package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

const (
	/*
		Consumer default values
	*/
	DEFAULT_QOS       = 68
	DEFAULT_PARALLISM = 1
	DEFAULT_RETRY     = -1
)

var (

	// Connection pointer, accessing it require obtaining 'mu' lock
	_conn *amqp.Connection
	// Global Mutex for connection and initialization stuff
	mu sync.Mutex

	/*
		Publisher
	*/
	pubChan    *amqp.Channel
	pubChanRwm sync.RWMutex
	pubWg      sync.WaitGroup

	/*
		Consumer
	*/
	msgListeners []MsgListener

	errPubChanClosed   = errors.New("publishing Channel is closed, unable to publish message")
	errMsgNotPublished = errors.New("message not published, server failed to confirm")
)

func init() {
	common.SetDefProp(common.PROP_RABBITMQ_ENABLED, false)
	common.SetDefProp(common.PROP_RABBITMQ_HOST, "localhost")
	common.SetDefProp(common.PROP_RABBITMQ_PORT, 5672)
	common.SetDefProp(common.PROP_RABBITMQ_USERNAME, "")
	common.SetDefProp(common.PROP_RABBITMQ_PASSWORD, "")
	common.SetDefProp(common.PROP_RABBITMQ_VHOST, "")
	common.SetDefProp(common.PROP_RABBITMQ_CONSUMER_QOS, DEFAULT_QOS)
	common.SetDefProp(common.PROP_RABBITMQ_CONSUMER_PARALLISM, DEFAULT_PARALLISM)
	common.SetDefProp(common.PROP_RABBITMQ_CONSUMER_RETRY, DEFAULT_RETRY)
}

/* Is RabbitMQ Enabled */
func IsEnabled() bool {
	return common.GetPropBool(common.PROP_RABBITMQ_ENABLED)
}

/*
	Message Listener for Queue
*/
type MsgListener struct {
	/* Name of the queue */
	QueueName string
	/* Handler of message */
	Handler func(payload string) error
}

func (m MsgListener) String() string {
	var funcName string = "nil"
	if m.Handler != nil {
		funcName = runtime.FuncForPC(reflect.ValueOf(m.Handler).Pointer()).Name()
	}

	return fmt.Sprintf("MsgListener{ QueueName: '%s', Handler: %s }", m.QueueName, funcName)
}

// Publish json message with confirmation
func PublishJson(obj any, exchange string, routingKey string) error {
	j, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return PublishMsg(j, exchange, routingKey, "application/json")
}

// Publish plain text message with confirmation
func PublishText(msg string, exchange string, routingKey string) error {
	return PublishMsg([]byte(msg), exchange, routingKey, "text/plain")
}

/*
	Publish message with confirmation
*/
func PublishMsg(msg []byte, exchange string, routingKey string, contentType string) error {
	pubChanRwm.RLock()
	defer pubChanRwm.RUnlock()
	pubWg.Add(1)
	defer pubWg.Done()

	if pubChan == nil || pubChan.IsClosed() {
		return errPubChanClosed
	}

	publishing := amqp.Publishing{
		ContentType:  contentType,
		DeliveryMode: amqp.Persistent,
		Body:         msg,
	}
	confirm, err := pubChan.PublishWithDeferredConfirmWithContext(context.Background(), exchange, routingKey, false, false, publishing)
	if err != nil {
		return err
	}

	if !confirm.Wait() {
		return errMsgNotPublished
	}

	return nil
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

		"rabbitmq.declaration.queue"
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

		"rabbitmq.declaration.queue"
		"rabbitmq.declaration.binding." + queueName + ".key"
		"rabbitmq.declaration.binding." + queueName + ".exchange"
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

		"rabbitmq.declaration.binding" + "." + queueName + ".key"
*/
func bindRoutingKeyProp(queue string) (propKey string) {
	propKey = common.PROP_RABBITMQ_DEC_BINDING + "." + queue + ".key"
	return
}

/*
	Get prop key for exchange name of queue

		"rabbitmq.declaration.binding." + queueName + ".exchange"
*/
func bindExchangeProp(queue string) (propKey string) {
	propKey = common.PROP_RABBITMQ_DEC_BINDING + "." + queue + ".exchange"
	return
}

/*
	Declare exchanges

	It looks for PROP:

		"rabbitmq.declaration.exchange"
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
	Start RabbitMQ Client (synchronous for the first time, then auto-reconnect later in another goroutine)

	This func will attempt to establish connection to broker, declare queues, exchanges and bindings.

	Listeners are also created once the intial setup is done.

	When connection is lost, it will attmpt to reconnect to recover, unless the given context is done.

	To register listener, please use 'AddListener' func.

*/
func StartRabbitMqClient(ctx context.Context) error {
	notifyCloseChan, err := initClient(ctx)
	if err != nil {
		return err
	}

	go func(notifyCloseChan chan *amqp.Error) {
		isInitial := true

		for {
			if isInitial {
				isInitial = false
			} else {
				notifyCloseChan, err = initClient(ctx)
				if err != nil {
					logrus.Infof("Error connecting to RabbitMQ: %v", err)
					time.Sleep(time.Second * 5)
					continue
				}
			}

			select {
			// block until connection is closed, then reconnect, thus continue
			case <-notifyCloseChan:
				continue
			// context is done, close the connection, and exit
			case <-ctx.Done():
				logrus.Info("Server context done, trying to close RabbitMQ connection")
				if err := ClientDisconnect(); err != nil {
					logrus.Warnf("Failed to close connection to RabbitMQ: %v", err)
				}
				return
			}
		}
	}(notifyCloseChan)

	return nil
}

// Disconnect from RabbitMQ server
func ClientDisconnect() error {
	mu.Lock()
	defer mu.Unlock()
	if _conn == nil {
		return nil
	}

	logrus.Info("Disconnecting RabbitMQ Connection")
	pubWg.Wait()
	return _conn.Close()
}

// Try to establish Connection
func tryEstablishConn() (*amqp.Connection, error) {
	if _conn != nil && !_conn.IsClosed() {
		return _conn, nil
	}

	logrus.Info("Establish connection to RabbitMQ")
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
	_conn = cn
	return _conn, nil
}

// Declare Queus, Exchanges, and Bindings
func declareComponents(ch *amqp.Channel) error {
	if e := declareQueues(ch); e != nil {
		return e
	}
	if e := declareExchanges(ch); e != nil {
		return e
	}
	if e := declareBindings(ch); e != nil {
		return e
	}
	return nil
}

/*
	Init RabbitMQ Client

	return notifyCloseChannel for connection and error
*/
func initClient(ctx context.Context) (chan *amqp.Error, error) {
	mu.Lock()
	defer mu.Unlock()

	// Establish connection if necessary
	conn, err := tryEstablishConn()
	if err != nil {
		return nil, err
	}

	notifyCloseChan := make(chan *amqp.Error)
	conn.NotifyClose(notifyCloseChan)

	logrus.Infof("Creating Channel to RabbitMQ")
	ch, e := conn.Channel()
	if e != nil {
		return nil, e
	}

	// queues, exchanges, bindings
	e = declareComponents(ch)
	if e != nil {
		return nil, e
	}
	ch.Close()

	// consumers
	if e = bootstrapConsumers(conn); e != nil {
		logrus.Errorf("Failed to bootstrap consumer: %v", e)
	}

	// publisher
	if e = bootstrapPublisher(conn); e != nil {
		logrus.Errorf("Failed to bootstrap publisher: %v", e)
	}

	logrus.Info("RabbitMQ client initialization finished")
	return notifyCloseChan, nil
}

func bootstrapConsumers(conn *amqp.Connection) error {
	qos := common.GetPropInt(common.PROP_RABBITMQ_CONSUMER_QOS)
	parallism := common.GetPropInt(common.PROP_RABBITMQ_CONSUMER_PARALLISM)
	if parallism < 1 {
		parallism = 1
	}
	logrus.Infof("RabbitMQ consumer parallism: %d", parallism)

	for _, v := range msgListeners {
		listener := v

		ch, e := conn.Channel()
		if e != nil {
			return e
		}

		e = ch.Qos(qos, 0, false)
		if e != nil {
			return e
		}

		msgs, err := ch.Consume(listener.QueueName, "", false, false, false, false, nil)
		if err != nil {
			logrus.Errorf("Failed to listen to '%s', err: %v", listener.QueueName, err)
		}

		maxRetry := common.GetPropInt(common.PROP_RABBITMQ_CONSUMER_RETRY)
		for i := 0; i < parallism; i++ {
			ic := i
			startListening(msgs, listener, ic, maxRetry)
		}
	}

	logrus.Info("RabbitMQ consumer initialization finished")
	return nil
}

func startListening(msgs <-chan amqp.Delivery, listener MsgListener, routineNo int, maxRetry int) {
	go func() {
		logrus.Infof("[R%d] %v started", routineNo, listener)
		for msg := range msgs {
			retry := maxRetry
			payload := string(msg.Body)

			for {
				e := listener.Handler(payload)
				if e == nil {
					msg.Ack(false)
					break
				}
				logrus.Errorf("Failed to handle message for queue: '%s', payload: '%v', err: '%v' (retry: %d)", listener.QueueName, payload, e, retry)

				// last retry
				if retry == 0 {
					msg.Ack(false)
					break
				}

				// disable retry, simply nack it
				if retry < 0 {
					msg.Nack(false, true)
					break
				}

				retry -= 1
				time.Sleep(time.Millisecond * 500) // sleep 500ms for every retry
			}
		}
		logrus.Infof("[R%d] %v stopped", routineNo, listener)
	}()
}

func bootstrapPublisher(conn *amqp.Connection) error {
	pubChanRwm.Lock()
	defer pubChanRwm.Unlock()

	if pubChan != nil && !pubChan.IsClosed() {
		logrus.Info("RabbitMQ publisher is already initialized")
		return nil
	}

	pc, err := conn.Channel()
	if err != nil {
		return err
	}
	if err = pc.Confirm(false); err != nil {
		return fmt.Errorf("publishing channel could not be put into confirm mode: %s", err)
	}
	pubChan = pc
	logrus.Info("RabbitMQ publisher initialization finished")
	return nil
}
