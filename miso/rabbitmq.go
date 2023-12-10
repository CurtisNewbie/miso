package miso

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
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
				Errorf("Failed to create new publishing channel, %v", err)
				return nil
			}

			// attach a finalizer to it since sync.Pool may GC it at any time
			runtime.SetFinalizer(c, amqpChannelFinalizer)

			return c
		},
	}

	_listeners            []RabbitListener
	_bindingRegistration  []BindingRegistration
	_queueRegistration    []QueueRegistration
	_exchangeRegistration []ExchangeRegistration

	errMissingChannel  = errors.New("channel is missing")
	errMsgNotPublished = errors.New("message not published, server failed to confirm")
)

func init() {
	SetDefProp(PropRabbitMqEnabled, false)
	SetDefProp(PropRabbitMqHost, "localhost")
	SetDefProp(PropRabbitMqPort, 5672)
	SetDefProp(PropRabbitMqUsername, "")
	SetDefProp(PropRabbitMqPassword, "")
	SetDefProp(PropRabbitMqVhost, "")
	SetDefProp(PropRabbitMqConsumerQos, DEFAULT_QOS)

	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap RabbitMQ",
		Bootstrap: RabbitBootstrap,
		Condition: RabbitBootstrapCondition,
		Order:     10,
	})
}

