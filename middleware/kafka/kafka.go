package kafka

import (
	"context"
	"errors"
	"io"
	"math"
	"runtime/debug"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/json"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/segmentio/kafka-go"
)

var mod = miso.InitAppModuleFunc(func() *kafkaModule {
	return &kafkaModule{
		mu:            &sync.RWMutex{},
		readerConfigs: make([]KafkaReaderConfig, 0),
	}
})

func init() {
	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name: "Bootstrap Kafka",
		Bootstrap: func(rail miso.Rail) error {
			return bootstrapKafka(rail)
		},
		Condition: func(rail miso.Rail) (bool, error) {
			return miso.GetPropBool(PropKafkaEnabled), nil
		},
		Order: miso.BootstrapOrderL4,
	})
}

type kafkaModule struct {
	mu            *sync.RWMutex
	w             *kafka.Writer
	readerConfigs []KafkaReaderConfig
}

type KafkaReaderConfig struct {
	Topic   string
	GroupId string
	// Number of Kafka Readers (consumer group members) to create, each consuming with a single
	// goroutine. Defaults to 1. Partitions are exclusively assigned to each member, so messages
	// within a partition are always processed in order. Should not exceed the topic's partition count.
	Concurrency int
	Listen      func(rail miso.Rail, m Message) error
}

// Create new Kafka Writer using default Transport.
//
// The created Writer is not managed by miso.
func NewWriter(addrs []string) (*kafka.Writer, error) {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(addrs...),
		Balancer:               &kafka.RoundRobin{},
		RequiredAcks:           kafka.RequireOne,
		AllowAutoTopicCreation: true,
		Logger:                 kafkaInfoLogger{},
		ErrorLogger:            kafkaErrorLogger{},
	}
	return w, nil
}

// Create new Kafka Reader using default Transport.
//
// The created Reader is not managed by miso.
//
// Notice that, internally miso uses kafka-go, which doesn't support CooperativeStickyAssigner and StickyPartitioner, while these are used by default in cpp and java client.
//
// This means the group consumer created here will not be compatible with other clients written in different languages. Just don't share the same topic with different clients.
func NewReader(addrs []string, groupId string, topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:               addrs,
		GroupID:               groupId,
		Topic:                 topic,
		StartOffset:           startOffset(), // only applies to consumer groups without committed offsets
		MaxAttempts:           math.MaxInt,   // retry forever
		MaxBytes:              10e6,          // 10MB
		Logger:                kafkaInfoLogger{},
		ErrorLogger:           kafkaErrorLogger{},
		WatchPartitionChanges: true,
	})
}

// Resolve the start offset for new consumer groups, defaults to the latest offset
// so that first deployment of a consumer group does not replay the entire topic history.
func startOffset() int64 {
	switch miso.GetPropStr(PropKafkaStartOffset) {
	case "first":
		return kafka.FirstOffset
	default:
		return kafka.LastOffset
	}
}

func WriteMessageJson(rail miso.Rail, topic string, key string, value any) error {
	byt, err := json.WriteJson(value)
	if err != nil {
		return err
	}
	return WriteMessage(rail, topic, key, byt)
}

func WriteMessage(rail miso.Rail, topic string, key string, value []byte) error {
	w := GetWriter()
	if w == nil {
		return errs.NewErrf("failed to obtain Kafka Writer")
	}

	// propagate trace through headers
	headers := []kafka.Header{}
	miso.UsePropagationKeys(func(key string) {
		headers = append(headers, kafka.Header{Key: key, Value: []byte(rail.CtxValStr(key))})
	})

	// bound the write so a slow or unreachable broker cannot stall the caller indefinitely.
	// note: a timed-out write may still be delivered by the broker, so the returned error
	// does not guarantee the message was not written — callers retrying may produce duplicates
	ctx, cancel := context.WithTimeout(context.Background(), miso.GetPropDuration(PropKafkaWriteTimeout))
	defer cancel()

	err := w.WriteMessages(ctx, kafka.Message{
		Topic:   topic,
		Headers: headers,
		Key:     []byte(key),
		Value:   value,
	})
	return err
}

// Get the managed Kafka writer.
func GetWriter() *kafka.Writer {
	m := mod()
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.w
}

// Register Kafka Listener.
//
// The message is committed only after Listen returns nil. If Listen returns an error or panics,
// the message is not committed and is only redelivered after a rebalance or process restart.
//
// Concurrency Kafka Readers (consumer group members) are created for this listener, each running
// a single goroutine. Kafka partitions are exclusively assigned to one member, so messages within
// a partition are always processed and committed in order. Concurrency effectively limits how many
// partitions are processed in parallel — values greater than the topic's partition count leave some
// members idle, so it should not exceed the partition count.
//
// Notice that, internally miso uses kafka-go, which doesn't support CooperativeStickyAssigner and StickyPartitioner, while these are used by default in cpp and java client.
//
// This means the group consumer created here will not be compatible with other clients written in different languages. Just don't share the same topic with different clients.
func AddKafkaListener(c KafkaReaderConfig) {
	m := mod()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readerConfigs = append(m.readerConfigs, c)
}

