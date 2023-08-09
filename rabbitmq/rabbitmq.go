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
	DEFAULT_QOS     = 68   // default QOS
	redeliverDelay  = 5000 // redeliver delay, changing it will also create a new queue for redelivery
	defaultExchange = ""   // default exchange that routes based on queue name using routing key
)

var (
	redeliverQueueMap sync.Map

	_mutex sync.Mutex       // global mutex for everything
	_conn  *amqp.Connection // connection pointer, accessing it require obtaining 'mu' lock
	_pubWg sync.WaitGroup   // number of messages being published

	// pool of channel for publishing message
	_pubChanPool sync.Pool = sync.Pool{
		New: func() any {
			c, err := newPubChan()
			if err != nil {
				logrus.Errorf("Failed to create new publishing channel, %v", err)
				return nil
			}

			// attach a finalizer to it since sync.Pool may GC it at any time
			runtime.SetFinalizer(c, func(c *amqp.Channel) {
				if !c.IsClosed() {
					logrus.Debugf("Garbage collecting *amqp.Channel from pool, closing channel")
					c.Close()
				}
			})

			return c
		},
	}

	_listeners            []Listener
	_bindingRegistration  []BindingRegistration
	_queueRegistration    []QueueRegistration
	_exchangeRegistration []ExchangeRegistration

	errMissingChannel  = errors.New("channel is missing")
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
}

type BindingRegistration struct {
	Queue      string
	RoutingKey string
	Exchange   string
}

type QueueRegistration struct {
	Name    string
	Durable bool
}

type ExchangeRegistration struct {
	Name       string
	Kind       string
	Durable    bool
	Properties map[string]any
}

/* Is RabbitMQ Enabled */
func IsEnabled() bool {
	return common.GetPropBool(common.PROP_RABBITMQ_ENABLED)
}

// Listener of Queue
type Listener interface {
	Queue() string               // return name of the queue
	Handle(payload string) error // handle message
	Concurrency() int
}

// Json Message Listener for Queue
type JsonMsgListener[T any] struct {
	QueueName     string
	Handler       func(payload T) error
	NumOfRoutines int
}

func (m JsonMsgListener[T]) Queue() string {
	return m.QueueName
}

func (m JsonMsgListener[T]) Handle(payload string) error {
	var t T
	if e := json.Unmarshal([]byte(payload), &t); e != nil {
		return e
	}
	return m.Handler(t)
}

func (m JsonMsgListener[T]) Concurrency() int {
	return m.NumOfRoutines
}

func (m JsonMsgListener[T]) String() string {
	var funcName string = "nil"
	if m.Handler != nil {
		funcName = runtime.FuncForPC(reflect.ValueOf(m.Handler).Pointer()).Name()
	}
	return fmt.Sprintf("Listener: '%s' --> '%s'", funcName, m.QueueName)
}

// Message Listener for Queue
type MsgListener struct {
	QueueName     string
	Handler       func(payload string) error
	NumOfRoutines int
}

func (m MsgListener) Queue() string {
	return m.QueueName
}

func (m MsgListener) Handle(payload string) error {
	return m.Handler(payload)
}

func (m MsgListener) Concurrency() int {
	return m.NumOfRoutines
}

func (m MsgListener) String() string {
	var funcName string = "nil"
	if m.Handler != nil {
		funcName = runtime.FuncForPC(reflect.ValueOf(m.Handler).Pointer()).Name()
	}
	return fmt.Sprintf("MsgListener{ QueueName: '%s', Handler: %s }", m.QueueName, funcName)
}

// Publish json message with confirmation
func PublishJson(c common.Rail, obj any, exchange string, routingKey string) error {
	j, err := json.Marshal(obj)
	if err != nil {
		return common.TraceErrf(err, "failed to marshal message body")
	}
	return PublishMsg(c, j, exchange, routingKey, "application/json", nil)
}

