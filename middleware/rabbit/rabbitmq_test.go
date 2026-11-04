package rabbit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/atom"
	"github.com/curtisnewbie/miso/util/pool"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitDummy struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

func msgHandler(rail miso.Rail, payload string) error {
	rail.Infof("Received message %s", payload)
	// return errors.New("nack intentionally")
	return nil
}

func jsonMsgHandler(rail miso.Rail, payload RabbitDummy) error {
	rail.Infof("Received message %s", payload)
	return nil
}

func TestInitClient(t *testing.T) {
	requireRabbitBroker(t)
	rail, cancel := miso.EmptyRail().WithCancel()
	miso.LoadConfigFromFile("../conf_dev.yml", rail)
	miso.SetProp(PropRabbitMqUsername, "guest")
	miso.SetProp(PropRabbitMqPassword, "guest")

	AddRabbitListener(JsonMsgListener[RabbitDummy]{QueueName: "dummy-queue", Handler: jsonMsgHandler})
	AddRabbitListener(MsgListener{QueueName: "my-first-queue", Handler: msgHandler})

	RegisterRabbitQueue(QueueRegistration{Name: "my-first-queue", Durable: true})
	RegisterRabbitQueue(QueueRegistration{Name: "my-second-queue", Durable: true})
	RegisterRabbitQueue(QueueRegistration{Name: "dummy-queue", Durable: true})

	RegisterRabbitExchange(ExchangeRegistration{Name: "my-exchange-one", Kind: "direct", Durable: true})
	RegisterRabbitExchange(ExchangeRegistration{Name: "my-exchange-two", Kind: "direct", Durable: true})
	RegisterRabbitExchange(ExchangeRegistration{Name: "dummy-exchange", Kind: "direct", Durable: true})

	RegisterRabbitBinding(BindingRegistration{Queue: "dummy-queue", RoutingKey: "#", Exchange: "dummy-exchange"})
	RegisterRabbitBinding(BindingRegistration{Queue: "my-first-queue", RoutingKey: "myKey1", Exchange: "my-exchange-one"})
	RegisterRabbitBinding(BindingRegistration{Queue: "my-second-queue", RoutingKey: "myKey2", Exchange: "my-exchange-two"})

	if e := StartRabbitMqClient(rail); e != nil {
		t.Fatal(e)
	}
	if !RabbitConnected() {
		t.Fatal("expected rabbitmq client to be connected")
	}

	// make sure that the consumer is created before we cancel the context
	time.Sleep(time.Second * 1)
	cancel()

	rail.Info("Cancelling background context")
	time.Sleep(time.Second * 3)
}

func TestPublishMessage(t *testing.T) {
	requireRabbitBroker(t)
	rail, cancel := miso.EmptyRail().WithCancel()
	miso.LoadConfigFromFile("../conf_dev.yml", rail)
	miso.SetProp(PropRabbitMqUsername, "guest")
	miso.SetProp(PropRabbitMqPassword, "guest")
	miso.SetLogLevel("debug")

	RegisterRabbitQueue(QueueRegistration{Name: "my-first-queue", Durable: true})
	RegisterRabbitQueue(QueueRegistration{Name: "my-second-queue", Durable: true})
	RegisterRabbitQueue(QueueRegistration{Name: "dummy-queue", Durable: true})

	RegisterRabbitExchange(ExchangeRegistration{Name: "my-exchange-one", Kind: "direct", Durable: true})
	RegisterRabbitExchange(ExchangeRegistration{Name: "my-exchange-two", Kind: "direct", Durable: true})
	RegisterRabbitExchange(ExchangeRegistration{Name: "dummy-exchange", Kind: "direct", Durable: true})

	RegisterRabbitBinding(BindingRegistration{Queue: "dummy-queue", RoutingKey: "#", Exchange: "dummy-exchange"})
	RegisterRabbitBinding(BindingRegistration{Queue: "my-first-queue", RoutingKey: "myKey1", Exchange: "my-exchange-one"})
	RegisterRabbitBinding(BindingRegistration{Queue: "my-second-queue", RoutingKey: "myKey2", Exchange: "my-exchange-two"})

	e := StartRabbitMqClient(rail)
	if e != nil {
		t.Error(e)
	}

	time.Sleep(time.Second * 1)

	for i := 0; i < 10; i++ {
		e = PublishText(rail, "yo check me out", "my-exchange-one", "myKey1")
		if e != nil {
			t.Fatal(e)
		}
	}
	time.Sleep(time.Second * 3)
	cancel()
	time.Sleep(time.Second * 1)
}

func TestPublishJsonMessage(t *testing.T) {
	requireRabbitBroker(t)
	rail, cancel := miso.EmptyRail().WithCancel()
	miso.LoadConfigFromFile("../conf_dev.yml", rail)
	miso.SetProp(PropRabbitMqUsername, "guest")
	miso.SetProp(PropRabbitMqPassword, "guest")
	miso.SetLogLevel("debug")

	e := StartRabbitMqClient(rail)
	if e != nil {
		t.Error(e)
	}
	defer cancel()

	time.Sleep(time.Second * 1)

	for i := 0; i < 50; i++ {
		j := i
		dummy := RabbitDummy{Name: fmt.Sprintf("dummy no.%v", j), Desc: "dummy with all the love"}
		e = PublishJson(rail, dummy, "dummy-exchange", "#")
		if e != nil {
			t.Error(e)
		}
	}
	time.Sleep(time.Second * 3)
}

func newTestGeneration() *rabbitGeneration {
	return &rabbitGeneration{
		connCloseCh:    make(chan *amqp.Error, 1),
		escalateSignal: make(chan struct{}, 1),
		startedAt:      atom.Now(),
		done:           make(chan struct{}),
	}
}

// withFastChannelRestart shortens the in-place restart timings so the slot loop
// can be driven in milliseconds. Tests set the attempt limit and the backoff
// independently when they need a specific value.
func withFastChannelRestart(t *testing.T) {
	backoff, stable, wait, attempts := channelRestartBackoff, channelStableAfter, generationConsumerWaitTimeout, maxChannelRestartAttempts
	channelRestartBackoff = []time.Duration{time.Millisecond, time.Millisecond}
	channelStableAfter = time.Millisecond * 20
	generationConsumerWaitTimeout = time.Second
	maxChannelRestartAttempts = 2
	t.Cleanup(func() {
		channelRestartBackoff, channelStableAfter, generationConsumerWaitTimeout, maxChannelRestartAttempts = backoff, stable, wait, attempts
	})
}

// waitForSignal reports whether ch receives a value before the timeout.
func waitForSignal(t *testing.T, ch <-chan struct{}, timeout time.Duration, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(timeout):
		t.Fatalf("timed out after %v waiting for %s", timeout, what)
	}
}

