# Kafka

Kafka producer/consumer integration with trace propagation via message headers.

**Package:** `github.com/curtisnewbie/miso/middleware/kafka`

## Configuration

```yaml
kafka:
  enabled: true
  server.addr: ["localhost:9092"]  # broker addresses
  start.offset: last               # first | last (new consumer groups only, default: last)
  write.timeout: 5s                # producer write timeout (default: 5s)
```

`kafka.start.offset` only applies to consumer groups without committed offsets. Default `last` prevents the first deployment of a consumer group from replaying the entire topic history.

## Producing Messages

```go
import "github.com/curtisnewbie/miso/middleware/kafka"

// JSON value (auto-marshaled), trace headers auto-propagated via kafka headers
err := kafka.WriteMessageJson(rail, "user.events", "user-123", UserEvent{Name: "John"})

// Raw bytes value
err := kafka.WriteMessage(rail, "user.events", "user-123", []byte("payload"))
```

- `GetWriter()` — the managed `*kafka.Writer` (round-robin balancer, `RequireOne` ack, auto topic creation, 5s write timeout)
- `NewWriter(addrs)` / `NewReader(addrs, groupId, topic)` — unmanaged writers/readers for custom use
- Trace context (`X-B3-TraceId`, `X-B3-SpanId`, `x-username`, etc.) is automatically attached to message headers on write and restored in the listener's Rail

## Consuming Messages

```go
kafka.AddKafkaListener(kafka.KafkaReaderConfig{
    Topic:       "user.events",
    GroupId:     "user-events-consumer",
    Concurrency: 3, // number of reader goroutines (default: 1)
    Listen: func(rail miso.Rail, m kafka.Message) error {
        rail.Infof("Received event, key: %s", m.Key)
        // parse value: m.Value ([]byte), headers: m.Headers (map[string][]byte)
        var evt UserEvent
        json.Unmarshal(m.Value, &evt)
        return nil
    },
})
```

### Key Semantics

- **Commit only on success** — the message offset is committed only after `Listen` returns `nil`. If `Listen` returns an error or panics, the message is **not** committed and is only redelivered after a rebalance or process restart (no automatic retry loop).
- **Partition-ordered processing** — Kafka partitions are exclusively assigned to one reader, so messages within a partition are always processed and committed in order. `Concurrency` creates that many consumer-group readers and effectively limits how many partitions are processed in parallel — don't set it higher than the topic's partition count (extra readers sit idle).
- **Consumer group compatibility** — miso uses kafka-go internally, which doesn't support `CooperativeStickyAssigner`/`StickyPartitioner` (used by default in Java/C++ clients). Group consumers created here are not compatible with clients written in other languages — don't share the same topic with different-language clients.
