package rabbit

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/encoding/json"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/cast"
)

var (
	_ RabbitListener = (*JsonMsgListener[any])(nil)
	_ RabbitListener = (*MsgListener)(nil)
	_ RabbitListener = (*wrappingListener)(nil)
)

const (
	redeliverDelay  = 5000 // redeliver delay, changing it will also create a new queue for redelivery
	defaultExchange = ""   // default exchange that routes based on queue name using routing key
)

const (
	// Default QOS
	DefaultQos = 68

	// RabbitMQ messages are redelivered every 5 seconds, 180 times is roughly equivalent to 15 minutes retry.
	MaxRetryTimes15Min = 180

	// Header key of rabbitmq message, specify how many times the message can be redelivered.
	//
	// Actual redelivery mechanism is implemented by miso's message listener.
	HeaderRabbitMaxRetry = "miso-rabbitmq-max-retry"

	// Header key of rabbitmq message, specify how many times the message has been redelivered.
	//
	// Actual redelivery mechanism is implemented by miso's message listener.
	HeaderRabbitCurrRetry = "miso-rabbitmq-curr-retry"
)

var (
	errMissingChannel  = errors.New("channel is missing")
	errMsgNotPublished = errors.New("message not published, server failed to confirm")
)

var module = miso.InitAppModuleFunc(newModule)

func init() {
	miso.SetDefProp(PropRabbitMqEnabled, false)
	miso.SetDefProp(PropRabbitMqHost, "localhost")
	miso.SetDefProp(PropRabbitMqPort, 5672)
	miso.SetDefProp(PropRabbitMqUsername, "guest")
	miso.SetDefProp(PropRabbitMqPassword, "guest")
	miso.SetDefProp(PropRabbitMqVhost, "")
	miso.SetDefProp(PropRabbitMqConsumerQos, DefaultQos)

	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Bootstrap RabbitMQ",
		Bootstrap: rabbitBootstrap,
		Condition: rabbitBootstrapCondition,
		Order:     miso.BootstrapOrderL4,
	})
}

type rabbitMqModule struct {
	mu                *sync.Mutex
	closed            bool
	redeliverQueueMap *sync.Map
	conn              *amqp.Connection // connection pointer, accessing it require obtaining 'mu' lock
	pubChanPool       *sync.Pool       // pool of channel for publishing message

	listeners            []RabbitListener
	bindingRegistration  []BindingRegistration
	queueRegistration    []QueueRegistration
	exchangeRegistration []ExchangeRegistration
}

func newModule() *rabbitMqModule {
	m := &rabbitMqModule{
		redeliverQueueMap: &sync.Map{},
		mu:                &sync.Mutex{},
	}
	m.pubChanPool = &sync.Pool{
		New: func() any {
			c, err := m.newPubChan()
			if err != nil {
				miso.NoRelayErrorf("Failed to create new publishing channel, %v", err)
				return nil
			}

			// attach a finalizer to it since sync.Pool may GC it at any time
			runtime.SetFinalizer(c, amqpChannelFinalizer)

			return c
		},
	}
	return m
}

