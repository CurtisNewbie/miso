package rabbit

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/atom"
	"github.com/curtisnewbie/miso/util/idutil"
	"github.com/curtisnewbie/miso/util/json"
	"github.com/curtisnewbie/miso/util/pool"
	"github.com/curtisnewbie/miso/util/retry"
	"github.com/curtisnewbie/miso/util/strutil"
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
	supervising       bool
	quit              chan struct{}
	quitOnce          sync.Once
	redeliverQueueMap *sync.Map
	gen               *rabbitGeneration // current connection generation, accessing it requires obtaining 'mu' lock

	listeners            []RabbitListener
	bindingRegistration  []BindingRegistration
	queueRegistration    []QueueRegistration
	exchangeRegistration []ExchangeRegistration
}

func newModule() *rabbitMqModule {
	m := &rabbitMqModule{
		redeliverQueueMap: &sync.Map{},
		mu:                &sync.Mutex{},
		quit:              make(chan struct{}),
	}
	return m
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
	if e := json.ParseJson(strutil.UnsafeStr2Byt(payload), &t); e != nil {
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
		return errs.Wrapf(err, "failed to marshal message body")
	}
	return m.publishMsg(c, j, exchange, routingKey, "application/json", headers)
}

func (m *rabbitMqModule) publishText(c miso.Rail, msg string, exchange string, routingKey string) error {
	return m.publishMsg(c, strutil.UnsafeStr2Byt(msg), exchange, routingKey, "text/plain", nil)
}

func (m *rabbitMqModule) publishMsg(c miso.Rail, msg []byte, exchange string, routingKey string, contentType string, headers map[string]any) error {
	lease, err := m.borrowPubChan()
	if err != nil {
		return errs.Wrapf(err, "failed to obtain channel, unable to publish message")
	}
	defer m.returnPubChan(lease)

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
		MessageId:    idutil.Id("mq_"),
	}
	confirm, err := lease.ch.PublishWithDeferredConfirmWithContext(context.Background(), exchange, routingKey,
		false, false, publishing)
	if err != nil {
		return errs.Wrapf(err, "failed to publish message")
	}

	if !confirm.Wait() {
		return errs.Wrapf(errMsgNotPublished, "failed to publish message, message was not confirmed by the broker, exchange '%v' may not exist or the connection was lost", exchange)
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
		return errs.Wrapf(e, "failed to declare queue, %v", queue.Name)
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
		return errs.Wrapf(e, "failed to declare binding, queue: %v, routingkey: %v, exchange: %v", bind.Queue, bind.RoutingKey, bind.Exchange)
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
		return errs.Wrapf(e, "failed to declare redeliver queue '%v' for '%v'", rq, bind.Queue)
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
		return errs.Wrapf(e, "failed to declare exchange, %v", exchange.Name)
	}
	miso.Debugf("Declared %s exchange '%s'", exchange.Kind, exchange.Name)
	return nil
}

/*
Start RabbitMQ Client (synchronous for the first time, then auto-reconnect later in another goroutine)

This func will attempt to establish connection to broker, declare queues, exchanges and bindings.

Listeners are also created once the intial setup is done.

When connection is lost, it will attmpt to reconnect to recover, unless the given context is done.

To register listener, please use 'AddRabbitListener' func.
*/
func StartRabbitMqClient(rail miso.Rail) error {
	return module().startClient(rail)
}

func (m *rabbitMqModule) startClient(rail miso.Rail) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	if m.supervising {
		m.mu.Unlock()
		return nil
	}
	m.supervising = true
	m.mu.Unlock()

	gen, err := m.establish(rail)
	if err != nil {
		m.mu.Lock()
		m.supervising = false
		m.mu.Unlock()
		return err
	}

	go m.supervise(rail, gen)
	return nil
}

// Disconnect from RabbitMQ server
func (m *rabbitMqModule) close(rail miso.Rail) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	gen := m.gen
	m.gen = nil
	m.mu.Unlock()

	m.quitOnce.Do(func() { close(m.quit) })

	if gen == nil {
		return nil
	}
	rail.Info("Closing RabbitMQ Connection")
	err := gen.retire()
	gen.waitConsumers(rail)
	return err
}