func bootstrapKafka(rail miso.Rail) error {
	m := mod()
	m.mu.Lock()
	defer m.mu.Unlock()

	addrs := miso.GetPropStrSlice(PropKafkaServerAddr)
	rail.Infof("Connecting to kafka: %v", addrs)

	w, err := NewWriter(addrs)
	if err != nil {
		return err
	}
	m.w = w
	miso.AddAsyncShutdownHook(func() {
		if err := w.Close(); err != nil {
			miso.Warnf("Failed to close kafka Writer, %v", err)
		} else {
			miso.Debug("Kafka Writer closed")
		}
		m.mu.Lock()
		m.w = nil
		m.mu.Unlock()
	})

	for _, rc := range m.readerConfigs {
		// wrap provided listener, make sure it's panic free
		listen := func(rail miso.Rail, m Message) (err error) {
			defer func() {
				if v := recover(); v != nil {
					miso.Errorf("panic recovered, %v\n%v", v, strutil.UnsafeByt2Str(debug.Stack()))
					err = errs.NewErrf("kafka listener panic recovered, %v", v)
				}
			}()

			err = rc.Listen(rail, m)
			return
		}

		conc := rc.Concurrency
		if conc < 1 {
			conc = 1
		}
		for i := 0; i < conc; i++ {
			r := NewReader(addrs, rc.GroupId, rc.Topic)

			miso.AddAsyncShutdownHook(func() {
				if err := r.Close(); err != nil {
					miso.Warnf("Failed to close kafka Reader (%v, %v), %v", rc.GroupId, rc.Topic, err)
				} else {
					miso.Debugf("Kafka Reader closed (%v, %v)", rc.GroupId, rc.Topic)
				}
			})

			go func() {
				backoff := 1 * time.Second
				for {
					rail := miso.EmptyRail()
					km, err := r.FetchMessage(rail.Context())
					if err != nil {
						if errors.Is(err, io.EOF) {
							rail.Infof("Kafka Reader for (%v, %v) closed, exiting", rc.GroupId, rc.Topic)
							return
						}
						// errors surfaced by the reader (e.g. unhandled broker error codes) would
						// otherwise kill this consumer permanently, retry with backoff instead
						rail.Errorf("Failed to read Kafka message (%v, %v), retrying in %v, %v", rc.GroupId, rc.Topic, backoff, err)
						time.Sleep(backoff)
						if backoff < 30*time.Second {
							backoff *= 2
						}
						continue
					}
					backoff = 1 * time.Second

					// retrieving trace info from headers
					tracedHeaders := map[string]string{}
					for _, k := range miso.GetPropagationKeys() {
						tracedHeaders[k] = ""
					}
					for _, h := range km.Headers {
						if _, ok := tracedHeaders[h.Key]; ok {
							tracedHeaders[h.Key] = string(h.Value)
						}
					}
					for k, v := range tracedHeaders {
						if v == "" {
							continue
						}
						rail = rail.WithCtxVal(k, v)
					}

					m := Message{}
					m.load(km)

					if err := listen(rail, m); err != nil {
						rail.Errorf("Failed to handle Kafka message (%v, %v), %v", rc.GroupId, rc.Topic, err)
						continue
					}

					if err := r.CommitMessages(rail.Context(), km); err != nil {
						if errors.Is(err, io.ErrClosedPipe) {
							return // reader closed, shutting down
						}
						rail.Errorf("Failed to commit Kafka message (%v, %v), offset: %v, %v", rc.GroupId, rc.Topic, km.Offset, err)
						time.Sleep(backoff) // avoid hot-looping on persistent commit rejection
						continue
					}

					rail.Infof("Kafka message committed at topic: %v, partition: %v, offset: %v", km.Topic, km.Partition, km.Offset)
				}
			}()
		}

		rail.Infof("Created %v Kafka Reader(s) for (%v, %v), one goroutine per reader", conc, rc.GroupId, rc.Topic)
	}

	return nil
}

type Message struct {
	Key     []byte
	Value   []byte
	Headers map[string][]byte
}

func (m *Message) load(km kafka.Message) {
	m.Key = km.Key
	m.Value = km.Value
	m.Headers = make(map[string][]byte, len(km.Headers))
	for _, h := range km.Headers {
		m.Headers[h.Key] = h.Value
	}
}

type kafkaInfoLogger struct {
}

func (k kafkaInfoLogger) Printf(p string, ar ...interface{}) {
	miso.Infof(p, ar...)
}

type kafkaErrorLogger struct {
}

func (k kafkaErrorLogger) Printf(p string, ar ...interface{}) {
	miso.Errorf(p, ar...)
}