func amqpChannelFinalizer(c *amqp.Channel) {
	if !c.IsClosed() {
		miso.Debugf("Garbage collecting *amqp.Channel from pool, closing channel")
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

// RabbitListener of Queue
type RabbitListener interface {
	Queue() string                               // return name of the queue
	Handle(rail miso.Rail, payload string) error // handle message
	Concurrency() int
	QosSpec() int
}

// Json Message Listener for Queue
type JsonMsgListener[T any] struct {
	QueueName     string
	Handler       func(rail miso.Rail, payload T) error
	NumOfRoutines int
	Qos           int
}

func (m JsonMsgListener[T]) Queue() string {
	return m.QueueName
}

func (m JsonMsgListener[T]) Handle(rail miso.Rail, payload string) error {
	var t T
	if e := json.ParseJson(util.UnsafeStr2Byt(payload), &t); e != nil {
		return e
	}
	return m.Handler(rail, t)
}

func (m JsonMsgListener[T]) Concurrency() int {
	return m.NumOfRoutines
}

func (m JsonMsgListener[T]) QosSpec() int {
	return m.Qos
}

// Message Listener for Queue
type MsgListener struct {
	QueueName     string
	Handler       func(rail miso.Rail, payload string) error
	NumOfRoutines int
	Qos           int
}

func (m MsgListener) Queue() string {
	return m.QueueName
}

func (m MsgListener) Handle(rail miso.Rail, payload string) error {
	return m.Handler(rail, payload)
}

func (m MsgListener) Concurrency() int {
	return m.NumOfRoutines
}

func (m MsgListener) QosSpec() int {
	return m.Qos
}

// Publish json message with confirmation
func PublishJson(c miso.Rail, obj any, exchange string, routingKey string) error {
	return module().publishJson(c, obj, exchange, routingKey)
}

// Publish json message with headers and confirmation
func PublishJsonHeaders(c miso.Rail, obj any, exchange string, routingKey string, headers map[string]any) error {
	return module().publishJsonHeaders(c, obj, exchange, routingKey, headers)
}

// Publish plain text message with confirmation
func PublishText(c miso.Rail, msg string, exchange string, routingKey string) error {
	return module().publishText(c, msg, exchange, routingKey)
}

// Publish message with confirmation
func PublishMsg(c miso.Rail, msg []byte, exchange string, routingKey string, contentType string, headers map[string]any) error {
	return module().publishMsg(c, msg, exchange, routingKey, contentType, headers)
}

func (m *rabbitMqModule) publishJson(c miso.Rail, obj any, exchange string, routingKey string) error {
	return m.publishJsonHeaders(c, obj, exchange, routingKey, nil)
}

func (m *rabbitMqModule) publishJsonHeaders(c miso.Rail, obj any, exchange string, routingKey string, headers map[string]any) error {
	j, err := json.WriteJson(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal message body, %w", err)
	}
	return m.publishMsg(c, j, exchange, routingKey, "application/json", headers)
}

func (m *rabbitMqModule) publishText(c miso.Rail, msg string, exchange string, routingKey string) error {
	return m.publishMsg(c, util.UnsafeStr2Byt(msg), exchange, routingKey, "text/plain", nil)
}

func (m *rabbitMqModule) publishMsg(c miso.Rail, msg []byte, exchange string, routingKey string, contentType string, headers map[string]any) error {
	pc, err := m.borrowPubChan()
	if err != nil {
		return fmt.Errorf("failed to obtain channel, unable to publish message, %w", err)
	}
	defer m.returnPubChan(pc)

	if headers == nil {
		headers = map[string]any{}
	}

	// propogate trace through headers
	miso.UsePropagationKeys(func(key string) {
		headers[key] = c.CtxValue(key)
	})

	publishing := amqp.Publishing{
		ContentType:  contentType,
		DeliveryMode: amqp.Persistent,
		Body:         msg,
		Headers:      headers,
		MessageId:    util.GenIdP("mq_"),
	}
	confirm, err := pc.PublishWithDeferredConfirmWithContext(context.Background(), exchange, routingKey,
		false, false, publishing)
	if err != nil {
		return fmt.Errorf("failed to publish message, %w", err)
	}

	if !confirm.Wait() {
		return fmt.Errorf("failed to publish message, exchange '%v' probably doesn't exist, %w", exchange,
			errMsgNotPublished)
	}

	c.Debugf("Published MQ to exchange '%v', '%s'", exchange, msg)
	return nil
}

func (m *rabbitMqModule) AddRabbitListener(listener RabbitListener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, newWrappingListener(listener))
}

func (m *rabbitMqModule) registerRabbitBinding(b BindingRegistration) {
	m.bindingRegistration = append(m.bindingRegistration, b)
}

func (m *rabbitMqModule) registerRabbitQueue(q QueueRegistration) {
	m.queueRegistration = append(m.queueRegistration, q)
}

func (m *rabbitMqModule) registerRabbitExchange(e ExchangeRegistration) {
	m.exchangeRegistration = append(m.exchangeRegistration, e)
}

// Declare queue using the provided channel immediately
func (m *rabbitMqModule) declareRabbitQueue(ch *amqp.Channel, queue QueueRegistration) error {
	dqueue, e := ch.QueueDeclare(queue.Name, queue.Durable, false, false, false, nil)
	if e != nil {
		return fmt.Errorf("failed to declare queue, %v, %w", queue.Name, e)
	}
	miso.Debugf("Declared queue '%s'", dqueue.Name)
	return nil
}

func redeliverQueue(exchange string, routingKey string) string {
	return fmt.Sprintf("redeliver_%v_%v_%v", exchange, routingKey, redeliverDelay)
}

// Declare binding using the provided channel immediately
func (m *rabbitMqModule) declareRabbitBinding(ch *amqp.Channel, bind BindingRegistration) error {
	if bind.RoutingKey == "" {
		bind.RoutingKey = "#"
	}
	e := ch.QueueBind(bind.Queue, bind.RoutingKey, bind.Exchange, false, nil)
	if e != nil {
		return fmt.Errorf("failed to declare binding, queue: %v, routingkey: %v, exchange: %v, %w", bind.Queue, bind.RoutingKey, bind.Exchange, e)
	}
	miso.Debugf("Declared binding for queue '%s' to exchange '%s' using routingKey '%s'", bind.Queue, bind.Exchange, bind.RoutingKey)

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
		return fmt.Errorf("failed to declare redeliver queue '%v' for '%v', %w", rq, bind.Queue, e)
	}
	m.redeliverQueueMap.Store(rqueue, true) // remember this redeliver queue
	miso.Debugf("Declared redeliver queue '%s' for '%v'", rq.Name, bind.Queue)
	return nil
}