var (
	// Delay before the first reconnect attempt, doubled per failed attempt.
	reconnectBaseDelay = time.Second * 1
	// Upper bound of the reconnect backoff.
	reconnectMaxDelay = time.Second * 30
	// A generation that stayed alive at least this long resets the backoff.
	reconnectStableAfter = time.Minute * 1
)

var (
	// Backoff before each in-place consumer channel restart attempt. The last
	// entry caps the delay once the attempt number exceeds the slice length.
	channelRestartBackoff = []time.Duration{time.Millisecond * 250, time.Millisecond * 500}
	// Number of consecutive in-place restart attempts before a slot escalates to
	// a connection rebuild.
	maxChannelRestartAttempts = 2
	// A restarted consumer channel that stays up this long resets the slot's
	// failure count, so an occasional death never accumulates into an escalation.
	channelStableAfter = time.Minute * 1
	// Upper bound on how long the connection supervisor waits for a retired
	// generation's consumer slots to exit before it carries on with the rebuild.
	generationConsumerWaitTimeout = time.Second * 5
)

// channelRestartDelay returns the backoff before the given zero-based attempt,
// clamping to the last configured delay when there are more attempts than delays.
func channelRestartDelay(attempt int) time.Duration {
	if len(channelRestartBackoff) == 0 {
		return 0
	}
	if attempt >= len(channelRestartBackoff) {
		return channelRestartBackoff[len(channelRestartBackoff)-1]
	}
	return channelRestartBackoff[attempt]
}

// rabbitGeneration is one dial's worth of live resources: the connection, its
// close listener, the publisher channel pool and the consumers bound to it.
//
// A generation is owned by exactly one supervisor goroutine, which retires it
// before establishing the next one. connCloseCh and escalateSignal are never shared
// across generations.
type rabbitGeneration struct {
	conn           *amqp.Connection
	connCloseCh    chan *amqp.Error
	escalateSignal chan struct{}
	publisher      *rabbitManagedPublisher
	startedAt      atom.Time
	retireOnce     sync.Once

	// done is closed by retire before the connection closes, so a consumer slot
	// never opens a channel or escalates against a generation being replaced.
	done chan struct{}
	// consumerWg tracks the generation's consumer slot goroutines. The supervisor
	// waits on it, bounded, after retiring the generation.
	consumerWg sync.WaitGroup
}

// retire closes the generation's connection at most once, no matter how many
// paths (supervisor, shutdown hook, failed build) race to tear it down.
//
// An already-closed connection is the expected case, not a failure: a generation
// is normally retired because the broker or the network closed the connection
// first, and amqp091 reports that as amqp.ErrClosed. That case is normalized to
// nil so callers don't log a bogus warning on every reconnect; genuine close
// errors still propagate.
func (g *rabbitGeneration) retire() error {
	var err error
	g.retireOnce.Do(func() {
		if g.done != nil {
			close(g.done)
		}
		if g.conn != nil {
			err = g.conn.Close()
		}
	})
	if errors.Is(err, amqp.ErrClosed) {
		return nil
	}
	return err
}

// waitConsumers blocks until the generation's consumer slots exit, or until the
// bounded teardown timeout expires. A slot that hangs must not block a rebuild.
func (g *rabbitGeneration) waitConsumers(rail miso.Rail) {
	finished := make(chan struct{})
	go func() {
		g.consumerWg.Wait()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(generationConsumerWaitTimeout):
		rail.Warnf("Timed out waiting for RabbitMQ consumer channels to stop")
	}
}