func waitForStartedChannel(t *testing.T, ch <-chan *amqp.Channel, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// TestChannelSlotRestartsInPlaceOnDeath drives one consumer channel death
// through the injected seam and asserts exactly one replacement channel is
// opened on the current connection, with prepare before start.
func TestChannelSlotRestartsInPlaceOnDeath(t *testing.T) {
	withFastChannelRestart(t)

	var opens, prepares, starts int32
	h := newSlotHarness(t, func() (*amqp.Channel, error) {
		atomic.AddInt32(&opens, 1)
		return nil, nil
	})
	h.slot.prepare = func(*amqp.Channel) error {
		atomic.AddInt32(&prepares, 1)
		h.record("prepare")
		return nil
	}
	h.slot.startConsume = func(ch *amqp.Channel) error {
		atomic.AddInt32(&starts, 1)
		h.record("start")
		h.started <- ch
		return nil
	}

	live := h.start()

	// the live channel dies: exactly one replacement must be opened
	live <- struct{}{}
	nextSlotDeath(t, h.deaths, "the replacement channel to be watched")
	waitForStartedChannel(t, h.started, "the replacement channel to start")

	if n := atomic.LoadInt32(&opens); n != 1 {
		t.Fatalf("expected exactly 1 replacement channel, got %v", n)
	}
	if n := atomic.LoadInt32(&prepares); n != 1 {
		t.Fatalf("expected the replacement channel to be prepared once, got %v", n)
	}
	if n := atomic.LoadInt32(&starts); n != 1 {
		t.Fatalf("expected the replacement channel to be started once, got %v", n)
	}
	events := h.eventLog()
	prepareAt, startAt := -1, -1
	for i, e := range events {
		switch e {
		case "prepare":
			prepareAt = i
		case "start":
			startAt = i
		}
	}
	if prepareAt < 0 || startAt < 0 || prepareAt > startAt {
		t.Fatalf("expected prepare before start, got %v", events)
	}

	// a consumer channel death must not wake the connection supervisor
	select {
	case <-h.slot.escalateSignal:
		t.Fatal("consumer channel restart escalated to a connection rebuild")
	default:
	}

	close(h.slot.done)
	waitGroupStopped(t, h.slot.wg, time.Second, "the slot goroutine to exit")
}

// TestChannelSlotRetireDuringBackoffOpensNothing retires the generation while
// the slot waits out its restart backoff: no channel is opened and no escalation
// is signalled.
func TestChannelSlotRetireDuringBackoffOpensNothing(t *testing.T) {
	withFastChannelRestart(t)
	channelRestartBackoff = []time.Duration{time.Second}
	maxChannelRestartAttempts = 2

	var opens int32
	h := newSlotHarness(t, func() (*amqp.Channel, error) {
		atomic.AddInt32(&opens, 1)
		return nil, nil
	})
	live := h.start()

	live <- struct{}{}
	close(h.slot.done)

	waitGroupStopped(t, h.slot.wg, time.Second, "the slot to stop during backoff")
	if n := atomic.LoadInt32(&opens); n != 0 {
		t.Fatalf("expected no channel open after retire, got %v", n)
	}
	select {
	case <-h.slot.escalateSignal:
		t.Fatal("a retired slot escalated to a connection rebuild")
	default:
	}
}

// slotHarness drives a channelSlot without a broker. It records the watch
// registrations, starts and close requests so tests can assert lifecycle order.
type slotHarness struct {
	slot    *channelSlot
	deaths  chan chan struct{}
	started chan *amqp.Channel
	closed  chan *amqp.Channel

	mu     sync.Mutex
	events []string
}

func newSlotHarness(t *testing.T, open func() (*amqp.Channel, error)) *slotHarness {
	t.Helper()
	h := &slotHarness{
		deaths:  make(chan chan struct{}, 16),
		started: make(chan *amqp.Channel, 16),
		closed:  make(chan *amqp.Channel, 16),
	}
	h.slot = &channelSlot{
		name: "Consumer 'q'",
		open: open,
		prepare: func(*amqp.Channel) error {
			h.record("prepare")
			return nil
		},
		startConsume: func(ch *amqp.Channel) error {
			h.record("start")
			h.started <- ch
			return nil
		},
		watchCloseCancel: func(miso.Rail, *amqp.Channel) <-chan struct{} {
			h.record("watch")
			d := make(chan struct{}, 1)
			h.deaths <- d
			return d
		},
		closeChannel: func(rail miso.Rail, ch *amqp.Channel) {
			h.recordClose(rail, ch)
		},
		escalateSignal: make(chan struct{}, 1),
		done:           make(chan struct{}),
		wg:             &sync.WaitGroup{},
	}
	return h
}

// recordClose makes closeChannel report the exact channel it was handed, which
// tests that only need the count can ignore via their own override.
func (h *slotHarness) recordClose(_ miso.Rail, ch *amqp.Channel) {
	h.record("close")
	// the synthetic bootstrap channel is nil; only real attempt channels matter
	if ch != nil {
		h.closed <- ch
	}
}

func (h *slotHarness) record(event string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
}

func (h *slotHarness) eventLog() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.events...)
}

// start runs the slot with a fresh, currently live death signal and returns it.
func (h *slotHarness) start() chan struct{} {
	live := make(chan struct{}, 1)
	h.slot.wg.Add(1)
	go h.slot.superviseRbtChannel(miso.EmptyRail(), liveChannel{died: live})
	return live
}

func nextSlotDeath(t *testing.T, deaths <-chan chan struct{}, what string) chan struct{} {
	t.Helper()
	select {
	case d := <-deaths:
		return d
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return nil
	}
}

func nextClosedChannel(t *testing.T, closed <-chan *amqp.Channel, what string) *amqp.Channel {
	t.Helper()
	select {
	case ch := <-closed:
		return ch
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return nil
	}
}

// TestChannelSlotEscalatesAfterTwoFailedRestarts proves a slot makes at most two
// in-place attempts, logs one warning per attempt and one error per escalation,
// and signals the connection supervisor exactly once.
func TestChannelSlotEscalatesAfterTwoFailedRestarts(t *testing.T) {
	withFastChannelRestart(t)
	logs := captureRabbitTestLogs(t)

	var opens int32
	h := newSlotHarness(t, func() (*amqp.Channel, error) {
		atomic.AddInt32(&opens, 1)
		return nil, errs.NewErrf("broker refused the channel")
	})
	live := h.start()
	live <- struct{}{}

	waitForSignal(t, h.slot.escalateSignal, time.Second, "the slot to escalate")

	if n := atomic.LoadInt32(&opens); n != int32(maxChannelRestartAttempts) {
		t.Fatalf("expected %v restart attempts, got %v", maxChannelRestartAttempts, n)
	}
	warnings := logs.count("restart attempt")
	errors := logs.count("could not be restarted")
	if warnings != maxChannelRestartAttempts {
		t.Fatalf("expected one warning per attempt (%v), got %v", maxChannelRestartAttempts, warnings)
	}
	if errors != 1 {
		t.Fatalf("expected one escalation error, got %v", errors)
	}

	// a stopped slot must not repeat itself
	time.Sleep(time.Millisecond * 100)
	if got := logs.count("restart attempt"); got != warnings {
		t.Fatalf("slot logged repeated warnings after stopping: %v -> %v", warnings, got)
	}
	waitGroupStopped(t, h.slot.wg, time.Second, "the escalated slot to stop")
}

// TestChannelSlotResetsFailureCountAfterStableWindow proves a restarted channel
// that survives the stable window resets the count: a later death gets two
// fresh attempts instead of escalating on the first failure.
func TestChannelSlotResetsFailureCountAfterStableWindow(t *testing.T) {
	withFastChannelRestart(t)
	channelStableAfter = time.Millisecond * 100
	maxChannelRestartAttempts = 2

	var opens int32
	var failOpens atomic.Bool
	h := newSlotHarness(t, func() (*amqp.Channel, error) {
		n := atomic.AddInt32(&opens, 1)
		if n == 1 || failOpens.Load() {
			return nil, errs.NewErrf("broker refused the channel")
		}
		return nil, nil
	})
	live := h.start()

	// first death: attempt 1 fails, attempt 2 succeeds
	live <- struct{}{}
	restarted := nextSlotDeath(t, h.deaths, "the replacement channel to be watched")
	waitForStartedChannel(t, h.started, "the replacement channel to start")

	// let the replacement pass the stable window, resetting the failure count
	time.Sleep(time.Millisecond * 150)

	failOpens.Store(true)
	before := atomic.LoadInt32(&opens)
	restarted <- struct{}{}

	waitForSignal(t, h.slot.escalateSignal, time.Second, "the slot to escalate after two fresh failures")
	if got := atomic.LoadInt32(&opens) - before; got != int32(maxChannelRestartAttempts) {
		t.Fatalf("expected %v attempts after the reset, got %v", maxChannelRestartAttempts, got)
	}
	waitGroupStopped(t, h.slot.wg, time.Second, "the escalated slot to stop")
}

// assertEventBefore fails when the first occurrence of before does not precede
// the first occurrence of after in the recorded slot lifecycle.
func assertEventBefore(t *testing.T, events []string, before, after string) {
	t.Helper()
	first, second := -1, -1
	for i, e := range events {
		if e == before && first < 0 {
			first = i
		}
		if e == after && second < 0 {
			second = i
		}
	}
	if first < 0 || second < 0 || first > second {
		t.Fatalf("expected %q before %q, got %v", before, after, events)
	}
}

// TestChannelSlotStartFailureClosesChannelAndEscalates proves a failed Consume
// counts as a failed attempt, closes the channel and escalates on the second
// failure.
func TestChannelSlotStartFailureClosesChannelAndEscalates(t *testing.T) {
	withFastChannelRestart(t)
	maxChannelRestartAttempts = 2

	sentinel := &amqp.Channel{}
	var opens int32
	h := newSlotHarness(t, func() (*amqp.Channel, error) {
		atomic.AddInt32(&opens, 1)
		return sentinel, nil
	})
	h.slot.startConsume = func(*amqp.Channel) error {
		h.record("start")
		return errs.NewErrf("consume refused")
	}

	live := h.start()
	live <- struct{}{}

	waitForSignal(t, h.slot.escalateSignal, time.Second, "the slot to escalate")
	if n := atomic.LoadInt32(&opens); n != int32(maxChannelRestartAttempts) {
		t.Fatalf("expected %v attempts, got %v", maxChannelRestartAttempts, n)
	}
	for i := 0; i < maxChannelRestartAttempts; i++ {
		if ch := nextClosedChannel(t, h.closed, "the failed attempt channel to be closed"); ch != sentinel {
			t.Fatalf("expected the failed attempt's channel to be closed, got %p", ch)
		}
	}
	// the watcher is registered before Consume, so a cancel cannot slip through
	assertEventBefore(t, h.eventLog(), "watch", "start")
	waitGroupStopped(t, h.slot.wg, time.Second, "the escalated slot to stop")
}

