package redis

import (
	"context"
	"sync"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/async"
	"github.com/curtisnewbie/miso/util/json"
)

type rtopicMessage[T any] struct {
	Headers map[string]string `json:"headers"`
	Payload T                 `json:"payload"`
}

type rtopic[T any] struct {
	topic string
}

func (p *rtopic[T]) SubscribeSync(handler func(rail miso.Rail, evt T) error) (context.CancelFunc, error) {
	pubsub := GetRedis().Subscribe(context.Background(), p.topic)
	var once sync.Once
	cancel := func() { once.Do(func() { pubsub.Close() }) }
	miso.AddShutdownHook(cancel)

	ch := pubsub.Channel()
	go func() {
		for m := range ch {
			rail := miso.EmptyRail()
			pm, err := json.SParseJsonAs[rtopicMessage[T]](m.Payload)
			if err != nil {
				rail.Errorf("Failed to handle redis channle message, topic: %v, %v", p.topic, err)
				continue
			}
			rail = miso.LoadPropagationKeysFromHeaders(rail, pm.Headers)
			rail.Infof("Receive redis channel message, topic: %v, payload: %#v", p.topic, pm.Payload)
			if err := handler(rail, pm.Payload); err != nil {
				rail.Errorf("Failed to handle redis channle message, topic: %v, %v", p.topic, err)
			}
		}
	}()
	return cancel, nil
}

func (p *rtopic[T]) Subscribe(pool async.AsyncPool, handler func(rail miso.Rail, evt T) error) error {
	_, err := p.SubscribeSync(func(rail miso.Rail, evt T) error {
		pool.Go(func() {
			if err := handler(rail, evt); err != nil {
				rail.Errorf("Failed to handle redis channle message, topic: %v, %v", p.topic, err)
			}
		})
		return nil
	})
	return err
}

func (p *rtopic[T]) Publish(rail miso.Rail, t T) error {
	m := rtopicMessage[T]{
		Headers: miso.BuildTraceHeadersStr((rail)),
		Payload: t,
	}
	ms, err := json.SWriteJson(m)
	if err != nil {
		return err
	}
	return errs.Wrap(GetRedis().Publish(rail.Context(), p.topic, ms).Err())
}

func NewTopic[T any](topic string) *rtopic[T] {
	return &rtopic[T]{
		topic: topic,
	}
}

// RSignalTopic is a pub/sub topic that carries no payload — publish is a fire-and-forget
// signal, and subscribers receive only a Rail for tracing.
//
// Useful for cross-instance broadcast signals (e.g. stop, refresh, invalidate).
//
// Example:
//
//	var stopTopic = redis.NewSignalTopic("myapp:stop")
//
//	// publisher
//	stopTopic.Signal(rail)
//
//	// subscriber — cancel rail context when signal is received
//	rail, cancel := stopTopic.OnSignalCancel(rail)
//	defer cancel()
type RSignalTopic struct {
	inner *rtopic[struct{}]
}

// NewSignalTopic creates a new RSignalTopic bound to the given Redis pub/sub channel key.
func NewSignalTopic(topic string) RSignalTopic {
	return RSignalTopic{inner: NewTopic[struct{}](topic)}
}

// Signal publishes a signal to all subscribers on this topic.
func (s RSignalTopic) Signal(rail miso.Rail) error {
	return s.inner.Publish(rail, struct{}{})
}

// OnSignalCancel returns a Rail derived from rail with a cancel func attached.
// When a signal is received on this topic, the cancel func is called, cancelling the returned Rail's context.
// The returned CancelFunc also unsubscribes from the topic — always defer it.
func (s RSignalTopic) OnSignalCancel(rail miso.Rail) (miso.Rail, context.CancelFunc) {
	child, cancelRail := rail.WithCancel()
	unsubscribe, _ := s.inner.SubscribeSync(func(_ miso.Rail, _ struct{}) error {
		cancelRail()
		return nil
	})
	return child, func() {
		cancelRail()
		unsubscribe()
	}
}