// Declare exchange using the provided channel immediately
func (m *rabbitMqModule) declareRabbitExchange(ch *amqp.Channel, exchange ExchangeRegistration) error {
	if exchange.Kind == "" {
		exchange.Kind = "direct"
	}

	e := ch.ExchangeDeclare(exchange.Name, exchange.Kind, exchange.Durable, false, false, false, exchange.Properties)
	if e != nil {
		return fmt.Errorf("failed to declare exchange, %v, %w", exchange.Name, e)
	}
	miso.Debugf("Declared %s exchange '%s'", exchange.Kind, exchange.Name)
	return nil
}

/*
Start RabbitMQ Client (synchronous for the first time, then auto-reconnect later in another goroutine)

This func will attempt to establish connection to broker, declare queues, exchanges and bindings.

Listeners are also created once the intial setup is done.

When connection is lost, it will attmpt to reconnect to recover, unless the given context is done.

To register listener, please use 'AddListener' func.
*/
func StartRabbitMqClient(rail miso.Rail) error {
	return module().startClient(rail)
}

func (m *rabbitMqModule) startClient(rail miso.Rail) error {
	notifyCloseChan, err := m.initRabbitClient(rail)
	if err != nil {
		return err
	}

	go func(notifyCloseChan chan *amqp.Error) {
		isInitial := true
		doneCh := rail.Done()

		for {
			if isInitial {
				isInitial = false
			} else {
				notifyCloseChan, err = m.initRabbitClient(rail)
				if err != nil {
					miso.NoRelayErrorf("Error connecting to RabbitMQ: %v", err)
					time.Sleep(time.Second * 5)
					continue
				}
				if notifyCloseChan == nil {
					return
				}
			}

			select {
			// block until connection is closed, then reconnect, thus continue
			case <-notifyCloseChan:
				miso.Infof("RabbitMq connection notifyCloseChan received, reconnecting")
				time.Sleep(time.Second * 1)
				continue
			// context is done, close the connection, and exit
			case <-doneCh:
				if err := m.close(rail); err != nil {
					miso.Warnf("Failed to close connection to RabbitMQ: %v", err)
				}
				return
			}
		}
	}(notifyCloseChan)

	return nil
}

// Disconnect from RabbitMQ server
func (m *rabbitMqModule) close(rail miso.Rail) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn == nil {
		return nil
	}

	rail.Info("Closing RabbitMQ Connection")
	err := m.conn.Close()
	m.conn = nil
	m.closed = true
	return err
}