// TestChannelSlotEarlyDeathClosesReplacementAndEscalates proves a replacement
// that dies before the stable window counts as a failed attempt, closes the
// replacement and escalates on the second death.
func TestChannelSlotEarlyDeathClosesReplacementAndEscalates(t *testing.T) {
	withFastChannelRestart(t)
	maxChannelRestartAttempts = 2
	channelStableAfter = time.Second

	sentinel := &amqp.Channel{}
	var opens int32
	h := newSlotHarness(t, func() (*amqp.Channel, error) {
		atomic.AddInt32(&opens, 1)
		return sentinel, nil
	})
	live := h.start()
	live <- struct{}{}

	for i := 0; i < int(maxChannelRestartAttempts); i++ {
		died := nextSlotDeath(t, h.deaths, "the replacement channel to be watched")
		waitForStartedChannel(t, h.started, "the replacement channel to start")
		died <- struct{}{} // dies before the stable window
		if ch := nextClosedChannel(t, h.closed, "the early-dead replacement to be closed"); ch != sentinel {
			t.Fatalf("expected the early-dead replacement to be closed, got %p", ch)
		}
	}

	waitForSignal(t, h.slot.escalateSignal, time.Second, "the slot to escalate")
	if n := atomic.LoadInt32(&opens); n != int32(maxChannelRestartAttempts) {
		t.Fatalf("expected %v attempts, got %v", maxChannelRestartAttempts, n)
	}
	assertEventBefore(t, h.eventLog(), "watch", "start")
	waitGroupStopped(t, h.slot.wg, time.Second, "the escalated slot to stop")
}

// TestEscalatingSlotsProduceOneRebuild proves several slots escalating at the
// same time collapse into one connection rebuild.
func TestEscalatingSlotsProduceOneRebuild(t *testing.T) {
	withFastReconnect(t)
	withFastChannelRestart(t)
	channelRestartBackoff = []time.Duration{time.Millisecond}
	maxChannelRestartAttempts = 1

	gen0 := newTestGeneration()
	gen1 := newTestGeneration()
	quit := make(chan struct{})
	done := make(chan struct{})
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var establishes int32

	go func() {
		defer close(done)
		superviseRbtConn(miso.EmptyRail(), gen0,
			func() (*rabbitGeneration, error) {
				atomic.AddInt32(&establishes, 1)
				entered <- struct{}{}
				<-release
				return gen1, nil
			},
			quit,
			func(g *rabbitGeneration) { g.retire() })
	}()

	newExplodingSlot := func(name string) *channelSlot {
		return &channelSlot{
			name:         name,
			open:         func() (*amqp.Channel, error) { return nil, errs.NewErrf("refused") },
			prepare:      func(*amqp.Channel) error { return nil },
			startConsume: func(*amqp.Channel) error { return nil },
			watchCloseCancel: func(miso.Rail, *amqp.Channel) <-chan struct{} {
				return make(chan struct{}, 1)
			},
			closeChannel:   func(miso.Rail, *amqp.Channel) {},
			escalateSignal: gen0.escalateSignal,
			done:           gen0.done,
			wg:             &sync.WaitGroup{},
		}
	}

	slotA := newExplodingSlot("Consumer 'a'")
	slotB := newExplodingSlot("Consumer 'b'")
	slotA.wg.Add(1)
	slotB.wg.Add(1)
	deadA := make(chan struct{}, 1)
	deadA <- struct{}{}
	deadB := make(chan struct{}, 1)
	deadB <- struct{}{}
	go slotA.superviseRbtChannel(miso.EmptyRail(), liveChannel{died: deadA})
	go slotB.superviseRbtChannel(miso.EmptyRail(), liveChannel{died: deadB})

	select {
	case <-entered:
	case <-time.After(time.Second * 2):
		t.Fatal("escalations did not trigger a rebuild")
	}
	close(release)
	time.Sleep(time.Millisecond * 50)
	if n := atomic.LoadInt32(&establishes); n != 1 {
		t.Fatalf("expected one coalesced rebuild, got %v", n)
	}
	waitGroupStopped(t, slotA.wg, time.Second, "slot a to stop")
	waitGroupStopped(t, slotB.wg, time.Second, "slot b to stop")
	close(quit)
	<-done
}

func TestSupervisorCoalescesEscalationWithConnectionClose(t *testing.T) {
	withFastReconnect(t)

	gen0 := newTestGeneration()
	gen1 := newTestGeneration()
	quit := make(chan struct{})
	done := make(chan struct{})
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var establishes int32

	go func() {
		defer close(done)
		superviseRbtConn(miso.EmptyRail(), gen0,
			func() (*rabbitGeneration, error) {
				atomic.AddInt32(&establishes, 1)
				entered <- struct{}{}
				<-release
				return gen1, nil
			},
			quit,
			func(g *rabbitGeneration) { g.retire() })
	}()

	// both events are pending before the supervisor picks one, which is the race
	// between a slot escalating and the broker closing the connection
	gen0.escalateSignal <- struct{}{}
	gen0.connCloseCh <- amqp.ErrClosed

	select {
	case <-entered:
	case <-time.After(time.Second * 2):
		t.Fatal("the connection event did not trigger a rebuild")
	}
	close(release)
	time.Sleep(time.Millisecond * 50)
	if n := atomic.LoadInt32(&establishes); n != 1 {
		t.Fatalf("expected one rebuild for an escalation racing a close, got %v", n)
	}
	close(quit)
	<-done
}

// TestChannelSlotRetireDuringStableWindowExitsPromptly proves a slot holding a
// freshly restarted channel still stops as soon as the generation is retired.
func TestChannelSlotRetireDuringStableWindowExitsPromptly(t *testing.T) {
	withFastChannelRestart(t)
	channelStableAfter = time.Second

	h := newSlotHarness(t, func() (*amqp.Channel, error) { return nil, nil })

	live := h.start()
	live <- struct{}{}
	nextSlotDeath(t, h.deaths, "the replacement channel to be watched")
	waitForStartedChannel(t, h.started, "the replacement channel to start")

	close(h.slot.done)
	waitGroupStopped(t, h.slot.wg, time.Second, "the slot to stop during the stable window")
	select {
	case <-h.slot.escalateSignal:
		t.Fatal("a retired slot escalated to a connection rebuild")
	default:
	}
}

func waitGroupStopped(t *testing.T, wg *sync.WaitGroup, timeout time.Duration, what string) {
	t.Helper()
	stopped := make(chan struct{})
	go func() {
		wg.Wait()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(timeout):
		t.Fatalf("timed out after %v waiting for %s", timeout, what)
	}
}

// TestSupervisorWaitsForSlotsOnRetire proves the supervisor does not install a
// new generation until the retired generation's consumer slots have exited.
func TestSupervisorWaitsForSlotsOnRetire(t *testing.T) {
	withFastReconnect(t)

	gen0 := newTestGeneration()
	gen0.consumerWg.Add(1)
	slotExited := make(chan struct{})
	go func() {
		<-gen0.done
		time.Sleep(time.Millisecond * 20)
		gen0.consumerWg.Done()
		close(slotExited)
	}()

	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		superviseRbtConn(miso.EmptyRail(), gen0,
			func() (*rabbitGeneration, error) { return newTestGeneration(), nil },
			quit,
			func(g *rabbitGeneration) { g.retire() })
	}()

	close(quit)
	select {
	case <-done:
	case <-time.After(time.Second * 2):
		t.Fatal("supervisor did not stop after quit")
	}
	select {
	case <-slotExited:
	default:
		t.Fatal("supervisor returned before the retired generation's slots exited")
	}
}

// TestSupervisorBoundsSlotWait proves a slot that never exits cannot block the
// supervisor: it logs one warning and carries on after the bound.
func TestSupervisorBoundsSlotWait(t *testing.T) {
	withFastReconnect(t)
	logs := captureRabbitTestLogs(t)
	generationConsumerWaitTimeout = time.Millisecond * 50

	gen0 := newTestGeneration()
	gen0.consumerWg.Add(1) // never Done

	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		superviseRbtConn(miso.EmptyRail(), gen0,
			func() (*rabbitGeneration, error) { return newTestGeneration(), nil },
			quit,
			func(g *rabbitGeneration) { g.retire() })
	}()

	close(quit)
	start := time.Now()
	select {
	case <-done:
	case <-time.After(time.Second * 2):
		t.Fatal("supervisor was blocked by a slot that never exits")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("supervisor ignored the slot wait bound, took %v", elapsed)
	}
	if logs.count("Timed out waiting for RabbitMQ consumer channels to stop") != 1 {
		t.Fatalf("expected one slot wait timeout warning, got %v", logs.count("Timed out waiting"))
	}
}

func withFastReconnect(t *testing.T) {
	base, max := reconnectBaseDelay, reconnectMaxDelay
	reconnectBaseDelay = time.Millisecond
	reconnectMaxDelay = time.Millisecond * 5
	t.Cleanup(func() {
		reconnectBaseDelay, reconnectMaxDelay = base, max
	})
}