// Publish plain text message with confirmation
func PublishText(c common.Rail, msg string, exchange string, routingKey string) error {
	return PublishMsg(c, []byte(msg), exchange, routingKey, "text/plain", nil)
}

// Publish message with confirmation
func PublishMsg(c common.Rail, msg []byte, exchange string, routingKey string, contentType string, headers map[string]interface{}) error {
	_pubWg.Add(1)
	defer _pubWg.Done()

	pc, err := borrowPubChan()
	if err != nil {
		return common.TraceErrf(err, "failed to obtain channel is closed, unable to publish message")
	}
	defer returnPubChan(pc)

	publishing := amqp.Publishing{
		ContentType:  contentType,
		DeliveryMode: amqp.Persistent,
		Body:         msg,
		Headers:      headers,
		MessageId:    common.GenIdP("mq_"),
	}
	confirm, err := pc.PublishWithDeferredConfirmWithContext(context.Background(), exchange, routingKey, false, false, publishing)
	if err != nil {
		return common.TraceErrf(err, "failed to publish message")
	}

	if !confirm.Wait() {
		return errMsgNotPublished
	}

	c.Debugf("Published MQ to exchange '%v', '%s'", exchange, msg)
	return nil
}

// Register pending message listener.
//
// Listeners will be started in StartRabbitMqClient func when the connection to broker is established.
//
// For any message that the listener is unable to process (returning error), the message is redelivered indefinitively
// with a delay of 10 seconds until the message is finally processed without error.
func AddListener(listener Listener) {
	_mutex.Lock()
	defer _mutex.Unlock()
	_listeners = append(_listeners, listener)
}

/*
Declare durable queues
*/
func declareQueues(ch *amqp.Channel) error {
	if ch == nil {
		return errMissingChannel
	}

	for _, queue := range _queueRegistration {
		dqueue, e := ch.QueueDeclare(queue.Name, queue.Durable, false, false, false, nil)
		if e != nil {
			return common.TraceErrf(e, "failed to declare queue, %v", queue.Name)
		}
		logrus.Debugf("Declared queue '%s'", dqueue.Name)
	}
	return nil
}

func redeliverQueue(exchange string, routingKey string) string {
	return fmt.Sprintf("redeliver_%v_%v_%v", exchange, routingKey, redeliverDelay)
}

// Declare binding on client initialization
func RegisterBinding(b BindingRegistration) {
	_bindingRegistration = append(_bindingRegistration, b)
}

// Declare queue on client initialization
func RegisterQueue(q QueueRegistration) {
	_queueRegistration = append(_queueRegistration, q)
}

// Declare exchange on client initialization
func RegisterExchange(e ExchangeRegistration) {
	_exchangeRegistration = append(_exchangeRegistration, e)
}

/*
Declare bindings
*/
func declareBindings(ch *amqp.Channel) error {
	if ch == nil {
		return errMissingChannel
	}

	for _, bind := range _bindingRegistration {
		if bind.RoutingKey == "" {
			bind.RoutingKey = "#"
		}
		e := ch.QueueBind(bind.Queue, bind.RoutingKey, bind.Exchange, false, nil)
		if e != nil {
			return common.TraceErrf(e, "failed to declare binding, queue: %v, routingkey: %v, exchange: %v", bind.Queue, bind.RoutingKey, bind.Exchange)
		}
		logrus.Debugf("Declared binding for queue '%s' to exchange '%s' using routingKey '%s'", bind.Queue, bind.Exchange, bind.RoutingKey)

		// declare a redeliver queue, this queue will not have any subscriber, once the messages are expired, they are
		// routed to the original queue
		//
		// for this to work, we have to know what the exchange and routing key is, we have to write this here
		//
		// 	src: https://ivanyu.me/blog/2015/02/16/delayed-message-delivery-in-rabbitmq/
		rqueue := redeliverQueue(bind.Exchange, bind.RoutingKey)
		rq, e := ch.QueueDeclare(rqueue, true, false, false, false, amqp.Table{
			"x-message-ttl":             redeliverDelay,
			"x-dead-letter-exchange":    bind.Exchange,
			"x-dead-letter-routing-key": bind.RoutingKey,
		})
		if e != nil {
			return common.TraceErrf(e, "failed to declare redeliver queue '%v' for '%v'", rq, bind.Queue)
		}
		redeliverQueueMap.Store(rqueue, true) // remember this redeliver queue
		logrus.Debugf("Declared redeliver queue '%s' for '%v'", rq.Name, bind.Queue)
	}
	return nil
}