// Try to establish Connection
func (m *rabbitMqModule) tryConnRabbit(rail miso.Rail) (*amqp.Connection, error) {
	if m.conn != nil && !m.conn.IsClosed() {
		return m.conn, nil
	}

	c := amqp.Config{}
	c.Properties = map[string]any{
		"connection_name": miso.GetPropStr(miso.PropAppName),
	}
	username := miso.GetPropStr(PropRabbitMqUsername)
	password := miso.GetPropStr(PropRabbitMqPassword)
	vhost := miso.GetPropStr(PropRabbitMqVhost)
	host := miso.GetPropStr(PropRabbitMqHost)
	port := miso.GetPropInt(PropRabbitMqPort)
	dialUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", username, password, host, port, vhost)

	rail.Infof("Establish connection to RabbitMQ: '%s@%s:%d/%s'", username, host, port, vhost)
	cn, e := amqp.DialConfig(dialUrl, c)
	if e != nil {
		return nil, e
	}
	m.conn = cn
	return m.conn, nil
}

// Declare Queus, Exchanges, and Bindings
func (m *rabbitMqModule) decRabbitComp(ch *amqp.Channel) error {
	if ch == nil {
		return errMissingChannel
	}

	for _, queue := range m.queueRegistration {
		if err := m.declareRabbitQueue(ch, queue); err != nil {
			return err
		}
	}

	for _, exchange := range m.exchangeRegistration {
		if err := m.declareRabbitExchange(ch, exchange); err != nil {
			return err
		}
	}

	for _, bind := range m.bindingRegistration {
		if err := m.declareRabbitBinding(ch, bind); err != nil {
			return err
		}
	}

	return nil
}

func (m *rabbitMqModule) rabbitConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.conn != nil
}

/*
Init RabbitMQ Client

return notifyCloseChannel for connection and error
*/
func (m *rabbitMqModule) initRabbitClient(rail miso.Rail) (chan *amqp.Error, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, nil
	}

	// Establish connection if necessary
	conn, err := m.tryConnRabbit(rail)
	if err != nil {
		return nil, err
	}

	notifyCloseChan := make(chan *amqp.Error)
	conn.NotifyClose(notifyCloseChan)

	rail.Debugf("Creating Channel to RabbitMQ")
	ch, e := conn.Channel()
	if e != nil {
		return nil, fmt.Errorf("failed to create channel, %w", e)
	}

	// queues, exchanges, bindings
	e = m.decRabbitComp(ch)
	if e != nil {
		return nil, e
	}
	ch.Close()

	// consumers
	if e = m.startRabbitConsumers(conn); e != nil {
		rail.Errorf("Failed to bootstrap consumer: %v", e)
		return nil, fmt.Errorf("failed to create consumer, %w", e)
	}

	rail.Debugf("RabbitMQ client initialization finished")
	return notifyCloseChan, nil
}

func (m *rabbitMqModule) startRabbitConsumers(conn *amqp.Connection) error {
	for _, v := range m.listeners {
		listener := v
		rmc := rabbitManagedConsumer{
			m:        m,
			conn:     conn,
			listener: listener,
		}
		rmc.start()
	}

	miso.Debug("RabbitMQ consumer initialization finished")
	return nil
}