func amqpChannelFinalizer(c *amqp.Channel) {
	if !c.IsClosed() {
		Debugf("Garbage collecting *amqp.Channel from pool, closing channel")
		c.Close()
	}
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
func RabbitMQEnabled() bool {
	return GetPropBool(PropRabbitMqEnabled)
}

// RabbitListener of Queue
type RabbitListener interface {
	Queue() string                          // return name of the queue
	Handle(rail Rail, payload string) error // handle message
	Concurrency() int
}

// Json Message Listener for Queue
type JsonMsgListener[T any] struct {
	QueueName     string
	Handler       func(rail Rail, payload T) error
	NumOfRoutines int
}

func (m JsonMsgListener[T]) Queue() string {
	return m.QueueName
}

func (m JsonMsgListener[T]) Handle(rail Rail, payload string) error {
	var t T
	if e := ParseJson([]byte(payload), &t); e != nil {
		return e
	}
	return m.Handler(rail, t)
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
	Handler       func(rail Rail, payload string) error
	NumOfRoutines int
}

func (m MsgListener) Queue() string {
	return m.QueueName
}

func (m MsgListener) Handle(rail Rail, payload string) error {
	return m.Handler(rail, payload)
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
func PublishJson(c Rail, obj any, exchange string, routingKey string) error {
	j, err := WriteJson(obj)
	if err != nil {
		return TraceErrf(err, "failed to marshal message body")
	}
	return PublishMsg(c, j, exchange, routingKey, "application/json", nil)
}

// Publish plain text message with confirmation
func PublishText(c Rail, msg string, exchange string, routingKey string) error {
	return PublishMsg(c, []byte(msg), exchange, routingKey, "text/plain", nil)
}

// Publish message with confirmation
func PublishMsg(c Rail, msg []byte, exchange string, routingKey string, contentType string, headers map[string]any) error {
	_pubWg.Add(1)
	defer _pubWg.Done()

	pc, err := borrowPubChan()
	if err != nil {
		return TraceErrf(err, "failed to obtain channel is closed, unable to publish message")
	}
	defer returnPubChan(pc)

	// propogate trace through headers
	if headers == nil {
		headers = map[string]any{}
		propagated := GetPropagationKeys()
		for i := range propagated {
			k := propagated[i]
			headers[k] = c.CtxValue(k)
		}
	}

	publishing := amqp.Publishing{
		ContentType:  contentType,
		DeliveryMode: amqp.Persistent,
		Body:         msg,
		Headers:      headers,
		MessageId:    GenIdP("mq_"),
	}
	confirm, err := pc.PublishWithDeferredConfirmWithContext(context.Background(), exchange, routingKey, false, false, publishing)
	if err != nil {
		return TraceErrf(err, "failed to publish message")
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
func AddRabbitListener(listener RabbitListener) {
	_mutex.Lock()
	defer _mutex.Unlock()
	_listeners = append(_listeners, listener)
}

// Declare queue using the provided channel immediately
func DeclareRabbitQueue(ch *amqp.Channel, queue QueueRegistration) error {
	dqueue, e := ch.QueueDeclare(queue.Name, queue.Durable, false, false, false, nil)
	if e != nil {
		return TraceErrf(e, "failed to declare queue, %v", queue.Name)
	}
	Debugf("Declared queue '%s'", dqueue.Name)
	return nil
}

func redeliverQueue(exchange string, routingKey string) string {
	return fmt.Sprintf("redeliver_%v_%v_%v", exchange, routingKey, redeliverDelay)
}

// Declare binding on client initialization
func RegisterRabbitBinding(b BindingRegistration) {
	_bindingRegistration = append(_bindingRegistration, b)
}

// Declare queue on client initialization
func RegisterRabbitQueue(q QueueRegistration) {
	_queueRegistration = append(_queueRegistration, q)
}

// Declare exchange on client initialization
func RegisterRabbitExchange(e ExchangeRegistration) {
	_exchangeRegistration = append(_exchangeRegistration, e)
}

// Declare binding using the provided channel immediately
func DeclareRabbitBinding(ch *amqp.Channel, bind BindingRegistration) error {
	if bind.RoutingKey == "" {
		bind.RoutingKey = "#"
	}
	e := ch.QueueBind(bind.Queue, bind.RoutingKey, bind.Exchange, false, nil)
	if e != nil {
		return TraceErrf(e, "failed to declare binding, queue: %v, routingkey: %v, exchange: %v", bind.Queue, bind.RoutingKey, bind.Exchange)
	}
	Debugf("Declared binding for queue '%s' to exchange '%s' using routingKey '%s'", bind.Queue, bind.Exchange, bind.RoutingKey)

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
		return TraceErrf(e, "failed to declare redeliver queue '%v' for '%v'", rq, bind.Queue)
	}
	redeliverQueueMap.Store(rqueue, true) // remember this redeliver queue
	Debugf("Declared redeliver queue '%s' for '%v'", rq.Name, bind.Queue)
	return nil
}

// Declare exchange using the provided channel immediately
func DeclareRabbitExchange(ch *amqp.Channel, exchange ExchangeRegistration) error {
	if exchange.Kind == "" {
		exchange.Kind = "direct"
	}

	e := ch.ExchangeDeclare(exchange.Name, exchange.Kind, exchange.Durable, false, false, false, exchange.Properties)
	if e != nil {
		return TraceErrf(e, "failed to declare exchange, %v", exchange.Name)
	}
	Debugf("Declared %s exchange '%s'", exchange.Kind, exchange.Name)
	return nil
}

/*
Start RabbitMQ Client (synchronous for the first time, then auto-reconnect later in another goroutine)

This func will attempt to establish connection to broker, declare queues, exchanges and bindings.

Listeners are also created once the intial setup is done.

When connection is lost, it will attmpt to reconnect to recover, unless the given context is done.

To register listener, please use 'AddListener' func.
*/
func StartRabbitMqClient(rail Rail) error {
	notifyCloseChan, err := initRabbitClient(rail)
	if err != nil {
		return err
	}

	go func(notifyCloseChan chan *amqp.Error) {
		isInitial := true
		doneCh := rail.Ctx.Done()

		for {
			if isInitial {
				isInitial = false
			} else {
				notifyCloseChan, err = initRabbitClient(rail)
				if err != nil {
					Errorf("Error connecting to RabbitMQ: %v", err)
					time.Sleep(time.Second * 5)
					continue
				}
			}

			select {
			// block until connection is closed, then reconnect, thus continue
			case <-notifyCloseChan:
				continue
			// context is done, close the connection, and exit
			case <-doneCh:
				if err := RabbitDisconnect(rail); err != nil {
					Warnf("Failed to close connection to RabbitMQ: %v", err)
				}
				return
			}
		}
	}(notifyCloseChan)

	return nil
}

// Disconnect from RabbitMQ server
func RabbitDisconnect(rail Rail) error {
	_mutex.Lock()
	defer _mutex.Unlock()
	if _conn == nil {
		return nil
	}

	rail.Info("Disconnecting RabbitMQ Connection")
	_pubWg.Wait()
	err := _conn.Close()
	_conn = nil
	return err
}

// Try to establish Connection
func tryConnRabbit(rail Rail) (*amqp.Connection, error) {
	if _conn != nil && !_conn.IsClosed() {
		return _conn, nil
	}

	c := amqp.Config{}
	username := GetPropStr(PropRabbitMqUsername)
	password := GetPropStr(PropRabbitMqPassword)
	vhost := GetPropStr(PropRabbitMqVhost)
	host := GetPropStr(PropRabbitMqHost)
	port := GetPropInt(PropRabbitMqPort)
	dialUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", username, password, host, port, vhost)

	rail.Infof("Establish connection to RabbitMQ: '%s@%s:%d/%s'", username, host, port, vhost)
	cn, e := amqp.DialConfig(dialUrl, c)
	if e != nil {
		return nil, e
	}
	_conn = cn
	return _conn, nil
}

// Declare Queus, Exchanges, and Bindings
func decRabbitComp(ch *amqp.Channel) error {
	if ch == nil {
		return errMissingChannel
	}

	for _, queue := range _queueRegistration {
		if err := DeclareRabbitQueue(ch, queue); err != nil {
			return err
		}
	}

	for _, exchange := range _exchangeRegistration {
		if err := DeclareRabbitExchange(ch, exchange); err != nil {
			return err
		}
	}

	for _, bind := range _bindingRegistration {
		if err := DeclareRabbitBinding(ch, bind); err != nil {
			return err
		}
	}

	return nil
}

// Create new channel from the established connection
func NewRabbitChan() (*amqp.Channel, error) {
	return newChan()
}

// Check if connection exists
func RabbitConnected() bool {
	_mutex.Lock()
	defer _mutex.Unlock()
	return _conn != nil
}

/*
Init RabbitMQ Client

return notifyCloseChannel for connection and error
*/
func initRabbitClient(rail Rail) (chan *amqp.Error, error) {
	_mutex.Lock()
	defer _mutex.Unlock()

	// Establish connection if necessary
	conn, err := tryConnRabbit(rail)
	if err != nil {
		return nil, err
	}

	notifyCloseChan := make(chan *amqp.Error)
	conn.NotifyClose(notifyCloseChan)

	rail.Debugf("Creating Channel to RabbitMQ")
	ch, e := conn.Channel()
	if e != nil {
		return nil, TraceErrf(e, "failed to create channel")
	}

	// queues, exchanges, bindings
	e = decRabbitComp(ch)
	if e != nil {
		return nil, e
	}
	ch.Close()

	// consumers
	if e = startRabbitConsumers(conn); e != nil {
		rail.Errorf("Failed to bootstrap consumer: %v", e)
		return nil, TraceErrf(e, "failed to create consumer")
	}

	rail.Debugf("RabbitMQ client initialization finished")
	return notifyCloseChan, nil
}

func startRabbitConsumers(conn *amqp.Connection) error {
	qos := GetPropInt(PropRabbitMqConsumerQos)

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
				Errorf("Failed to listen to '%s', err: %v", qname, err)
			}
			startListening(msgCh, listener, ic)
		}
	}

	Debug("RabbitMQ consumer initialization finished")
	return nil
}