// testSupervisorRebuild makes sure that either a connection-level close or a
// channel-level signal retires the current generation and installs the next one.
func testSupervisorRebuild(t *testing.T, trigger func(*rabbitGeneration)) {
	t.Helper()
	withFastReconnect(t)

	gen0 := newTestGeneration()
	gen1 := newTestGeneration()

	retired := make(chan *rabbitGeneration, 4)
	established := make(chan struct{}, 4)
	quit := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		superviseRbtConn(miso.EmptyRail(), gen0,
			func() (*rabbitGeneration, error) {
				established <- struct{}{}
				return gen1, nil
			},
			quit,
			func(g *rabbitGeneration) { retired <- g })
	}()

	trigger(gen0)

	select {
	case g := <-retired:
		if g != gen0 {
			t.Fatalf("expected gen0 to be retired, got %p", g)
		}
	case <-time.After(time.Second * 2):
		t.Fatal("gen0 was not retired")
	}

	select {
	case <-established:
	case <-time.After(time.Second * 2):
		t.Fatal("establish was not called after the connection event")
	}

	close(quit)
	<-done

	select {
	case g := <-retired:
		if g != gen1 {
			t.Fatalf("expected gen1 to be retired on quit, got %p", g)
		}
	default:
		t.Fatal("gen1 was not retired on quit")
	}
}

func TestSupervisorReconnectsOnConnClose(t *testing.T) {
	testSupervisorRebuild(t, func(g *rabbitGeneration) {
		g.connCloseCh <- amqp.ErrClosed
	})
}

func TestSupervisorBouncesOnChannelSignal(t *testing.T) {
	testSupervisorRebuild(t, func(g *rabbitGeneration) {
		g.escalateSignal <- struct{}{}
	})
}

func TestSupervisorCoalescesSignals(t *testing.T) {
	withFastReconnect(t)

	gen0 := newTestGeneration()
	gen1 := newTestGeneration()
	quit := make(chan struct{})
	done := make(chan struct{})
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var establishes int32

	go func() {
		defer close(done)
		superviseRbtConn(miso.EmptyRail(), gen0,
			func() (*rabbitGeneration, error) {
				atomic.AddInt32(&establishes, 1)
				entered <- struct{}{}
				<-release
				return gen1, nil
			},
			quit,
			func(g *rabbitGeneration) {})
	}()

	gen0.escalateSignal <- struct{}{}
	select {
	case <-entered:
	case <-time.After(time.Second * 2):
		t.Fatal("establish was not called")
	}

	// flood signals while the rebuild is in progress, they must coalesce
	for i := 0; i < 10; i++ {
		select {
		case gen0.escalateSignal <- struct{}{}:
		default:
		}
	}
	close(release)

	// give the supervisor a chance to erroneously rebuild again
	time.Sleep(time.Millisecond * 50)
	if n := atomic.LoadInt32(&establishes); n != 1 {
		t.Fatalf("expected 1 establish for coalesced signals, got %v", n)
	}

	close(quit)
	<-done
}

func TestSupervisorStopsOnQuit(t *testing.T) {
	withFastReconnect(t)

	gen0 := newTestGeneration()
	quit := make(chan struct{})
	done := make(chan struct{})
	var establishes, retires int32
	baseline := runtime.NumGoroutine()

	go func() {
		defer close(done)
		superviseRbtConn(miso.EmptyRail(), gen0,
			func() (*rabbitGeneration, error) {
				atomic.AddInt32(&establishes, 1)
				return newTestGeneration(), nil
			},
			quit,
			func(g *rabbitGeneration) { atomic.AddInt32(&retires, 1) })
	}()

	close(quit)
	select {
	case <-done:
	case <-time.After(time.Second * 2):
		t.Fatal("supervisor did not stop after quit")
	}

	if n := atomic.LoadInt32(&retires); n != 1 {
		t.Fatalf("expected 1 retire, got %v", n)
	}
	if n := atomic.LoadInt32(&establishes); n != 0 {
		t.Fatalf("expected no establish after quit, got %v", n)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= baseline {
			break
		}
		time.Sleep(time.Millisecond * 5)
	}
	if n := runtime.NumGoroutine(); n > baseline {
		t.Fatalf("supervisor goroutine leaked, baseline %v, now %v", baseline, n)
	}
}

func TestSupervisorSurvivesEstablishErrors(t *testing.T) {
	withFastReconnect(t)

	gen0 := newTestGeneration()
	gen1 := newTestGeneration()
	quit := make(chan struct{})
	done := make(chan struct{})
	established := make(chan struct{}, 1)
	var establishes int32

	go func() {
		defer close(done)
		superviseRbtConn(miso.EmptyRail(), gen0,
			func() (*rabbitGeneration, error) {
				n := atomic.AddInt32(&establishes, 1)
				if n <= 3 {
					return nil, errs.NewErrf("dial failed")
				}
				established <- struct{}{}
				return gen1, nil
			},
			quit,
			func(g *rabbitGeneration) {})
	}()

	gen0.connCloseCh <- amqp.ErrClosed
	select {
	case <-established:
	case <-time.After(time.Second * 2):
		t.Fatal("supervisor did not recover after establish failures")
	}

	if n := atomic.LoadInt32(&establishes); n != 4 {
		t.Fatalf("expected 4 establish attempts, got %v", n)
	}

	close(quit)
	<-done
}

// testPublisher returns a publisher whose channels are opaque sentinels, so the
// borrow path can be exercised without a broker. The pool filter and
// returnPubChan only call IsClosed on the channel, which is safe on a zero value.
func testPublisher(open func() (*amqp.Channel, error)) *rabbitManagedPublisher {
	return &rabbitManagedPublisher{
		EphPool: pool.NewEphPool(func(c *amqp.Channel) (dropped bool) { return c.IsClosed() }),
		open:    open,
	}
}

func testModuleWithPublisher(pub *rabbitManagedPublisher) *rabbitMqModule {
	m := newModule()
	m.gen = &rabbitGeneration{publisher: pub}
	return m
}

func TestPublishChannelRefillWhenPoolDrained(t *testing.T) {
	var opens int32
	created := &amqp.Channel{}
	pub := testPublisher(func() (*amqp.Channel, error) {
		atomic.AddInt32(&opens, 1)
		return created, nil
	})
	m := testModuleWithPublisher(pub)

	// every bootstrap channel was discarded as dead on return, pool is empty
	if _, ok := pub.Pop(); ok {
		t.Fatal("expected an empty pool before borrowing")
	}

	lease, err := m.borrowPubChan()
	if err != nil {
		t.Fatalf("expected a lazily created channel, got error: %v", err)
	}
	if lease.ch != created {
		t.Fatalf("expected the lazily created channel to be leased, got %p", lease.ch)
	}
	if n := atomic.LoadInt32(&opens); n != 1 {
		t.Fatalf("expected 1 lazy channel, got %v", n)
	}

	// the lease puts the replacement back, it becomes the fast path again
	m.returnPubChan(lease)

	lease2, err := m.borrowPubChan()
	if err != nil {
		t.Fatalf("expected the pool to be used after the refill, got error: %v", err)
	}
	m.returnPubChan(lease2)
	if n := atomic.LoadInt32(&opens); n != 1 {
		t.Fatalf("expected no further lazy channel after the refill, got %v", n)
	}
}

func TestPublishChannelRefillFailureAppliesBackoff(t *testing.T) {
	backoff := borrowPubChanBackoff
	borrowPubChanBackoff = []time.Duration{time.Millisecond}
	t.Cleanup(func() { borrowPubChanBackoff = backoff })

	var opens int32
	pub := testPublisher(func() (*amqp.Channel, error) {
		atomic.AddInt32(&opens, 1)
		return nil, errs.NewErrf("channel refused")
	})
	m := testModuleWithPublisher(pub)

	lease, err := m.borrowPubChan()
	if err == nil {
		m.returnPubChan(lease)
		t.Fatal("expected borrow to fail when every refill attempt fails")
	}

	// one attempt per backoff step, each failure deferred to the backoff
	if n := atomic.LoadInt32(&opens); n != int32(len(borrowPubChanBackoff)+1) {
		t.Fatalf("expected %v refill attempts, got %v", len(borrowPubChanBackoff)+1, n)
	}
	if _, ok := pub.Pop(); ok {
		t.Fatal("a failed refill must not leave a channel in the pool")
	}
}

// requireRabbitBroker skips a broker-backed test unless MISO_RABBIT_BROKER_TESTS=1,
// so `go test ./...` stays broker-free by default while the live-broker suite stays
// runnable on demand:
//
//	MISO_RABBIT_BROKER_TESTS=1 go test ./middleware/rabbit/ -count=1
func requireRabbitBroker(t *testing.T) {
	t.Helper()
	if os.Getenv("MISO_RABBIT_BROKER_TESTS") != "1" {
		t.Skip("set MISO_RABBIT_BROKER_TESTS=1 to run against a live RabbitMQ broker")
	}
}

// waitUntil polls cond until it returns true or the deadline expires. Polling
// keeps the integration tests off fixed sleeps for synchronization.
func waitUntil(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
	t.Fatalf("timed out after %v waiting for %s", timeout, what)
}

