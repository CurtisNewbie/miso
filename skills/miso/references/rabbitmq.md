# RabbitMQ

RabbitMQ producer and consumer integration with confirm-mode publishing, delayed redelivery, and per-consumer channel isolation.

**Package:** `github.com/curtisnewbie/miso/middleware/rabbit`

## Configuration

```yaml
rabbitmq:
  enabled: true                                  # default: false
  host: localhost                                # default: localhost
  port: 5672                                     # default: 5672
  username: guest                                # default: guest
  password: guest                                # default: guest
  vhost: /                                       # no default, an empty value dials "/"
  consumer.qos: 68                               # default: 68, prefetch count per consumer channel
  publisher.channel-pool-size: 20                # default: 20, confirmed channels kept per connection
```

The property constants are `PropRabbitMqEnabled`, `PropRabbitMqHost`, `PropRabbitMqPort`, `PropRabbitMqUsername`, `PropRabbitMqPassword`, `PropRabbitMqVhost`, `PropRabbitMqConsumerQos` and `PropRabbitMqPublisherChanPoolSize` in `middleware/rabbit/conf.go`.

The client bootstraps at order L4 and only when `rabbitmq.enabled` is true. With it false, registering listeners and components is harmless and nothing dials.

`consumer.qos` is the per-channel prefetch count, so it caps how many unacknowledged messages the broker hands one consumer channel. A listener can override it with its own `Qos` field. `publisher.channel-pool-size` is a cache size, not a connection limit.

## Declaring Components

Three registration functions queue a declaration for the next connection generation.

```go
rabbit.RegisterRabbitQueue(rabbit.QueueRegistration{Name: "order.created", Durable: true})
rabbit.RegisterRabbitExchange(rabbit.ExchangeRegistration{Name: "order.events", Kind: "direct", Durable: true})
rabbit.RegisterRabbitBinding(rabbit.BindingRegistration{Queue: "order.created", RoutingKey: "order.#", Exchange: "order.events"})
```

A registration records intent only, so call these before the client starts. The declaration runs once per generation, on a scratch channel that is closed afterwards. A declaration failure fails the generation. The first attempt surfaces the error from `StartRabbitMqClient`, and a later reconnect retries the whole declaration.

An empty `Kind` becomes `direct`. An empty `RoutingKey` becomes `#`. A binding to an exchange that was never registered is still declared, so the broker decides whether it resolves.