// channelSlot owns exactly one consumer channel for the life of a generation.
// It reopens the channel on the generation's connection whenever it dies, and
// escalates to the connection supervisor when restarts keep failing.
//
// open, prepare, start, watchCloseCancel and closeChannel are injected so the restart cycle
// can run without a broker.
// channelSlot owns exactly one consumer channel for the life of a generation
// and restarts it in place when it dies. Its operations are injected so the
// restart cycle can run without a broker.
type channelSlot struct {
	// name identifies the slot in logs, e.g. "Consumer 'event.bus.file.moved'".
	name string

	// open creates one channel on the current generation's connection. It
	// declares nothing and subscribes to nothing; an error is reported once and
	// counts as a failed restart attempt.
	open func() (*amqp.Channel, error)

	// prepare re-declares the broker resources this listener needs on a freshly
	// opened channel: its queue, the bindings that target that queue, and the
	// exchanges and redeliver queues those bindings reference. Declaration is
	// idempotent, so it also recreates a queue the broker deleted. It never
	// subscribes.
	prepare func(*amqp.Channel) error

	// start begins consuming on a prepared channel: applies the quality of
	// service, calls Consume, and starts one handler goroutine per configured
	// routine. Once it returns nil the slot is live and only a close or a cancel can end it.
	startConsume func(*amqp.Channel) error

	// watchCloseCancel registers the channel's close and cancel listeners and reports the
	// first of them through the returned channel. It is called right after open,
	// before prepare and start, so a cancel delivered during the subscribe is not
	// lost. It never restarts anything itself.
	watchCloseCancel func(miso.Rail, *amqp.Channel) <-chan struct{}

	// closeChannel releases a channel the slot owns. Closing a channel that is
	// already closed, or was cancelled by the broker, is a no-op, so the slot can
	// call it on every exit path without tracking the cause of death.
	closeChannel func(miso.Rail, *amqp.Channel)

	// escalateSignal is the generation's cap-1 escalation channel. When in-place
	// restarts are exhausted the slot sends on it without blocking; the
	// connection supervisor answers by rebuilding the generation. Several slots
	// escalating at once collapse into one rebuild.
	escalateSignal chan struct{}

	// done is closed when the generation is retired. Every wait in the slot
	// selects on it, so a retired slot exits without opening a channel or
	// escalating against a generation that is already being replaced.
	done chan struct{}

	// wg is the generation's slot WaitGroup. The slot calls Done as it returns,
	// and the supervisor waits on it, with a bound, before installing the next
	// generation.
	wg *sync.WaitGroup

	// failures counts consecutive failed restarts. Only the slot goroutine
	// touches it: it is incremented in recordFailure and reset once a restarted
	// channel survives channelStableAfter.
	failures int
}

// liveChannel pairs a channel the slot owns with the death notification already
// registered on it. The notification is registered before the channel is
// prepared or consumed, so a cancel delivered during the subscribe is never
// missed, and it stays bound to that channel for the channel's whole life.
type liveChannel struct {
	ch   *amqp.Channel
	died <-chan struct{}
}

// superviseRbtChannel owns the live channel and restarts it in place each time
// it dies. A restart counts as a failure when opening, preparing or starting the
// replacement fails, or when the replacement dies before it has stayed up for
// channelStableAfter. The second consecutive failure escalates to the connection
// supervisor and stops the slot.
func (s *channelSlot) superviseRbtChannel(rail miso.Rail, live liveChannel) {
	defer s.wg.Done()

	for {
		select {
		case <-s.done:
			return
		case <-live.died:
		}

		// basic.cancel tells the client the consumer is gone but leaves the
		// channel open, and an abnormal close may already have closed it. Closing
		// is a no-op when it is already gone, so the channel whose death was just
		// observed never leaks before its replacement opens.
		s.closeChannel(rail, live.ch)

		for attempt := 1; attempt <= maxChannelRestartAttempts; attempt++ {
			if !s.waitBackoff(rail, channelRestartDelay(attempt-1)) {
				return
			}

			next, err := s.openChannel(rail)
			if err != nil {
				if s.recordFailure(rail, attempt, err) {
					s.escalate(rail)
					return
				}
				continue
			}

			// The replacement is only a successful restart once it has stayed up
			// for the stable window; dying earlier fails this attempt.
			live = next
			stable := time.NewTimer(channelStableAfter)
			select {
			case <-s.done:
				stable.Stop()
				s.closeChannel(rail, live.ch)
				return
			case <-live.died:
				stable.Stop()
				s.closeChannel(rail, live.ch)
				if s.recordFailure(rail, attempt, "channel died before the stable window") {
					s.escalate(rail)
					return
				}
				continue
			case <-stable.C:
			}

			s.failures = 0
			rail.Debugf("RabbitMQ consumer '%v' restarted in place", s.name)
			break
		}
	}
}