// resetRabbitModuleForTest stops any supervisor left over by a previous test and
// restores the module singleton to a pristine, startable state. The module is
// stored in the global app store, so the tests must reset it in place.
func resetRabbitModuleForTest(t *testing.T) {
	t.Helper()
	m := module()

	if err := m.close(miso.EmptyRail()); err != nil {
		t.Logf("failed to close rabbitmq client during reset: %v", err)
	}

	waitUntil(t, time.Second*10, "rabbitmq supervisor to stop", func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return !m.supervising
	})

	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = false
	m.gen = nil
	m.quit = make(chan struct{})
	m.quitOnce = sync.Once{}
	m.listeners = nil
	m.bindingRegistration = nil
	m.queueRegistration = nil
	m.exchangeRegistration = nil
	m.redeliverQueueMap = &sync.Map{}
}

// prepareRabbitIntegrationTest points the module at the live local broker and
// gives each test a clean module. The cleanup runs before withFastReconnect's
// restore because it is registered later.
func prepareRabbitIntegrationTest(t *testing.T) {
	t.Helper()
	miso.SetProp(PropRabbitMqHost, "localhost")
	miso.SetProp(PropRabbitMqPort, 5672)
	miso.SetProp(PropRabbitMqUsername, "guest")
	miso.SetProp(PropRabbitMqPassword, "guest")
	miso.SetProp(PropRabbitMqVhost, "/")

	t.Cleanup(func() { resetRabbitModuleForTest(t) })
	resetRabbitModuleForTest(t)
}

func rabbitTestName(prefix string) string {
	return prefix + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func currentRabbitGeneration(t *testing.T) *rabbitGeneration {
	t.Helper()
	m := module()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.gen == nil {
		t.Fatal("expected a live rabbitmq generation")
	}
	return m.gen
}

func collectRabbitPayloads(t *testing.T, ch <-chan string, n int, timeout time.Duration) []string {
	t.Helper()
	got := make([]string, 0, n)
	deadline := time.After(timeout)
	for len(got) < n {
		select {
		case payload := <-ch:
			got = append(got, payload)
		case <-deadline:
			t.Fatalf("received %v/%v payloads before timeout: %v", len(got), n, got)
		}
	}
	return got
}

func assertSamePayloadSet(t *testing.T, expected, actual []string) {
	t.Helper()
	e := append([]string(nil), expected...)
	a := append([]string(nil), actual...)
	sort.Strings(e)
	sort.Strings(a)
	if len(e) != len(a) {
		t.Fatalf("expected payloads %v, got %v", e, a)
	}
	for i := range e {
		if e[i] != a[i] {
			t.Fatalf("expected payloads %v, got %v", e, a)
		}
	}
}

// TestIntegrationReconnectRestoresConsumers proves that a connection death is
// observed by the supervisor, a new generation re-declares the queue and
// re-registers Consume, and that messages consumed and acked before the close
// are not redelivered by the new generation.
func TestIntegrationReconnectRestoresConsumers(t *testing.T) {
	requireRabbitBroker(t)
	withFastReconnect(t)
	prepareRabbitIntegrationTest(t)
	logs := captureRabbitTestLogs(t)

	queueName := rabbitTestName("it-reconnect-")
	RegisterRabbitQueue(QueueRegistration{Name: queueName, Durable: true})
	secondQueueName := rabbitTestName("it-reconnect-second-")
	RegisterRabbitQueue(QueueRegistration{Name: secondQueueName, Durable: true})

	received := make(chan string, 64)
	AddRabbitListener(MsgListener{
		QueueName: queueName,
		Handler: func(rail miso.Rail, payload string) error {
			received <- payload
			return nil
		},
	})
	secondReceived := make(chan string, 64)
	AddRabbitListener(MsgListener{
		QueueName: secondQueueName,
		Handler: func(rail miso.Rail, payload string) error {
			secondReceived <- payload
			return nil
		},
	})

	rail := miso.EmptyRail()
	if err := StartRabbitMqClient(rail); err != nil {
		t.Fatalf("failed to start rabbitmq client: %v", err)
	}
	gen0 := currentRabbitGeneration(t)

	publish := func(prefix string) []string {
		payloads := make([]string, 0, 5)
		for i := 0; i < 5; i++ {
			payload := fmt.Sprintf("%s-%d", prefix, i)
			if err := PublishText(rail, payload, "", queueName); err != nil {
				t.Fatalf("failed to publish %q: %v", payload, err)
			}
			payloads = append(payloads, payload)
		}
		return payloads
	}

	before := publish("before")
	assertSamePayloadSet(t, before, collectRabbitPayloads(t, received, len(before), time.Second*15))

	// everything consumed above was acked before the close, so a later
	// redelivery of any of these payloads means the queue was not drained.
	baselineGoroutines := runtime.NumGoroutine()

	// the supervisor watches this exact connection, closing it is the same event
	// the broker would deliver on a network drop.
	if err := gen0.conn.Close(); err != nil {
		t.Fatalf("failed to close the live connection: %v", err)
	}

	waitUntil(t, time.Second*15, "a new generation to be installed", func() bool {
		m := module()
		m.mu.Lock()
		gen := m.gen
		m.mu.Unlock()
		return gen != nil && gen != gen0 && gen.conn != nil && !gen.conn.IsClosed() && RabbitConnected()
	})

	// the connection death rebuilds exactly once; the initial dial plus this
	// rebuild are the only establish attempts
	time.Sleep(time.Millisecond * 500)
	if n := logs.count("Establish connection to RabbitMQ"); n != 2 {
		t.Fatalf("expected exactly one connection rebuild, got %v dials", n)
	}

	after := publish("after")
	assertSamePayloadSet(t, after, collectRabbitPayloads(t, received, len(after), time.Second*15))

	// every consumer resumes on the new generation, not just the first one
	secondPayloads := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		payload := fmt.Sprintf("after-second-%d", i)
		if err := PublishText(rail, payload, "", secondQueueName); err != nil {
			t.Fatalf("failed to publish %q: %v", payload, err)
		}
		secondPayloads = append(secondPayloads, payload)
	}
	assertSamePayloadSet(t, secondPayloads, collectRabbitPayloads(t, secondReceived, len(secondPayloads), time.Second*15))

	// no duplicate delivery of the drained first batch (or anything else) after
	// the reconnect; a duplicate would show up in the channel beyond the batch.
	quiet := time.After(time.Millisecond * 300)
quietLoop:
	for {
		select {
		case payload := <-received:
			t.Fatalf("unexpected extra delivery after reconnect: %q", payload)
		case payload := <-secondReceived:
			t.Fatalf("unexpected extra delivery on the sibling queue after reconnect: %q", payload)
		case <-quiet:
			break quietLoop
		}
	}

	// the retired generation's consumer, watcher and connection goroutines must
	// have exited; only the replacement generation's delta is tolerated.
	waitUntil(t, time.Second*10, "old generation goroutines to exit", func() bool {
		return runtime.NumGoroutine() <= baselineGoroutines+15
	})

	// the retired connection was already closed, which amqp091 reports as
	// ErrClosed; that expected case must not surface as a close failure.
	if logs.contains("Failed to close RabbitMQ connection") {
		t.Fatal("retiring the already-closed generation logged a close failure")
	}
}

// TestIntegrationReconnectRedeliversUnacked proves the broker requeue path and
// re-Consume work end to end: a delivery that was handed to a listener but not
// acked before the connection died is delivered again on the new generation.
func TestIntegrationReconnectRedeliversUnacked(t *testing.T) {
	requireRabbitBroker(t)
	withFastReconnect(t)
	prepareRabbitIntegrationTest(t)

	queueName := rabbitTestName("it-unacked-")
	RegisterRabbitQueue(QueueRegistration{Name: queueName, Durable: true})

	var deliveries int32
	firstDelivered := make(chan struct{})
	releaseFirst := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseFirst) }) }
	redelivered := make(chan string, 4)

	AddRabbitListener(MsgListener{
		QueueName: queueName,
		Handler: func(rail miso.Rail, payload string) error {
			if atomic.AddInt32(&deliveries, 1) == 1 {
				// keep the delivery unacked past the connection close
				close(firstDelivered)
				<-releaseFirst
				return nil
			}
			redelivered <- payload
			return nil
		},
	})
	t.Cleanup(release)

	rail := miso.EmptyRail()
	if err := StartRabbitMqClient(rail); err != nil {
		t.Fatalf("failed to start rabbitmq client: %v", err)
	}
	gen0 := currentRabbitGeneration(t)

	payload := "unacked-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := PublishText(rail, payload, "", queueName); err != nil {
		t.Fatalf("failed to publish %q: %v", payload, err)
	}

	select {
	case <-firstDelivered:
	case <-time.After(time.Second * 15):
		t.Fatal("the first delivery never reached the listener")
	}

	// the listener is still blocked in Handle, so the delivery is unacked when
	// the connection dies; the broker must requeue it.
	if err := gen0.conn.Close(); err != nil {
		t.Fatalf("failed to close the live connection: %v", err)
	}

	select {
	case got := <-redelivered:
		if got != payload {
			t.Fatalf("expected the unacked payload %q to be redelivered, got %q", payload, got)
		}
	case <-time.After(time.Second * 15):
		t.Fatal("the unacked message was not redelivered after reconnect")
	}
	release()
}