Declaring a binding also declares a redelivery queue for it, described under [Redelivery](#redelivery).

The event bus wraps the three registrations into one call. `rabbit.NewEventBus("order.events")` declares a direct exchange and a durable queue under the same name, bound with routing key `#`.

## Listening

`rabbit.AddRabbitListener` registers a listener for one queue. Two types satisfy `RabbitListener`.

```go
rabbit.AddRabbitListener(rabbit.JsonMsgListener[OrderCreated]{
    QueueName:     "order.created",
    NumOfRoutines: 4,
    Qos:           32,
    Handler: func(rail miso.Rail, evt OrderCreated) error {
        return nil
    },
})

rabbit.AddRabbitListener(rabbit.MsgListener{
    QueueName:     "order.raw",
    NumOfRoutines: 1,
    Handler: func(rail miso.Rail, payload string) error {
        return nil
    },
})
```

`JsonMsgListener[T]` parses the body with `json.ParseJson` before calling the handler and returns the parse error as a handler failure. `MsgListener` passes the body through as a string. A custom type only needs the four `RabbitListener` methods, `Queue`, `Handle`, `Concurrency` and `QosSpec`.

`NumOfRoutines` is the number of goroutines reading the queue's delivery stream, and a value below 1 becomes 1. `Qos` overrides `rabbitmq.consumer.qos` for this listener when it is greater than zero.

Delivery runs with manual acknowledgement. A nil return acks the message. An error nacks it after the redelivery step below. A panic inside the handler is recovered, logged with a stack trace, and treated as an error.

Every listener the client knows about gets its own channel on each connection generation. Listeners are snapshotted when a generation is established, so a listener added later joins on the next generation.

The event bus offers a typed shortcut. `rabbit.SubEventBus[T](name, concurrency, handler)` declares the bus if needed and registers a `JsonMsgListener` for it. `rabbit.NewEventPipeline[T](name)` chains the same intent with `.MaxRetry(n)`, `.ListenerQos(n)` and `.Listen(concurrency, handler)`.

## Publishing

```go
err := rabbit.PublishText(rail, "hello", "order.events", "order.created")
err = rabbit.PublishJson(rail, OrderCreated{Id: 1}, "order.events", "order.created")
err = rabbit.PublishJsonHeaders(rail, OrderCreated{Id: 1}, "order.events", "order.created", map[string]any{"tenant": "acme"})
err = rabbit.PublishMsg(rail, []byte("raw"), "order.events", "order.created", "text/plain", nil)
```

All four go through the same path. The body is published as persistent, given a `mq_` prefixed message id, and sent through a channel in confirm mode. Trace keys registered with `miso.UsePropagationKeys` are copied from the rail into the headers.

A publish returns nil only after the broker confirms it. A failed confirmation returns `errMsgNotPublished`, and its message names both possibilities, that the exchange may not exist or that the connection was lost. Publishing to a missing exchange is not an immediate error, because the broker accepts the frame and only the confirm reports the routing problem.

A publish borrows a channel from the current generation's pool and returns it afterwards. A channel that closed during the publish is dropped rather than returned, and the next borrow opens a replacement.

The event bus publishes JSON to the bus exchange with routing key `#`. `PubRetryEventBus` does the same and sets the `miso-rabbitmq-max-retry` header.

## Bootstrap And Shutdown

```go
rail := miso.EmptyRail()
if err := rabbit.StartRabbitMqClient(rail); err != nil {
    return err
}
if rabbit.RabbitConnected() {
    // a generation is installed and its connection is open
}
```

`StartRabbitMqClient` is synchronous for the first connection. It dials, declares every registered component, fills the publisher pool, and opens the first consumer channel of every listener inline. Any failure retires the half-built connection, so a failed start leaves no socket open. After the first generation it returns and a supervisor goroutine owns reconnection.

Calling it twice is safe. A second call while supervising, or after `RabbitDisconnect`, returns nil without touching the connection.

`RabbitConnected` reports whether a generation exists and its connection is not closed. It does not probe the broker, so a half dead connection can still report true.

`RabbitDisconnect` shuts the client down. It marks the module closed, closes the quit channel, retires the generation and waits for the consumer slots, bounded at 5 seconds. It is idempotent.

The L4 bootstrap callback wraps the same lifecycle. It calls `StartRabbitMqClient` and registers an async shutdown hook that calls `close`, so an application that enables the property does not need to call either function.

Retiring a generation closes its `done` channel first, then the connection, then waits for its consumer slots. Closing `done` first stops a slot from opening a channel or escalating against a connection that is already being replaced.

## Connection Lifecycle

One generation owns one connection together with everything built on it, which is its close listener, its escalation signal, its publisher pool, and its consumer slots. A generation belongs to exactly one supervisor goroutine, and only that goroutine replaces the installed one.

The supervisor waits on four events, a connection close, an escalation from a consumer slot, the quit channel, and the rail's done channel. Any of the first two tears the generation down. Reconnect then backs off and calls establish again.

Reconnect backoff starts at 1 second and doubles per failed attempt up to 30 seconds. A generation that lived at least 1 minute resets the delay to 1 second, so a single blip does not leave the next reconnect waiting 30 seconds.

A connection close is logged at error level with the broker's reason. The quiet path, where the close carries no error, logs at info. A shutdown retires the current generation and returns without attempting a reconnect.

## Consumer Channel Lifecycle

A consumer slot owns exactly one channel for the life of its generation, and no two listeners share a channel. A slot watches its channel for both close and cancel and restarts it in place, on the connection it already has, without touching the supervisor or any other listener's channel. One queue's channel dying leaves every sibling consumer subscribed.

A restart attempt closes the channel whose death was observed, waits its backoff, opens a new channel, registers the close and cancel watchers before anything else, re-declares the listener's own broker resources, then applies the quality of service and calls consume.

The re-declaration covers the listener's queue, the bindings that target that queue, the exchanges those bindings reference, and the redelivery queues those bindings create. Declaration is idempotent, so a channel death that left the broker resources intact costs one round trip, and a durable queue deleted by an operator comes back on the next restart. A queue the client never registered cannot be re-declared, and the restart fails until the escalation limit is reached.

Two in-place attempts are allowed, with 250 milliseconds before the first and 500 milliseconds before the second. An attempt fails when opening, preparing or starting the replacement returns an error, or when the replacement dies before staying up for 1 minute.

Two consecutive failures escalate. The slot logs one error and signals the generation's escalation channel, which is cap 1, so several slots escalating at once collapse into one connection rebuild. The escalated slot stops, and the rebuild creates fresh slots on the new connection.

A slot that restarts successfully resets its failure count after the replacement has stayed up 1 minute, so an occasional death never accumulates into an escalation.

A retired generation stops its slots through the `done` channel. A slot that is mid-backoff aborts the wait, opens nothing, and escalates nothing.

## Publisher Channels

Publisher channels are pool entries, not supervised slots. The pool drops a channel that is already closed when it is borrowed, and a borrow that finds the pool drained opens a replacement on demand, making up to seven attempts over roughly 600 milliseconds of backoff before returning the error.

A dead publisher channel therefore never escalates and never rebuilds the connection. Publishing keeps working as long as opening a replacement channel succeeds, and returns an error once the pool and the demand path both fail.

## Redelivery

A failed handler is redelivered after a delay instead of being requeued immediately. Every binding declaration also declares a redelivery queue named `redeliver_<exchange>_<routingKey>_<delay>`, durable, with a 5000 millisecond TTL and a dead letter target of the original exchange and routing key. Nothing consumes it. A message published into it expires and the broker routes it back to the original exchange.

On a handler error the client publishes the body to the default exchange with the redelivery queue name as routing key, then acks the original message. If that publish fails, the message is nacked with requeue set, so the broker redelivers it immediately.

Two headers bound the retry count. `miso-rabbitmq-max-retry` is the maximum number of redeliveries and `miso-rabbitmq-curr-retry` is the number already performed. When the current count reaches the maximum, the message is acked and dropped with an info log, so a poisonous message leaves the queue instead of cycling forever.

A message without a maximum header is redelivered indefinitely. `MaxRetryTimes15Min` is 180, which at the 5000 millisecond delay is roughly 15 minutes. Use it as the maximum when a message should give up after that window.

`rabbit.PubRetryEventBus` sets the maximum header for one event. `EventPipeline.MaxRetry(n)` does the same for a pipeline, and its default of -1 means unlimited.

Delivery is at least once. A channel close returns unacknowledged messages to the queue, and the restarted consumer receives them again. A message that was acked before the close is not redelivered.

## Logging

A channel death logs one warning naming the queue and the cause, either an abnormal close or a broker cancel. A clean close logs at debug.

A failed in-place attempt logs one warning with the queue name and the attempt number. An escalation logs one error. A successful restart logs one debug line with the queue name.

Connection loss logs one error with the broker reason. A reconnect attempt that fails logs one error with the next delay.

A healthy reconnect therefore produces a few lines rather than one line per second.

## Testing

`go test ./middleware/rabbit/` runs without a broker. The connection supervisor and the slot restart cycle take their open, prepare, start, watch, retire and establish operations as functions, so tests drive them with fake generations.

`MISO_RABBIT_BROKER_TESTS=1 go test ./middleware/rabbit/ -count=1 -timeout 600s` adds the live suite against `localhost:5672` with `guest` credentials. Those tests declare uniquely named queues, and several use the management API on `localhost:15672` to count connections and channels, which is how isolation is asserted rather than inferred from logs.

Test helpers exist for the timing windows. `withFastReconnect` shrinks the reconnect delays, and `withFastChannelRestart` shrinks the slot backoff, the stable window and the consumer wait. Both restore the previous values on cleanup. `waitUntil` polls a condition instead of sleeping, and `resetRabbitModuleForTest` returns the module singleton to a startable state.

## Known Limits

Restarts and escalations are log lines only. There are no per-slot metrics, so a channel that keeps dying is visible in the log and on the broker's channel count, not on a dashboard.

A handler that blocks occupies that queue's concurrency slots and its messages pile up. The channel stays healthy, so nothing restarts and no alert fires from this client.

A connection that answers heartbeats but hangs a channel operation is not detected. There is no per-channel operation timeout, so a hung RPC keeps the slot waiting rather than escalating.