// recordFailure logs one failed in-place attempt and reports whether the slot
// has used every attempt and must escalate.
func (s *channelSlot) recordFailure(rail miso.Rail, attempt int, reason any) bool {
	s.failures++
	rail.Warnf("RabbitMQ consumer '%v' restart attempt %v/%v failed, %v", s.name, attempt, maxChannelRestartAttempts, reason)
	return s.failures >= maxChannelRestartAttempts
}

// escalate asks the connection supervisor to rebuild the generation. The
// generation's cap-1 signal coalesces concurrent escalations into one rebuild.
// A retired slot escalates nothing: the failure is logged and handled elsewhere.
func (s *channelSlot) escalate(rail miso.Rail) {
	select {
	case <-s.done:
		return
	default:
	}
	rail.Errorf("RabbitMQ consumer '%v' could not be restarted after %v attempts, rebuilding connection", s.name, maxChannelRestartAttempts)
	select {
	case s.escalateSignal <- struct{}{}:
	default:
	}
}

// waitBackoff sleeps for d and reports whether the slot should keep going. A
// retired generation aborts the wait immediately, even when the timer and the
// done signal are ready at the same time.
func (s *channelSlot) waitBackoff(rail miso.Rail, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		select {
		case <-s.done:
			return false
		default:
			return true
		}
	case <-s.done:
		return false
	}
}

// closeSlotChannel closes a channel owned by a slot explicitly, so a restart
// iteration never relies on a defer inside the loop.
func closeSlotChannel(rail miso.Rail, ch *amqp.Channel) {
	if ch == nil || ch.IsClosed() {
		return
	}
	if err := ch.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
		rail.Debugf("Failed to close RabbitMQ consumer channel: %v", err)
	}
}

// openChannel runs one restart attempt: open a channel, register its close and
// cancel watchers immediately, then re-declare the listener's components and
// subscribe again.
//
// The watcher must be registered before prepare and start: a basic.cancel can
// arrive the moment Consume returns, and a listener registered afterwards would
// miss it and leave the slot waiting on a channel that no longer has a consumer.
// A failed attempt closes the channel, so its discarded watcher exits too.
func (s *channelSlot) openChannel(rail miso.Rail) (liveChannel, error) {
	ch, err := s.open()
	if err != nil {
		return liveChannel{}, errs.Wrapf(err, "failed to open channel")
	}

	live := liveChannel{ch: ch, died: s.watchCloseCancel(rail, ch)}

	if err := s.prepare(ch); err != nil {
		s.closeChannel(rail, ch)
		return liveChannel{}, errs.Wrapf(err, "failed to prepare channel")
	}
	if err := s.startConsume(ch); err != nil {
		s.closeChannel(rail, ch)
		return liveChannel{}, errs.Wrapf(err, "failed to start channel")
	}
	return live, nil
}

// Dial and register the connection close listener in the same place, so that no
// path can create a connection without a receiver for its close event.
//
// The listener MUST be buffered (cap >= 1): amqp091 delivers the close error to
// every listener while holding the connection lock, so an unbuffered channel
// that nobody reads wedges the connection's shutdown forever.
func dialRabbit(rail miso.Rail) (*amqp.Connection, chan *amqp.Error, error) {
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
	conn, err := amqp.DialConfig(dialUrl, c)
	if err != nil {
		return nil, nil, errs.Wrapf(err, "failed to connect to RabbitMQ")
	}
	return conn, conn.NotifyClose(make(chan *amqp.Error, 1)), nil
}