func (m *rabbitMqModule) startListening(msgCh <-chan amqp.Delivery, listener RabbitListener, routineNo int) {
	go func() {
		name := fmt.Sprintf("Listener [%d-%v]", routineNo, listener.Queue())
		defer miso.Infof("%v stopped", name)
		miso.Infof("%v started", name)

		for msg := range msgCh {

			// read trace from headers
			rail := miso.EmptyRail()
			if msg.Headers != nil {
				miso.UsePropagationKeys(func(k string) {
					if hv, ok := msg.Headers[k]; ok {
						rail = rail.WithCtxVal(k, fmt.Sprintf("%v", hv))
					}
				})
			}

			// message body, we only support text
			payload := util.UnsafeByt2Str(msg.Body)

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

			// make sure we indeed have a redeliver queue for this message
			if _, ok := m.redeliverQueueMap.Load(rq); ok {

				var nextHeaders map[string]any = nil

				// before we retry, check whether the message configured max retry, and whether we should continue redelivering the message.
				if msg.Headers != nil {
					if maxv, ok := msg.Headers[HeaderRabbitMaxRetry]; ok {
						max := cast.ToInt(maxv)
						curr := 0
						if currv, ok := msg.Headers[HeaderRabbitCurrRetry]; ok {
							curr = cast.ToInt(currv)
						}
						if curr >= max {
							msg.Ack(false)
							rail.Infof("RabbitMQ message %v exceeds configured max redelivery times: %d, payload: '%s', message dropped", msg.MessageId, max, payload)
							continue
						}
						nextHeaders = make(map[string]any, 2)
						nextHeaders[HeaderRabbitMaxRetry] = maxv
						nextHeaders[HeaderRabbitCurrRetry] = curr + 1
					}
				}

				err := PublishMsg(rail, msg.Body, defaultExchange, rq, msg.ContentType, nextHeaders)
				if err == nil {
					msg.Ack(false)
					rail.Debugf("Sent message to delayed redeliver queue: %v", msg.MessageId)
					continue
				}
				rail.Errorf("Failed to send message to delayed redeliver queue: %v, %v", msg.MessageId, err)
			}

			// nack the message
			msg.Nack(false, true)
			rail.Debugf("Nacked message: %v", msg.MessageId)
		}

	}()
}

// borrow a publishing channel from the pool.
func (m *rabbitMqModule) borrowPubChan() (*amqp.Channel, error) {
	ch := m.pubChanPool.Get()
	if ch == nil {
		return nil, errors.New("could not create new RabbitMQ channel")
	}

	ach := ch.(*amqp.Channel)
	if ach.IsClosed() { // ach for whatever reason is now closed, we just dump it, create a new one manually
		ch = m.pubChanPool.New() // force to create a new one
		ach = ch.(*amqp.Channel)

		if ach == nil { // still unable to create a new channel, the connection may be broken
			return nil, errors.New("could not create new RabbitMQ channel")
		}
	}
	return ach, nil
}

// return a publishing channel back to the pool.
//
// if the channel is closed already, it's simply ignored.
func (m *rabbitMqModule) returnPubChan(ch *amqp.Channel) {
	if ch.IsClosed() {
		return // for some errors, the channel may be closed, we simply dump it
	}
	m.pubChanPool.Put(ch) // put it back to the pool
}

// open a new publishing channel.
//
// return error if it failed to open new channel, e.g., connection is also missing or closed.
func (m *rabbitMqModule) newPubChan() (*amqp.Channel, error) {
	newChan, err := m.newChan()
	if err != nil {
		return nil, fmt.Errorf("could not create new RabbitMQ channel, %w", err)
	}

	if err = newChan.Confirm(false); err != nil {
		return nil, fmt.Errorf("channel could not be put into confirm mode, %w", err)
	}

	miso.Debugf("Created new RabbitMQ publishing channel")

	return newChan, nil
}

func (m *rabbitMqModule) newChan() (*amqp.Channel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return nil, errors.New("rabbitmq connection is missing")
	}
	return m.conn.Channel()
}

func rabbitBootstrap(rail miso.Rail) error {
	m := module()
	if e := m.startClient(rail); e != nil {
		return fmt.Errorf("failed to establish connection to RabbitMQ, %w", e)
	}
	miso.AddShutdownHook(func() {
		if err := m.close(miso.EmptyRail()); err != nil {
			miso.NoRelayErrorf("Failed to close rabbitmq connection, %v", err)
		}
	})
	return nil
}

func rabbitBootstrapCondition(rail miso.Rail) (bool, error) {
	return miso.GetPropBool(PropRabbitMqEnabled), nil
}

func RabbitConnected() bool {
	return module().rabbitConnected()
}

func RabbitDisconnect(rail miso.Rail) error {
	return module().close(rail)
}

// Declare binding on client initialization
func RegisterRabbitBinding(b BindingRegistration) {
	module().registerRabbitBinding(b)
}

// Declare queue on client initialization
func RegisterRabbitQueue(q QueueRegistration) {
	module().registerRabbitQueue(q)
}

