package redis

import (
	"context"

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

func (p *rtopic[T]) Subscribe(pool async.AsyncPool, handler func(rail miso.Rail, evt T) error) error {

	pubsub := GetRedis().Subscribe(context.Background(), p.topic)
	miso.AddShutdownHook(func() { pubsub.Close() })

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

			// redis subscription cannot be blocked for more than 30s, have to handle these asynchronously
			pool.Go(func() {
				if err := handler(rail, pm.Payload); err != nil {
					rail.Errorf("Failed to handle redis channle message, topic: %v, %v", p.topic, err)
				}
			})
		}
	}()
	return nil
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
