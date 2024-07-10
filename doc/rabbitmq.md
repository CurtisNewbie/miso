# RabbitMQ

## RabbitMQ Integration

Miso internally relies on `github.com/rabbitmq/amqp091-go` for RabbitMQ integration.

With RabbitMQ, there are three components that we need to create before server startup. These are:
- exchange
- queue
- binding
- and of course, listener to a queue (not really a RabbitMQ concept tho).

To simplify the whole RabbitMQ AMQP protocol concept, miso introduces the **event bus**. With which you don't need to declare these three types of components individually.

Before server bootstrap, you will need to declare the event bus as follows using `miso.NewEventBus`:

```go
miso.PreServerBootstrap(func(rail miso.Rail) error {
    rabbit.NewEventBus("vfm.image.compression")
    return nil
})
```

If you are creating listener for the event bus (i.e., you are the consumer), then you can simply use `miso.SubEventBus`:

```go
type CompressionEvent struct {
    // ...
}

miso.PreServerBootstrap(func(rail miso.Rail) error {
    rabbit.SubEventBus(
        /* event bus name */ "vfm.image.compression" ,
        /* concurrency */ 2,
        func(rail miso.Rail, evt CompressionEvent) error {
            return nil
        })
    return nil
})
```

Once the server is fully up and running, to dispatch a new message to the event bus, you can use `miso.PubEventBus`

```go
if err := rabbit.PubEventBus(rail, CompressionEvent{}, "vfm.image.compression"); err != nil {
    return fmt.Errorf("failed to send event, %w", err)
}
```

Basically, event bus declare exchange, queue and binding for you. A direct exchange is created for each event bus. Both the queue and exchange use exactly the same name as the name of the event bus, and they are bound together using routing key `'#'`. This should satisfy most of the usage of message queue.

Of course, you may declare the queue, exchange and binding yourself using following API:

- `rabbit.RegisterRabbitQueue`
- `rabbit.RegisterRabbitBinding`
- `rabbit.RegisterRabbitExchange`

```go
miso.PreServerBootstrap(func(rail miso.Rail) error {
    name := "vfm.image.compression"
    rabbit.RegisterRabbitQueue(rabbit.QueueRegistration{Name: name, Durable: true})
    rabbit.RegisterRabbitBinding(rabbit.BindingRegistration{Queue: name, RoutingKey: "#", Exchange: name})
    rabbit.RegisterRabbitExchange(rabbit.ExchangeRegistration{Name: name, Durable: true, Kind: "direct"})
    return nil
})
```

To declare a listener, you will need to use `miso.AddRabbitListener`:

```go
miso.PreServerBootstrap(func(rail miso.Rail) error {
    name := "vfm.image.compression"
    rabbit.AddRabbitListener(rabbit.JsonMsgListener[CompressionEvent]{
        QueueName:     name,
        NumOfRoutines: 2,
        Handler: func(rail miso.Rail, payload CompressionEvent) error {
            return nil
        }})
    return nil
})
```

To send a RabbitMQ message, you will need to use `miso.PublishJson`:

```go
err := rabbit.PublishJson(rail, CompressionEvent{}, "vfm.image.compression.exchange" /* exchange */, "#" /* routingKey */)
if err != nil {
    return fmt.Errorf("failed to send event, %w", err)
}
```

It's pretty much the same as the event bus.

## Message Redelivery Mechanism

What is special about miso is that miso provides a message redelivery mechanism with 5 second delay. It's inspired by [https://ivanyu.me/blog/2015/02/16/delayed-message-delivery-in-rabbitmq/](https://ivanyu.me/blog/2015/02/16/delayed-message-delivery-in-rabbitmq/).

The redelivery mechanism takes effect on the consumer side. Everytime a queue and exchange are declared, miso also declares a redelivery queue specifically for the original exchange. The redelivery queue specifies a ttl for the message (5 seconds), it uses the dead letter mechanism to bind to the original exchange, so that whenever the message expired in the redelivery queue, the messages are dispatched to the original exchange and retried by the consumer.

With miso, the wrapping message listener understands this behaviour, and will try to dispatch the messages back to the redelivery queue if the messsages cannot be consumed by the application code without error.

## Tracing

In miso, both the message dispatcher and message consumer support tracing by using `miso.Rail`. Tracing information are propagated by passing the `miso.Rail` around (essentially `context.Context`) and are stored in message headers. The wrapping message listener provided by miso will extract the tracing information from the message headers automatically, and propagate the information back to application code using `miso.Rail`.