// Declare Queus, Exchanges, and Bindings
func (m *rabbitMqModule) declareComponents(ch *amqp.Channel) error {
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

// establish dials the broker, declares components, starts the publisher and the
// consumers, then installs the generation. Every error path retires the partial
// generation, so a half-built connection never keeps a live listener around.
func (m *rabbitMqModule) establish(rail miso.Rail) (*rabbitGeneration, error) {
	conn, connCloseCh, err := dialRabbit(rail)
	if err != nil {
		return nil, err
	}

	gen := &rabbitGeneration{
		conn:           conn,
		connCloseCh:    connCloseCh,
		escalateSignal: make(chan struct{}, 1),
		startedAt:      atom.Now(),
		done:           make(chan struct{}),
	}

	// gen is a local, not a named return: a `return nil, err` would nil a named
	// return and the deferred retire would then never close the half-built
	// connection, leaking a socket per failed attempt.
	installed := false
	defer func() {
		if installed {
			return
		}
		if e := gen.retire(); e != nil {
			rail.Warnf("Failed to close half-built RabbitMQ connection: %v", e)
		}
		gen.waitConsumers(rail)
	}()

	ch, e := conn.Channel()
	if e != nil {
		err = errs.Wrapf(e, "failed to create channel")
		rail.Errorf("Failed to open RabbitMQ channel for components declaration, %v", err)
		return nil, err
	}
	e = m.declareComponents(ch)
	if cerr := ch.Close(); cerr != nil {
		rail.Debugf("Failed to close RabbitMQ declaration channel, %v", cerr)
	}
	if e != nil {
		err = e
		rail.Errorf("Failed to declare RabbitMQ components, %v", err)
		return nil, err
	}

	// publisher
	if err = m.startRabbitPublisher(rail, gen); err != nil {
		rail.Errorf("Failed to bootstrap publisher: %v", err)
		err = errs.Wrapf(err, "failed to create publisher")
		return nil, err
	}

	// consumers
	if err = m.startRabbitConsumers(rail, gen); err != nil {
		rail.Errorf("Failed to bootstrap consumer: %v", err)
		err = errs.Wrapf(err, "failed to create consumer")
		return nil, err
	}

	// install the new generation only if the module was not closed in the meantime
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		err = errs.NewErrf("rabbitmq client is closed")
		return nil, err
	}
	m.gen = gen
	m.mu.Unlock()

	installed = true
	rail.Debugf("RabbitMQ client initialization finished")
	return gen, nil
}

// supervise owns the current generation until the connection or one of its
// channels dies, then rebuilds. It is the only goroutine that replaces m.gen.
func (m *rabbitMqModule) supervise(rail miso.Rail, gen *rabbitGeneration) {
	defer func() {
		m.mu.Lock()
		if m.gen == gen {
			m.gen = nil
		}
		if m.gen == nil || m.gen == gen {
			m.supervising = false
		}
		m.mu.Unlock()
	}()

	superviseRbtConn(rail, gen,
		func() (*rabbitGeneration, error) {
			return m.establish(rail)
		},
		m.quit,
		func(g *rabbitGeneration) {
			if err := g.retire(); err != nil {
				rail.Warnf("Failed to close RabbitMQ connection: %v", err)
			}
		})
}

// superviseRbtConn is the reconnect loop, decoupled from amqp so it can be tested
// without a broker. It watches the current generation's connection and channels,
// retires it on any event, then establishes generations with exponential backoff
// until one succeeds or quit is closed.
func superviseRbtConn(rail miso.Rail, gen *rabbitGeneration,
	establish func() (*rabbitGeneration, error),
	quit <-chan struct{},
	retire func(*rabbitGeneration)) {
	if gen == nil {
		return
	}

	delay := reconnectBaseDelay
	for {
		select {
		case err := <-gen.connCloseCh:
			if err != nil {
				rail.Errorf("RabbitMQ connection closed, reconnecting, %v", err)
			} else {
				rail.Infof("RabbitMQ connection closed, reconnecting")
			}
		case <-gen.escalateSignal:
			rail.Infof("Rebuilding RabbitMQ connection after a consumer channel could not be restarted")
		case <-quit:
			retire(gen)
			gen.waitConsumers(rail)
			return
		// Bootstrap passes EmptyRail, whose Done() is nil, so this case may never
		// be ready. It is a best-effort stop for callers that pass a cancellable
		// rail; quit is the authoritative one.
		case <-rail.Done():
			retire(gen)
			gen.waitConsumers(rail)
			return
		}

		// A generation that stayed up long enough resets the backoff, so a single
		// blip does not leave the next reconnect waiting for 30s.
		if time.Since(gen.startedAt.Time) >= reconnectStableAfter {
			delay = reconnectBaseDelay
		}
		retire(gen)
		gen.waitConsumers(rail)

		for {
			if !waitReconnect(rail, quit, delay) {
				return
			}
			next, err := establish()
			if err == nil && next != nil {
				gen = next
				break
			}
			if err == nil {
				err = errs.NewErrf("rabbitmq connection generation is nil")
			}
			delay = delay * 2
			if delay > reconnectMaxDelay {
				delay = reconnectMaxDelay
			}
			rail.Errorf("Failed to reconnect to RabbitMQ, retrying in %v, %v", delay, err)
		}
	}
}