// Declare exchange on client initialization
func RegisterRabbitExchange(e ExchangeRegistration) {
	module().registerRabbitExchange(e)
}

// Register pending message listener.
//
// Listeners will be started in StartRabbitMqClient func when the connection to broker is established.
//
// For any message that the listener is unable to process (returning error), the message is redelivered indefinitively
// with a delay of 10 seconds until the message is finally processed without error.
func AddRabbitListener(listener RabbitListener) {
	module().AddRabbitListener(listener)
}

// Wrapping RabbitListener that makes sure Handle(..) doesn't panic
type wrappingListener struct {
	delegate RabbitListener
}

func (w wrappingListener) Queue() string {
	return w.delegate.Queue()
}

func (w wrappingListener) Handle(rail miso.Rail, payload string) (err error) {
	defer func() {
		if v := recover(); v != nil {
			util.PanicLog("panic recovered, %v\n%v", v, util.UnsafeByt2Str(debug.Stack()))
			err = fmt.Errorf("listener panic recovered, %v", v)
		}
	}()

	err = w.delegate.Handle(rail, payload)
	return
}

func (w wrappingListener) Concurrency() int {
	return w.delegate.Concurrency()
}

func (w wrappingListener) QosSpec() int {
	return w.delegate.QosSpec()
}

func (w wrappingListener) String() string {
	if ws, ok := w.delegate.(fmt.Stringer); ok {
		return ws.String()
	}
	return fmt.Sprintf("%v", w.delegate)
}

func newWrappingListener(l RabbitListener) RabbitListener {
	if l == nil {
		panic(fmt.Errorf("RabbitListener is nil, unable to wrappingListener"))
	}
	return wrappingListener{
		delegate: l,
	}
}

type rabbitManagedConsumer struct {
	m        *rabbitMqModule
	conn     *amqp.Connection
	listener RabbitListener
}

func (r *rabbitManagedConsumer) start() error {
	global := miso.GetPropInt(PropRabbitMqConsumerQos)
	qname := r.listener.Queue()
	concurrency := r.listener.Concurrency()
	if concurrency < 1 {
		concurrency = 1
	}

	if r.conn.IsClosed() {
		miso.Warn("RabbitMQ connection has been closed, abort rabbitManagedConsumer.Start(), new *rabbitManagedConsumer will be created")
		return nil
	}

	ch, e := r.conn.Channel()
	if e != nil {
		return e
	}
	qos := global
	if r.listener.QosSpec() > 0 {
		qos = r.listener.QosSpec()
	}

	if e := ch.Qos(qos, 0, false); e != nil {
		ch.Close()
		return e
	}

	msgCh, err := ch.Consume(qname, "", false, false, false, false, nil)
	if err != nil {
		miso.NoRelayErrorf("Failed to listen to '%s', err: %v", qname, err)
		return err
	}

	for i := 0; i < concurrency; i++ {
		r.m.startListening(msgCh, r.listener, i)
	}
	r.onNotifyCancelClose(ch)
	miso.Infof("Bootstrapped consumer for '%v' with concurrency: %v", r.listener.Queue(), r.listener.Concurrency())

	return nil
}

func (r *rabbitManagedConsumer) retryStart() {
	for {
		err := r.start()
		if err == nil {
			return
		}
		miso.NoRelayErrorf("Failed to start rabbitManagedConsumer, retry, %v", err)
		time.Sleep(time.Second * 1)
	}
}

func (r *rabbitManagedConsumer) onNotifyCancelClose(ch *amqp.Channel) {
	go func() {
		notifyCloseChan := ch.NotifyClose(make(chan *amqp.Error, 1))
		notifyCancelChan := ch.NotifyCancel(make(chan string, 1))

		select {
		case err := <-notifyCloseChan:
			if err != nil {
				miso.NoRelayErrorf("receive from notifyCloseChan, reconnecting to amqp server, %v", err)
				r.retryStart()
				return
			}
			miso.Infof("receive from notifyCloseChan, consumer exiting")
		case err := <-notifyCancelChan:
			miso.NoRelayErrorf("receive from notifyCancelChan, reconnecting to amqp server, %v", err)
			r.retryStart()
		}
	}()
}