/*
Declare exchanges
*/
func declareExchanges(ch *amqp.Channel) error {
	if ch == nil {
		return errMissingChannel
	}

	for _, exchange := range _exchangeRegistration {
		if exchange.Kind == "" {
			exchange.Kind = "direct"
		}

		e := ch.ExchangeDeclare(exchange.Name, exchange.Kind, exchange.Durable, false, false, false, exchange.Properties)
		if e != nil {
			return common.TraceErrf(e, "failed to declare exchange, %v", exchange.Name)
		}
		logrus.Debugf("Declared %s exchange '%s'", exchange.Kind, exchange.Name)
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
					logrus.Errorf("Error connecting to RabbitMQ: %v", err)
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
	_mutex.Lock()
	defer _mutex.Unlock()
	if _conn == nil {
		return nil
	}

	logrus.Info("Disconnecting RabbitMQ Connection")
	_pubWg.Wait()
	err := _conn.Close()
	_conn = nil
	return err
}

// Try to establish Connection
func tryEstablishConn() (*amqp.Connection, error) {
	if _conn != nil && !_conn.IsClosed() {
		return _conn, nil
	}

	c := amqp.Config{}
	username := common.GetPropStr(common.PROP_RABBITMQ_USERNAME)
	password := common.GetPropStr(common.PROP_RABBITMQ_PASSWORD)
	vhost := common.GetPropStr(common.PROP_RABBITMQ_VHOST)
	host := common.GetPropStr(common.PROP_RABBITMQ_HOST)
	port := common.GetPropInt(common.PROP_RABBITMQ_PORT)
	dialUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", username, password, host, port, vhost)

	logrus.Infof("Establish connection to RabbitMQ: '%s@%s:%d/%s'", username, host, port, vhost)
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
	rail := common.NewRail(ctx)
	_mutex.Lock()
	defer _mutex.Unlock()

	// Establish connection if necessary
	conn, err := tryEstablishConn()
	if err != nil {
		return nil, err
	}

	notifyCloseChan := make(chan *amqp.Error)
	conn.NotifyClose(notifyCloseChan)

	rail.Debugf("Creating Channel to RabbitMQ")
	ch, e := conn.Channel()
	if e != nil {
		return nil, common.TraceErrf(e, "failed to create channel")
	}

	// queues, exchanges, bindings
	e = declareComponents(ch)
	if e != nil {
		return nil, e
	}
	ch.Close()

	// consumers
	if e = bootstrapConsumers(conn); e != nil {
		rail.Errorf("Failed to bootstrap consumer: %v", e)
		return nil, common.TraceErrf(e, "failed to create consumer")
	}

	rail.Debugf("RabbitMQ client initialization finished")
	return notifyCloseChan, nil
}

func bootstrapConsumers(conn *amqp.Connection) error {
	qos := common.GetPropInt(common.PROP_RABBITMQ_CONSUMER_QOS)

	for _, v := range _listeners {
		listener := v

		qname := listener.Queue()
		concurrency := v.Concurrency()
		if concurrency < 1 {
			concurrency = 1
		}

		for i := 0; i < concurrency; i++ {
			ic := i
			ch, e := conn.Channel()
			if e != nil {
				return e
			}

			e = ch.Qos(qos, 0, false)
			if e != nil {
				return e
			}
			msgCh, err := ch.Consume(qname, "", false, false, false, false, nil)
			if err != nil {
				logrus.Errorf("Failed to listen to '%s', err: %v", qname, err)
			}
			startListening(msgCh, listener, ic)
		}
	}

	logrus.Debug("RabbitMQ consumer initialization finished")
	return nil
}

func startListening(msgCh <-chan amqp.Delivery, listener Listener, routineNo int) {
	go func() {
		rail := common.EmptyRail()

		rail.Debugf("%d-%v started", routineNo, listener)
		for msg := range msgCh {
			payload := string(msg.Body)

			e := listener.Handle(payload)
			if e == nil {
				msg.Ack(false)
				continue
			}

			rail.Errorf("Failed to handle message for queue: '%s', exchange: '%s', routingKey: '%s', payload: '%v', err: '%v', messageId: '%v'",
				listener.Queue(), msg.Exchange, msg.RoutingKey, payload, e, msg.MessageId)

			// before we nack the message, we check if it's possible to put the message in a persudo 'delayed' queue
			//
			// rabbitmq doesn't provide mechanism to configure the maximum time of redelivery for each message
			// we don't want the server to be flooded with messages that we cannot handle for now for whatever reason
			// if the message is redelivered, we send the same message to the broker with delay, and we simply ack the current message
			//
			// this implementation doesn't use x-delay plugin at all, installing a plugin for this is very incovenient, the plugin
			// may not be available as well.
			//
			// notice that when we create binding for queue and exchange, we also create a redeliver queue to mimic the 'delay' behaviour
			//
			// 	src: https://ivanyu.me/blog/2015/02/16/delayed-message-delivery-in-rabbitmq/
			rq := redeliverQueue(msg.Exchange, msg.RoutingKey)
			if _, ok := redeliverQueueMap.Load(rq); ok { // have to make sure we indeed have a redeliver queue for this message
				err := PublishMsg(rail, msg.Body, defaultExchange, rq, msg.ContentType, nil)
				if err == nil {
					msg.Ack(false)
					rail.Debugf("Sent message to delayed redeliver queue: %v", msg.MessageId)
					continue
				}
				rail.Debugf("Failed to send message to delayed redeliver queue: %v, %v", msg.MessageId, err)
			}

			// nack the message
			msg.Nack(false, true)
			rail.Debugf("Nacked message: %v", msg.MessageId)
		}
		rail.Debugf("%d-%v stopped", routineNo, listener)
	}()
}

// borrow a publishing channel from the pool.
func borrowPubChan() (*amqp.Channel, error) {
	ch := _pubChanPool.Get()
	if ch == nil {
		return nil, common.NewTraceErrf("could not create new RabbitMQ channel")
	}

	ach := ch.(*amqp.Channel)
	if ach.IsClosed() { // ach for whatever reason is now closed, we just dump it, create a new one manually
		ch = _pubChanPool.New() // force to create a new one
		ach = ch.(*amqp.Channel)

		if ach == nil { // still unable to create a new channel, the connection may be broken
			return nil, common.NewTraceErrf("could not create new RabbitMQ channel")
		}
	}
	return ach, nil
}

// return a publishing channel back to the pool.
//
// if the channel is closed already, it's simply ignored.
func returnPubChan(ch *amqp.Channel) {
	if ch.IsClosed() {
		return // for some errors, the channel may be closed, we simply dump it
	}
	_pubChanPool.Put(ch) // put it back to the pool
}

// open a new publishing channel.
//
// return error if it failed to open new channel, e.g., connection is also missing or closed.
func newPubChan() (*amqp.Channel, error) {
	_mutex.Lock()
	defer _mutex.Unlock()

	if _conn == nil {
		return nil, common.NewTraceErrf("rabbitmq connection is missing")
	}

	newChan, err := _conn.Channel()
	if err != nil {
		return nil, common.TraceErrf(err, "could not create new RabbitMQ channel")
	}

	if err = newChan.Confirm(false); err != nil {
		return nil, common.TraceErrf(err, "channel could not be put into confirm mode")
	}

	logrus.Debugf("Created new RabbitMQ publishing channel")

	return newChan, nil
}