// waitReconnect sleeps for d and reports whether the supervisor should continue.
func waitReconnect(rail miso.Rail, quit <-chan struct{}, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-quit:
		return false
	case <-rail.Done():
		return false
	}
}

func (m *rabbitMqModule) rabbitConnected() bool {
	m.mu.Lock()
	gen := m.gen
	m.mu.Unlock()
	return gen != nil && gen.conn != nil && !gen.conn.IsClosed()
}

func (m *rabbitMqModule) startRabbitConsumers(rail miso.Rail, gen *rabbitGeneration) error {
	// snapshot the listeners, AddRabbitListener may append concurrently
	m.mu.Lock()
	listeners := make([]RabbitListener, len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.Unlock()

	for _, listener := range listeners {
		slot := m.newConsumerSlot(rail, gen, listener)
		// fail-fast bootstrap: the first channel is opened inline, so a broken
		// declaration still fails startClient instead of failing in the background
		live, err := slot.openChannel(rail)
		if err != nil {
			return errs.Wrapf(err, "failed to start %v", slot.name)
		}
		gen.consumerWg.Add(1)
		go slot.superviseRbtChannel(rail, live)
	}

	miso.Debug("RabbitMQ consumer initialization finished")
	return nil
}

// newConsumerSlot wires one consumer listener to the generation it belongs to.
// The slot owns the listener's single channel and restarts it in place.
func (m *rabbitMqModule) newConsumerSlot(rail miso.Rail, gen *rabbitGeneration, listener RabbitListener) *channelSlot {
	name := fmt.Sprintf("Consumer '%v'", listener.Queue())
	consumer := rabbitManagedConsumer{m: m, listener: listener}
	return &channelSlot{
		name: name,
		open: func() (*amqp.Channel, error) {
			ch, err := gen.conn.Channel()
			if err != nil {
				return nil, errs.Wrapf(err, "failed to obtain rabbitmq channel for %v", name)
			}
			return ch, nil
		},
		prepare: func(ch *amqp.Channel) error {
			return m.prepareConsumerChannel(ch, listener.Queue())
		},
		startConsume: func(ch *amqp.Channel) error {
			return consumer.start(rail, ch)
		},
		watchCloseCancel: func(rail miso.Rail, ch *amqp.Channel) <-chan struct{} {
			return watchChannelDeath(rail, ch, name, gen.done)
		},
		closeChannel:   closeSlotChannel,
		escalateSignal: gen.escalateSignal,
		done:           gen.done,
		wg:             &gen.consumerWg,
	}
}

// prepareConsumerChannel re-declares the components this listener consumes:
// its queue, the bindings that target that queue, the exchanges those bindings
// reference, and the redeliver queues those bindings create. Declaration is
// idempotent, so a channel death that left the queue intact costs one round
// trip.
func (m *rabbitMqModule) prepareConsumerChannel(ch *amqp.Channel, queue string) error {
	if ch == nil {
		return errMissingChannel
	}

	for _, bind := range m.bindingRegistration {
		if bind.Queue != queue || bind.Exchange == "" {
			continue
		}
		if ex, ok := m.exchangeRegistrationFor(bind.Exchange); ok {
			if err := m.declareRabbitExchange(ch, ex); err != nil {
				return err
			}
		}
	}

	for _, q := range m.queueRegistration {
		if q.Name != queue {
			continue
		}
		if err := m.declareRabbitQueue(ch, q); err != nil {
			return err
		}
	}

	for _, bind := range m.bindingRegistration {
		if bind.Queue != queue {
			continue
		}
		if err := m.declareRabbitBinding(ch, bind); err != nil {
			return err
		}
	}

	return nil
}

func (m *rabbitMqModule) exchangeRegistrationFor(name string) (ExchangeRegistration, bool) {
	for _, ex := range m.exchangeRegistration {
		if ex.Name == name {
			return ex, true
		}
	}
	return ExchangeRegistration{}, false
}

// watchChannelDeath reports the death of one consumer channel through the
// returned channel and exits. It is used by channel slots instead of the
// generation-wide signal: a consumer channel death restarts that slot in place
// and never wakes the connection supervisor.
func watchChannelDeath(rail miso.Rail, ch *amqp.Channel, name string, done <-chan struct{}) <-chan struct{} {
	died := make(chan struct{}, 1)
	closeCh := ch.NotifyClose(make(chan *amqp.Error, 1))
	cancelCh := ch.NotifyCancel(make(chan string, 1))

	go func() {
		select {
		case err := <-closeCh:
			if err != nil {
				rail.Warnf("%v closed abnormally, %v", name, err)
			} else {
				rail.Debugf("%v closed", name)
			}
		case ctag, ok := <-cancelCh:
			if ok {
				rail.Warnf("%v cancelled by server, consumer tag: %v", name, ctag)
			} else {
				rail.Debugf("%v closed", name)
			}
		case <-done:
			return
		}

		select {
		case died <- struct{}{}:
		default:
		}
	}()

	return died
}

func (m *rabbitMqModule) startRabbitPublisher(rail miso.Rail, gen *rabbitGeneration) error {

	n := miso.GetPropInt(PropRabbitMqPublisherChanPoolSize)
	if n < 1 {
		n = 1
	}
	rail.Infof("RabbitMQ publisher channel pool size: %v", n)
	gen.publisher = &rabbitManagedPublisher{
		EphPool: pool.NewEphPool(
			func(c *amqp.Channel) (dropped bool) { return c.IsClosed() },
		),
	}

	// a drained pool is refilled on demand from this generation's connection,
	// the same way the bootstrap channels are created
	gen.publisher.open = func() (*amqp.Channel, error) {
		return m.openPublisherChannel(gen)
	}

	for i := 0; i < n; i++ {
		// publisher channels are a cache, not supervised slots: open, confirm and
		// push into the pool. A dead one is dropped at the next borrow.
		ch, err := m.openPublisherChannel(gen)
		if err != nil {
			return errs.Wrapf(err, "failed to create publisher channel")
		}
		gen.publisher.Push(ch)
	}

	miso.Debug("RabbitMQ publisher initialization finished")
	return nil
}

// openPublisherChannel opens one confirmed publisher channel on demand, for when
// the pool has been drained by dead channels.
//
// Nothing watches a publisher channel: the pool drops a closed channel at the
// next borrow and this function opens its replacement, so a dead publisher
// channel never escalates to a connection rebuild.
func (m *rabbitMqModule) openPublisherChannel(gen *rabbitGeneration) (*amqp.Channel, error) {
	ch, err := gen.conn.Channel()
	if err != nil {
		return nil, errs.Wrapf(err, "failed to open rabbitmq publisher channel")
	}

	if err := confirmPublisherChannel(ch); err != nil {
		_ = ch.Close()
		return nil, err
	}

	return ch, nil
}

func (m *rabbitMqModule) startListening(rail miso.Rail, msgCh <-chan amqp.Delivery, listener RabbitListener, routineNo int) {
	go func() {
		name := fmt.Sprintf("Listener [%d-%v]", routineNo, listener.Queue())
		defer rail.Debugf("%v stopped", name)
		rail.Debugf("%v started", name)

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
			payload := strutil.UnsafeByt2Str(msg.Body)

			// listener handling the message payload
			e := listener.Handle(rail, payload)
			if e == nil {
				msg.Ack(false)
				continue
			}

			rail.Errorf("Failed to handle message for queue: '%s', exchange: '%s', routingKey: '%s', payload: '%v', messageId: '%v', %v",
				listener.Queue(), msg.Exchange, msg.RoutingKey, payload, msg.MessageId, e)

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

// 600ms in total
var borrowPubChanBackoff = []time.Duration{
	time.Millisecond * 10,
	time.Millisecond * 30,
	time.Millisecond * 60,
	time.Millisecond * 100,
	time.Millisecond * 150,
	time.Millisecond * 250,
}

// rabbitPublisherLease is a publishing channel plus the pool it came from. The
// pool belongs to a single generation, so returning the channel can never push
// it into the pool of a newer connection.
type rabbitPublisherLease struct {
	pub *rabbitManagedPublisher
	ch  *amqp.Channel
}

// borrow a publishing channel from the pool of the current generation.
func (m *rabbitMqModule) borrowPubChan() (*rabbitPublisherLease, error) {
	m.mu.Lock()
	gen := m.gen
	m.mu.Unlock()

	if gen == nil || gen.publisher == nil || gen.publisher.open == nil {
		return nil, errs.NewErrf("publisher is missing")
	}

	pub := gen.publisher
	ch, err := retry.GetOneWithBackoff(borrowPubChanBackoff, func() (*amqp.Channel, error) {
		if ch, ok := pub.Pop(); ok {
			return ch, nil
		}
		// The pool is drained: channels died and returnPubChan discards closed
		// ones instead of putting them back. Open a replacement instead of
		// failing every publish until the supervisor rebuilds the connection.
		// The lease returns it to the pool, so the pool only ever holds channels
		// that were handed out.
		return pub.open()
	})
	if err != nil {
		return nil, err
	}
	return &rabbitPublisherLease{pub: pub, ch: ch}, nil
}

// return a publishing channel back to the pool it was borrowed from.
//
// if the channel is closed already, it's simply ignored.
func (m *rabbitMqModule) returnPubChan(lease *rabbitPublisherLease) {
	if lease == nil || lease.ch == nil || lease.pub == nil {
		return
	}
	if lease.ch.IsClosed() {
		return // for some errors, the channel may be closed, we simply dump it
	}

	// put it back to the pool
	lease.pub.Push(lease.ch)
}

func rabbitBootstrap(rail miso.Rail) error {
	m := module()
	if e := m.startClient(rail); e != nil {
		return errs.Wrapf(e, "failed to establish connection to RabbitMQ")
	}
	miso.AddAsyncShutdownHook(func() {
		if err := m.close(miso.EmptyRail()); err != nil {
			miso.Errorf("Failed to close rabbitmq connection, %v", err)
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
			miso.Errorf("panic recovered, %v\n%v", v, strutil.UnsafeByt2Str(debug.Stack()))
			err = errs.NewErrf("rabbitmq listener panic recovered, %v", v)
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
		panic(errs.NewErrf("RabbitListener is nil, unable to create wrappingListener"))
	}
	return wrappingListener{
		delegate: l,
	}
}

type rabbitManagedConsumer struct {
	m        *rabbitMqModule
	listener RabbitListener
}

func (r *rabbitManagedConsumer) start(rail miso.Rail, ch *amqp.Channel) error {
	global := miso.GetPropInt(PropRabbitMqConsumerQos)
	qname := r.listener.Queue()
	concurrency := r.listener.Concurrency()
	if concurrency < 1 {
		concurrency = 1
	}

	qos := global
	if r.listener.QosSpec() > 0 {
		qos = r.listener.QosSpec()
	}

	if err := ch.Qos(qos, 0, false); err != nil {
		return errs.Wrap(err)
	}

	msgCh, err := ch.Consume(qname, "", false, false, false, false, nil)
	if err != nil {
		rail.Errorf("Failed to listen to '%s', err: %v", qname, err)
		return errs.Wrap(err)
	}

	for i := 0; i < concurrency; i++ {
		r.m.startListening(rail, msgCh, r.listener, i)
	}

	rail.Infof("Bootstrapped consumer for '%v' with concurrency: %v", r.listener.Queue(), r.listener.Concurrency())

	return nil
}

type rabbitManagedPublisher struct {
	*pool.EphPool[*amqp.Channel]

	// open creates a confirmed publisher channel from the generation's
	// connection. It refills a drained pool lazily; startRabbitPublisher wires it
	// and tests replace it so the borrow path can run without a broker.
	open func() (*amqp.Channel, error)
}

// confirmPublisherChannel puts a publishing channel into confirm mode, which the
// publish path relies on for every channel it borrows.
func confirmPublisherChannel(ch *amqp.Channel) error {
	if err := ch.Confirm(false); err != nil {
		return errs.Wrapf(err, "channel could not be put into confirm mode")
	}
	return nil
}
