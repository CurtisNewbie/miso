package kafka

import (
	"context"
	"errors"
	"io"
	"math"
	"runtime/debug"
	"sync"

	"github.com/curtisnewbie/miso/encoding/json"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/protocol"
)

//lint:ignore U1000 for future use
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
		Order: miso.BootstrapOrderL1,
	})
}

type kafkaModule struct {
	mu            *sync.RWMutex
	w             *kafka.Writer
	readerConfigs []KafkaReaderConfig
}

type KafkaReaderConfig struct {
	Topic       string
	GroupId     string
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
		Logger:                 miso.EmptyRail(),
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
		MaxAttempts:           math.MaxInt, // retry forever?
		MaxBytes:              10e6,        // 10MB
		Logger:                miso.EmptyRail(),
		WatchPartitionChanges: true,
	})
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
		return miso.NewErrf("failed to obtain Kafka Writer")
	}

	// propogate trace through headers
	headers := []kafka.Header{}
	miso.UsePropagationKeys(func(key string) {
		headers = append(headers, protocol.Header{Key: key, Value: []byte(rail.CtxValStr(key))})
	})

	err := w.WriteMessages(context.Background(), kafka.Message{
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
	miso.AddShutdownHook(func() {
		if err := w.Close(); err != nil {
			miso.Warnf("Failed to close kafka Writer, %v", err)
		} else {
			miso.Debug("Kafka Writer closed")
		}
		m.w = nil
	})

	for _, rc := range m.readerConfigs {
		r := NewReader(addrs, rc.GroupId, rc.Topic)

		// wrap provided listener, make sure it's panic free
		listen := func(rail miso.Rail, m Message) (err error) {
			defer func() {
				if v := recover(); v != nil {
					util.PanicLog("panic recovered, %v\n%v", v, util.UnsafeByt2Str(debug.Stack()))
					err = miso.NewErrf("kafka listener panic recovered, %v", v)
				}
			}()

			err = rc.Listen(rail, m)
			return
		}

		miso.AddShutdownHook(func() {
			if err := r.Close(); err != nil {
				miso.Warnf("Failed to close kafka Reader (%v, %v), %v", rc.GroupId, rc.Topic, err)
			} else {
				miso.Debugf("Kafka Reader closed (%v, %v)", rc.GroupId, rc.Topic)
			}
		})
		if rc.Concurrency < 1 {
			rc.Concurrency = 1
		}
		for i := 0; i < rc.Concurrency; i++ {
			go func() {
				for {
					rail := miso.EmptyRail()
					km, err := r.FetchMessage(rail.Context())
					if err != nil {
						if errors.Is(err, io.EOF) {
							rail.Infof("Kafka Reader for (%v, %v) closed, exiting", rc.GroupId, rc.Topic)
							return
						}
						rail.Errorf("Failed to read Kafka message, %v", err)
						return
					}

					// retriving trace info from headers
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
						rail.Errorf("Failed to commit Kafka message (%v, %v), offset: %v, %v", rc.GroupId, rc.Topic, km.Offset, err)
						continue
					}

					rail.Infof("Kafka message commited at topic: %v, partition: %v, offset: %v", km.Topic, km.Partition, km.Offset)
				}
			}()
		}

		rail.Infof("Created Kafka Reader for (%v, %v)", rc.GroupId, rc.Topic)
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