func startListening(msgCh <-chan amqp.Delivery, listener RabbitListener, routineNo int) {
	go func() {
		Debugf("%d-%v started", routineNo, listener)
		for msg := range msgCh {

			// read trace from headers
			rail := EmptyRail()
			if msg.Headers != nil {
				keys := GetPropagationKeys()
				for i := range keys {
					k := keys[i]
					if hv, ok := msg.Headers[k]; ok {
						rail = rail.WithCtxVal(k, fmt.Sprintf("%v", hv))
					}
				}
			}

			// message body, we only support text
			payload := string(msg.Body)

			// listener handling the message payload
			e := listener.Handle(rail, payload)
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
		Debugf("%d-%v stopped", routineNo, listener)
	}()
}

// borrow a publishing channel from the pool.
func borrowPubChan() (*amqp.Channel, error) {
	ch := _pubChanPool.Get()
	if ch == nil {
		return nil, NewTraceErrf("could not create new RabbitMQ channel")
	}

	ach := ch.(*amqp.Channel)
	if ach.IsClosed() { // ach for whatever reason is now closed, we just dump it, create a new one manually
		ch = _pubChanPool.New() // force to create a new one
		ach = ch.(*amqp.Channel)

		if ach == nil { // still unable to create a new channel, the connection may be broken
			return nil, NewTraceErrf("could not create new RabbitMQ channel")
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
	newChan, err := newChan()
	if err != nil {
		return nil, TraceErrf(err, "could not create new RabbitMQ channel")
	}

	if err = newChan.Confirm(false); err != nil {
		return nil, TraceErrf(err, "channel could not be put into confirm mode")
	}

	Debugf("Created new RabbitMQ publishing channel")

	return newChan, nil
}

func newChan() (*amqp.Channel, error) {
	_mutex.Lock()
	defer _mutex.Unlock()

	if _conn == nil {
		return nil, NewTraceErrf("rabbitmq connection is missing")
	}

	return _conn.Channel()
}

func RabbitBootstrap(rail Rail) error {
	if e := StartRabbitMqClient(rail); e != nil {
		return TraceErrf(e, "Failed to establish connection to RabbitMQ")
	}
	return nil
}

func RabbitBootstrapCondition(rail Rail) (bool, error) {
	return RabbitMQEnabled(), nil
}