// TestIntegrationPublisherRefillUsesRealChannel proves that refilling a drained
// publisher pool opens a real, confirmed channel on the live generation: the
// lazily opened channel publishes a message that the broker confirms and that
// can then be consumed from the queue.
func TestIntegrationPublisherRefillUsesRealChannel(t *testing.T) {
	requireRabbitBroker(t)
	prepareRabbitIntegrationTest(t)

	queueName := rabbitTestName("it-refill-")
	RegisterRabbitQueue(QueueRegistration{Name: queueName, Durable: true})

	rail := miso.EmptyRail()
	if err := StartRabbitMqClient(rail); err != nil {
		t.Fatalf("failed to start rabbitmq client: %v", err)
	}
	gen := currentRabbitGeneration(t)

	// drain the pool without closing the channels: the borrow path has to refill
	// from the live connection.
	drained := 0
	for {
		if _, ok := gen.publisher.Pop(); !ok {
			break
		}
		drained++
	}
	if drained == 0 {
		t.Fatal("expected a non-empty publisher pool before draining it")
	}

	m := module()
	lease, err := m.borrowPubChan()
	if err != nil {
		t.Fatalf("expected the drained pool to be refilled, got error: %v", err)
	}
	defer m.returnPubChan(lease)

	if lease.ch == nil {
		t.Fatal("expected a publisher channel lease")
	}
	if lease.ch.IsClosed() {
		t.Fatal("expected the lazily opened publisher channel to be live")
	}

	// confirm mode is only wired by openPublisherChannel, so a confirmed publish
	// through this exact channel proves the refill did not skip it.
	body := "refill-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	confirmation, err := lease.ch.PublishWithDeferredConfirm("", queueName, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(body),
	})
	if err != nil {
		t.Fatalf("failed to publish through the refilled channel: %v", err)
	}
	if confirmation == nil {
		t.Fatal("the refilled channel is not in confirm mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	acknowledged, err := confirmation.WaitContext(ctx)
	if err != nil {
		t.Fatalf("waiting for the broker confirmation failed: %v", err)
	}
	if !acknowledged {
		t.Fatal("the broker did not confirm the publish from the refilled channel")
	}

	// consume the message back from the broker with an independent channel, so
	// confirmation is proven, not just the local publish call succeeding.
	verifyCh, err := gen.conn.Channel()
	if err != nil {
		t.Fatalf("failed to open the verification channel: %v", err)
	}
	defer verifyCh.Close()

	var actual string
	waitUntil(t, time.Second*10, "the published message to be consumable", func() bool {
		delivery, ok, err := verifyCh.Get(queueName, true)
		if err != nil {
			t.Fatalf("failed to read from the queue: %v", err)
		}
		if !ok {
			return false
		}
		actual = string(delivery.Body)
		return true
	})
	if actual != body {
		t.Fatalf("expected to consume %q, got %q", body, actual)
	}
}

// TestIntegrationQueueDeletionPastLimitRebuildsOnce proves two failed in-place
// restarts escalate to exactly one connection rebuild, with one warning per
// attempt and one escalation error, and that the consumer recovers once the
// declaration conflict that kept the restarts failing is removed.
func TestIntegrationQueueDeletionPastLimitRebuildsOnce(t *testing.T) {
	requireRabbitBroker(t)

	// a 100ms in-place backoff leaves time to install the conflicting queue
	// between the cancellation and the first restart attempt, while two 100ms
	// attempts keep the test short.
	backoff, reconnectBase, reconnectMax := channelRestartBackoff, reconnectBaseDelay, reconnectMaxDelay
	channelRestartBackoff = []time.Duration{time.Millisecond * 100, time.Millisecond * 100}
	reconnectBaseDelay = time.Millisecond * 100
	reconnectMaxDelay = time.Millisecond * 300
	t.Cleanup(func() {
		channelRestartBackoff, reconnectBaseDelay, reconnectMaxDelay = backoff, reconnectBase, reconnectMax
	})

	prepareRabbitIntegrationTest(t)
	logs := captureRabbitTestLogs(t)

	queueName := rabbitTestName("it-escalate-")
	RegisterRabbitQueue(QueueRegistration{Name: queueName, Durable: true})

	received := make(chan string, 16)
	AddRabbitListener(MsgListener{
		QueueName: queueName,
		Handler: func(rail miso.Rail, payload string) error {
			received <- payload
			return nil
		},
	})

	rail := miso.EmptyRail()
	if err := StartRabbitMqClient(rail); err != nil {
		t.Fatalf("failed to start rabbitmq client: %v", err)
	}
	gen0 := currentRabbitGeneration(t)

	rawConn := dialRawRabbit(t)
	rawCh, err := rawConn.Channel()
	if err != nil {
		t.Fatalf("failed to open the raw channel: %v", err)
	}
	defer rawCh.Close()

	// deleting the consumed durable queue cancels the consumer; the conflicting
	// non-durable declaration makes every re-declare fail with 406
	if _, err := rawCh.QueueDelete(queueName, false, false, false); err != nil {
		t.Fatalf("failed to delete the client queue: %v", err)
	}
	if _, err := rawCh.QueueDeclare(queueName, false, false, false, false, amqp.Table{
		"x-message-ttl": int32(60000),
	}); err != nil {
		t.Fatalf("failed to install the conflicting queue declaration: %v", err)
	}

	waitUntil(t, time.Second*30, "two failed restarts to escalate", func() bool {
		return logs.count("could not be restarted after") >= 1
	})

	if got := logs.count("restart attempt"); got != 2 {
		t.Fatalf("expected one warning per in-place attempt (2), got %v", got)
	}
	if got := logs.count("could not be restarted after"); got != 1 {
		t.Fatalf("expected one escalation for two failed attempts, got %v", got)
	}

	// the supervisor retries the connection rebuild, but the slot must not log
	// another escalation line while it waits
	time.Sleep(time.Second)
	if got := logs.count("could not be restarted after"); got != 1 {
		t.Fatalf("the slot repeated its escalation line while waiting: %v", got)
	}

	// remove the conflict; the next rebuild re-declares the durable queue and
	// resumes the listener on the new generation
	if _, err := rawCh.QueueDelete(queueName, false, false, false); err != nil {
		t.Fatalf("failed to remove the conflicting queue: %v", err)
	}

	waitUntil(t, time.Second*25, "recovery after the conflicting declaration is removed", func() bool {
		m := module()
		m.mu.Lock()
		current := m.gen
		m.mu.Unlock()
		return current != nil && current != gen0 && current.conn != nil &&
			!current.conn.IsClosed() && RabbitConnected()
	})

	payload := "after-escalation-recovery-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := PublishText(rail, payload, "", queueName); err != nil {
		t.Fatalf("failed to publish after recovery: %v", err)
	}
	assertSamePayloadSet(t, []string{payload}, collectRabbitPayloads(t, received, 1, time.Second*15))

	if got := logs.count("could not be restarted after"); got != 1 {
		t.Fatalf("expected exactly one escalation across the recovery, got %v", got)
	}
}

