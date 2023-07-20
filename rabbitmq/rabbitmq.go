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
	DEFAULT_QOS = 68 // default QOS
)

var (
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
	Name    string
	Kind    string
	Durable bool
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
func PublishJson(c common.ExecContext, obj any, exchange string, routingKey string) error {
	j, err := json.Marshal(obj)
	if err != nil {
		return common.TraceErrf(err, "failed to marshal message body")
	}
	return PublishMsg(c, j, exchange, routingKey, "application/json")
}

// Publish plain text message with confirmation
func PublishText(c common.ExecContext, msg string, exchange string, routingKey string) error {
	return PublishMsg(c, []byte(msg), exchange, routingKey, "text/plain")
}

// Publish message with confirmation
func PublishMsg(c common.ExecContext, msg []byte, exchange string, routingKey string, contentType string) error {
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
	}
	confirm, err := pc.PublishWithDeferredConfirmWithContext(context.Background(), exchange, routingKey, false, false, publishing)
	if err != nil {
		return common.TraceErrf(err, "failed to publish message")
	}

	if !confirm.Wait() {
		return errMsgNotPublished
	}

	c.Log.Debugf("Published MQ to %v, %s", exchange, msg)
	return nil
}

/*
Add message Listener

Listeners will be registered in StartRabbitMqClient func when the connection to broker is established.
*/
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

		e := ch.ExchangeDeclare(exchange.Name, exchange.Kind, exchange.Durable, false, false, false, nil)
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
	_mutex.Lock()
	defer _mutex.Unlock()

	// Establish connection if necessary
	conn, err := tryEstablishConn()
	if err != nil {
		return nil, err
	}

	notifyCloseChan := make(chan *amqp.Error)
	conn.NotifyClose(notifyCloseChan)

	logrus.Debugf("Creating Channel to RabbitMQ")
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

	// // publisher
	// if e = bootstrapPublisher(conn); e != nil {
	// 	return nil, common.TraceErrf(e, "failed to create publisher")
	// }

	// consumers
	if e = bootstrapConsumers(conn); e != nil {
		logrus.Errorf("Failed to bootstrap consumer: %v", e)
		return nil, common.TraceErrf(e, "failed to create consumer")
	}

	logrus.Debugf("RabbitMQ client initialization finished")
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
		logrus.Debugf("%d-%v started", routineNo, listener)
		for msg := range msgCh {
			payload := string(msg.Body)

			e := listener.Handle(payload)
			if e == nil {
				msg.Ack(false)
				continue
			}

			logrus.Errorf("Failed to handle message for queue: '%s', payload: '%v', err: '%v'", listener.Queue(), payload, e)
			msg.Nack(false, true)
		}
		logrus.Debugf("%d-%v stopped", routineNo, listener)
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