// TestIntegrationPublisherChannelDeathLeavesConnection proves publisher channels
// are pool entries, not supervised resources: when every pooled channel is
// already dead, publishing drops them, opens a replacement on demand and never
// rebuilds the connection.
func TestIntegrationPublisherChannelDeathLeavesConnection(t *testing.T) {
	requireRabbitBroker(t)
	prepareRabbitIntegrationTest(t)

	queueName := rabbitTestName("it-pub-death-")
	RegisterRabbitQueue(QueueRegistration{Name: queueName, Durable: true})

	// tag this module's connection so the management API can count it
	connectionName := rabbitTestName("it-pub-death-conn-")
	previousAppName := miso.GetPropStr(miso.PropAppName)
	miso.SetProp(miso.PropAppName, connectionName)
	t.Cleanup(func() { miso.SetProp(miso.PropAppName, previousAppName) })

	rail := miso.EmptyRail()
	if err := StartRabbitMqClient(rail); err != nil {
		t.Fatalf("failed to start rabbitmq client: %v", err)
	}
	gen0 := currentRabbitGeneration(t)

	// empty the pool, close every bootstrap channel, then put the dead channels
	// back, which is what a generation looks like after a broker-side kill
	var dead []*amqp.Channel
	for {
		ch, ok := gen0.publisher.Pop()
		if !ok {
			break
		}
		dead = append(dead, ch)
	}
	if len(dead) == 0 {
		t.Fatal("expected bootstrap publisher channels in the pool")
	}
	for _, ch := range dead {
		// a close may fail with ErrClosed once the connection itself has gone,
		// which is itself the failure mode this test guards against
		_ = ch.Close()
		gen0.publisher.Push(ch)
	}

	// a dead publisher channel must not wake the connection supervisor
	time.Sleep(time.Millisecond * 300)
	if got := currentRabbitGeneration(t); got != gen0 {
		t.Fatal("a publisher channel death rebuilt the connection")
	}
	select {
	case <-gen0.escalateSignal:
		t.Fatal("a publisher channel death escalated to a connection rebuild")
	default:
	}

	connectionsBefore := countRabbitModuleConnections(t, connectionName)
	if connectionsBefore == 0 {
		waitUntil(t, time.Second*10, "the module connection to register with the broker", func() bool {
			return countRabbitModuleConnections(t, connectionName) >= 1
		})
		connectionsBefore = countRabbitModuleConnections(t, connectionName)
	}

	// every borrow path sees only dead channels: the pool drops them and opens a
	// confirmed replacement instead of failing the publish
	for i := 0; i < 2; i++ {
		payload := fmt.Sprintf("pub-death-%d-%d", i, time.Now().UnixNano())
		if err := PublishText(rail, payload, "", queueName); err != nil {
			t.Fatalf("publish %d through a dead pool failed: %v", i, err)
		}
	}

	if got := currentRabbitGeneration(t); got != gen0 {
		t.Fatal("publishing through a dead pool rebuilt the connection")
	}
	// the management API registers connections with a small lag, so give a
	// reconnect time to show up before asserting the count is unchanged
	time.Sleep(time.Second * 2)
	if n := countRabbitModuleConnections(t, connectionName); n != connectionsBefore {
		t.Fatalf("publishing through a dead pool changed the connection count: %d -> %d", connectionsBefore, n)
	}

	// both publishes reached the broker through the replacement channel
	rawConn := dialRawRabbit(t)
	rawCh, err := rawConn.Channel()
	if err != nil {
		t.Fatalf("failed to open the raw channel: %v", err)
	}
	defer rawCh.Close()
	for i := 0; i < 2; i++ {
		var delivered bool
		waitUntil(t, time.Second*10, "the published message to be consumable", func() bool {
			_, ok, err := rawCh.Get(queueName, true)
			if err != nil {
				t.Fatalf("failed to read from the queue: %v", err)
			}
			delivered = ok
			return ok
		})
		if !delivered {
			t.Fatalf("message %d never reached the broker", i)
		}
	}
}

// rabbitTestLogWriter captures framework logging so integration tests can
// observe supervisor retries and the channel/connection path that fired.
type rabbitTestLogWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *rabbitTestLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *rabbitTestLogWriter) count(sub string) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return strings.Count(w.buf.String(), sub)
}

func (w *rabbitTestLogWriter) contains(sub string) bool {
	return w.count(sub) > 0
}

func captureRabbitTestLogs(t *testing.T) *rabbitTestLogWriter {
	t.Helper()
	w := new(rabbitTestLogWriter)
	previous := miso.GetLogrusLogger().Out
	miso.SetLogOutput(io.MultiWriter(previous, w))
	t.Cleanup(func() { miso.SetLogOutput(previous) })
	return w
}

func dialRawRabbit(t *testing.T) *amqp.Connection {
	t.Helper()
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Fatalf("failed to open the raw broker connection: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// rabbitQueueExists uses a fresh channel per probe because a passive declare of
// a missing queue closes the probing channel with 404.
func rabbitQueueExists(conn *amqp.Connection, name string) bool {
	ch, err := conn.Channel()
	if err != nil {
		return false
	}
	defer ch.Close()
	_, err = ch.QueueDeclarePassive(name, true, false, false, false, nil)
	return err == nil
}

// rabbitManagementConnection is the subset of /api/connections the tests need.
type rabbitManagementConnection struct {
	Name             string         `json:"name"`
	ClientProperties map[string]any `json:"client_properties"`
}

// getRabbitManagement decodes a management API document into target.
func getRabbitManagement(t *testing.T, path string, target any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://localhost:15672"+path, nil)
	if err != nil {
		t.Fatalf("failed to build the management request: %v", err)
	}
	req.SetBasicAuth("guest", "guest")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to query the management API: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("management API returned %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("failed to decode the management response: %v", err)
	}
}

func fetchRabbitManagementConnections(t *testing.T) []rabbitManagementConnection {
	t.Helper()
	var conns []rabbitManagementConnection
	getRabbitManagement(t, "/api/connections", &conns)
	return conns
}

// countRabbitModuleConnections counts broker connections whose
// connection_name matches the supplied value. The management API is used because
// rabbitmqctl is not available here; in this broker closed connections leave the
// list immediately, though registration of a new one can lag ~1.5s.
func countRabbitModuleConnections(t *testing.T, connectionName string) int {
	t.Helper()
	count := 0
	for _, c := range fetchRabbitManagementConnections(t) {
		if name, ok := c.ClientProperties["connection_name"].(string); ok && name == connectionName {
			count++
		}
	}
	return count
}

// countRabbitModuleChannels counts the channels opened on the broker connection
// whose client property connection_name matches the supplied value. The
// management API identifies a channel's owner by broker connection name, so the
// connection list is consulted first to map the app-level name.
func countRabbitModuleChannels(t *testing.T, connectionName string) int {
	t.Helper()
	brokerName := ""
	for _, c := range fetchRabbitManagementConnections(t) {
		if name, ok := c.ClientProperties["connection_name"].(string); ok && name == connectionName {
			brokerName = c.Name
			break
		}
	}
	if brokerName == "" {
		return 0
	}

	var channels []map[string]any
	getRabbitManagement(t, "/api/channels", &channels)

	count := 0
	for _, ch := range channels {
		details, ok := ch["connection_details"].(map[string]any)
		if !ok {
			// the management API reports a closed bootstrap channel with an
			// empty connection_details list and no owner; it is not this
			// connection's channel
			continue
		}
		if details["name"] == brokerName {
			count++
		}
	}
	return count
}

// TestIntegrationRecoversFromFailedEstablish covers the defect-A shape: a
// rebuild whose establish fails at declare time must retire the half-built
// connection, keep retrying with backoff, stop any abandoned consumer, and
// recover once the conflicting declaration is gone.
func TestIntegrationRecoversFromFailedEstablish(t *testing.T) {
	requireRabbitBroker(t)

	// a 100ms base leaves the test time to install the conflict after the queue
	// delete cancels the consumer, while keeping retries observable.
	base, max := reconnectBaseDelay, reconnectMaxDelay
	reconnectBaseDelay = time.Millisecond * 100
	reconnectMaxDelay = time.Millisecond * 500
	t.Cleanup(func() { reconnectBaseDelay, reconnectMaxDelay = base, max })

	prepareRabbitIntegrationTest(t)
	logs := captureRabbitTestLogs(t)

	queueName := rabbitTestName("it-failed-establish-")
	RegisterRabbitQueue(QueueRegistration{Name: queueName, Durable: true})

	received := make(chan string, 16)
	AddRabbitListener(MsgListener{
		QueueName: queueName,
		Handler: func(rail miso.Rail, payload string) error {
			received <- payload
			return nil
		},
	})

	// tag this module's connections so the broker can be asked how many of them
	// are alive; the raw test connections stay unnamed.
	connectionName := rabbitTestName("it-failed-establish-conn-")
	previousAppName := miso.GetPropStr(miso.PropAppName)
	miso.SetProp(miso.PropAppName, connectionName)
	t.Cleanup(func() { miso.SetProp(miso.PropAppName, previousAppName) })

	rail := miso.EmptyRail()
	if err := StartRabbitMqClient(rail); err != nil {
		t.Fatalf("failed to start rabbitmq client: %v", err)
	}
	gen0 := currentRabbitGeneration(t)

	rawConn := dialRawRabbit(t)
	rawCh, err := rawConn.Channel()
	if err != nil {
		t.Fatalf("failed to open the raw channel: %v", err)
	}
	defer rawCh.Close()

	baselineGoroutines := runtime.NumGoroutine()

	// Replace the client's durable queue with a conflicting, non-durable one, so
	// the next establish fails at declare time with 406 PRECONDITION_FAILED.
	if _, err := rawCh.QueueDelete(queueName, false, false, false); err != nil {
		t.Fatalf("failed to delete the client queue: %v", err)
	}
	if _, err := rawCh.QueueDeclare(queueName, false, false, false, false, amqp.Table{
		"x-message-ttl": int32(60000),
	}); err != nil {
		t.Fatalf("failed to install the conflicting queue declaration: %v", err)
	}

	// deleting the queue already cancels the consumer; closing the connection is
	// the explicit rebuild trigger, and may be a no-op if the cancel got there first.
	_ = gen0.conn.Close()

	waitUntil(t, time.Second*25, "the first failed establish attempt", func() bool {
		return logs.count("Failed to reconnect to RabbitMQ") >= 1
	})

	// Every failed establish must retire the connection it dialed. Count the
	// module's live broker connections now, then again two failed attempts later;
	// a leak would grow this count once per attempt.
	firstFailures := logs.count("Failed to reconnect to RabbitMQ")
	startsBefore := logs.count("Establish connection to RabbitMQ")
	connectionsBefore := countRabbitModuleConnections(t, connectionName)

	waitUntil(t, time.Minute, "two more failed establish attempts", func() bool {
		return logs.count("Failed to reconnect to RabbitMQ") >= firstFailures+2
	})

	// Retries are serialized (one dial in flight), not a hot loop: two more
	// failures allow a handful of attempt starts at most.
	attemptStarts := logs.count("Establish connection to RabbitMQ") - startsBefore
	if attemptStarts > 6 {
		t.Fatalf("supervisor hot-looped on failed establish: %v attempt starts for 2 failures", attemptStarts)
	}

	// sample several times to tolerate registration lag for the in-flight attempt
	connectionsAfter := 0
	for i := 0; i < 6; i++ {
		if n := countRabbitModuleConnections(t, connectionName); n > connectionsAfter {
			connectionsAfter = n
		}
		time.Sleep(time.Millisecond * 300)
	}
	t.Logf("failed-establish connections: connection_name=%s before=%d afterMax=%d attemptStarts=%d",
		connectionName, connectionsBefore, connectionsAfter, attemptStarts)
	if connectionsBefore > 1 {
		t.Fatalf("expected at most one live module connection during failed establishes, got %d", connectionsBefore)
	}
	if connectionsAfter > connectionsBefore+1 {
		t.Fatalf("failed establish leaked broker connections: before=%d afterMax=%d", connectionsBefore, connectionsAfter)
	}

	// the half-built generation must not be installed or report connected
	if RabbitConnected() {
		t.Fatal("module reports connected while establish keeps failing")
	}
	m := module()
	m.mu.Lock()
	gen := m.gen
	m.mu.Unlock()
	if gen != nil && gen != gen0 && gen.conn != nil && !gen.conn.IsClosed() {
		t.Fatal("a failed establish installed a live generation")
	}
	// each failed attempt must retire its dialed connection and declaration channel
	if n := runtime.NumGoroutine(); n > baselineGoroutines+15 {
		t.Fatalf("failed establishes leaked goroutines, baseline %v, now %v", baselineGoroutines, n)
	}

	// no abandoned listener: a message published to the conflicting queue must
	// stay queued while every establish is failing.
	sentinel := "sentinel-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := rawCh.PublishWithContext(context.Background(), "", queueName, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(sentinel),
	}); err != nil {
		t.Fatalf("failed to publish the sentinel: %v", err)
	}
	select {
	case payload := <-received:
		t.Fatalf("an abandoned listener consumed %q while establish was failing", payload)
	case <-time.After(time.Millisecond * 400):
	}

	// remove the conflict; the next attempt re-declares the durable queue and
	// re-registers the consumer.
	if _, err := rawCh.QueueDelete(queueName, false, false, false); err != nil {
		t.Fatalf("failed to remove the conflicting queue: %v", err)
	}

	waitUntil(t, time.Second*25, "recovery after the conflicting declaration is removed", func() bool {
		m.mu.Lock()
		current := m.gen
		m.mu.Unlock()
		return current != nil && current != gen0 && current.conn != nil &&
			!current.conn.IsClosed() && RabbitConnected()
	})

	payload := "after-recovery-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := PublishText(rail, payload, "", queueName); err != nil {
		t.Fatalf("failed to publish after recovery: %v", err)
	}
	assertSamePayloadSet(t, []string{payload}, collectRabbitPayloads(t, received, 1, time.Second*15))
}

// TestIntegrationChannelIsolationKeepsSiblingsConsuming proves a broker-side
// channel death is isolated: deleting one consumer's queue makes the broker
// cancel that one channel while the connection stays up. The slot re-opens on
// the same connection, re-declares the queue and drains it, and a sibling queue
// keeps consuming throughout without the broker seeing a reconnect.
func TestIntegrationChannelIsolationKeepsSiblingsConsuming(t *testing.T) {
	requireRabbitBroker(t)
	withFastReconnect(t)
	prepareRabbitIntegrationTest(t)
	logs := captureRabbitTestLogs(t)

	queueName := rabbitTestName("it-iso-")
	siblingQueue := rabbitTestName("it-iso-sibling-")
	RegisterRabbitQueue(QueueRegistration{Name: queueName, Durable: true})
	RegisterRabbitQueue(QueueRegistration{Name: siblingQueue, Durable: true})

	received := make(chan string, 16)
	AddRabbitListener(MsgListener{
		QueueName: queueName,
		Handler: func(rail miso.Rail, payload string) error {
			received <- payload
			return nil
		},
	})
	siblingReceived := make(chan string, 16)
	AddRabbitListener(MsgListener{
		QueueName: siblingQueue,
		Handler: func(rail miso.Rail, payload string) error {
			siblingReceived <- payload
			return nil
		},
	})

	// tag this module's connection so the management API can count it
	connectionName := rabbitTestName("it-iso-conn-")
	previousAppName := miso.GetPropStr(miso.PropAppName)
	miso.SetProp(miso.PropAppName, connectionName)
	t.Cleanup(func() { miso.SetProp(miso.PropAppName, previousAppName) })

	rail := miso.EmptyRail()
	if err := StartRabbitMqClient(rail); err != nil {
		t.Fatalf("failed to start rabbitmq client: %v", err)
	}
	gen0 := currentRabbitGeneration(t)
	waitUntil(t, time.Second*10, "the module connection to register with the broker", func() bool {
		return countRabbitModuleConnections(t, connectionName) >= 1
	})
	connectionsBefore := countRabbitModuleConnections(t, connectionName)
	if connectionsBefore != 1 {
		t.Fatalf("expected one module connection before the test, got %d", connectionsBefore)
	}

	// baseline channel count: the publisher pool plus one channel per listener.
	// Wait for every channel to register so a lagging baseline cannot hide a leak.
	expectedChannels := miso.GetPropInt(PropRabbitMqPublisherChanPoolSize) + 2
	waitUntil(t, time.Second*10, "all module channels to register with the broker", func() bool {
		return countRabbitModuleChannels(t, connectionName) >= expectedChannels
	})
	channelsBefore := countRabbitModuleChannels(t, connectionName)
	if channelsBefore != expectedChannels {
		t.Fatalf("expected %d module channels before the test, got %d", expectedChannels, channelsBefore)
	}

	first := "before-cancel-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := PublishText(rail, first, "", queueName); err != nil {
		t.Fatalf("failed to publish before the cancellation: %v", err)
	}
	assertSamePayloadSet(t, []string{first}, collectRabbitPayloads(t, received, 1, time.Second*15))

	rawConn := dialRawRabbit(t)
	rawCh, err := rawConn.Channel()
	if err != nil {
		t.Fatalf("failed to open the raw channel: %v", err)
	}
	defer rawCh.Close()

	if _, err := rawCh.QueueDelete(queueName, false, false, false); err != nil {
		t.Fatalf("failed to delete the consumed queue: %v", err)
	}
	// the sibling keeps consuming while the affected slot restarts
	siblingPayload := "sibling-during-restart-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := PublishText(rail, siblingPayload, "", siblingQueue); err != nil {
		t.Fatalf("failed to publish to the sibling queue: %v", err)
	}
	assertSamePayloadSet(t, []string{siblingPayload},
		collectRabbitPayloads(t, siblingReceived, 1, time.Second*15))

	// the slot re-declares the registered queue instead of waiting for a rebuild
	waitUntil(t, time.Second*10, "the consumer to re-declare the queue", func() bool {
		return rabbitQueueExists(rawConn, queueName)
	})

	// the connection generation must not be replaced by a channel-level death
	if current := currentRabbitGeneration(t); current != gen0 {
		t.Fatal("a consumer channel death replaced the connection generation")
	}
	if logs.contains("RabbitMQ connection closed, reconnecting") {
		t.Fatal("a consumer channel death rebuilt the connection")
	}
	if got := logs.count("could not be restarted"); got != 0 {
		t.Fatalf("a recoverable channel death escalated %v times", got)
	}

	second := "after-restart-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := PublishText(rail, second, "", queueName); err != nil {
		t.Fatalf("failed to publish after the restart: %v", err)
	}
	assertSamePayloadSet(t, []string{second}, collectRabbitPayloads(t, received, 1, time.Second*15))

	// the broker never saw a reconnect; give the management API time to catch up
	time.Sleep(time.Second * 2)
	if n := countRabbitModuleConnections(t, connectionName); n != connectionsBefore {
		t.Fatalf("channel isolation changed the broker connection count: %d -> %d", connectionsBefore, n)
	}

	// a cancelled consumer channel is closed before its replacement opens, so the
	// open channel count returns to the baseline instead of growing by one per
	// restart (a cancel does not close the channel)
	waitUntil(t, time.Second*10, "the broker channel count to return to baseline after the restart", func() bool {
		return countRabbitModuleChannels(t, connectionName) == channelsBefore
	})
}
