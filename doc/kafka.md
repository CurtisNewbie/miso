# Kafka

Configure Kafka connection in your `conf.yml` file. Internally, miso will maintain same pool of Kafka connection shared by all kafka Writer and kafka Reader. When server exists, miso close these managed Readers and Writer automatically.

## Send Kafka Byte Message

```go
func sendByteMessage(rail miso.Rail, key string, topic string, data []byte) error {
	err := kafka.WriteMessage(rail, topic, key, data)
	if err != nil {
		return err
	}
	return nil
}
```

## Send Kafka Json Message

```go
type MyMessage struct {
	SomeId   string
	SomeText string
}

func sendJsonMessage(rail miso.Rail, topic string, key string, myMessage MyMessage) error {
	err := kafka.WriteMessageJson(rail, topic, key, myMessage)
	if err != nil {
		return err
	}
	return nil
}
```

## Register Kafka Listener

Register Kafka listener before server bootstrap:

```go
miso.PreServerBootstrap(func(rail miso.Rail) error {
    kafka.AddKafkaListener(kafka.KafkaReaderConfig{
        Topic:       "my-topic",
        GroupId:     "my-group",
        Concurrency: 3,
        Listen: func(rail miso.Rail, m kafka.Message) error {
            rail.Infof("message headers: %#v, key: %s, body: %s", m.Headers, m.Key, m.Value)
            return nil
        },
    })
    return nil
})
```

## Obtain Underlying Managed Kafka Writer

If you want to get the underlying kafka Writer, you can:

```go
writer := kafka.GetWriter()

// do something with the writer
// ...
```

## Create New, Unmanaged Kafka Writer / Reader

You can create Writer that is not managed by miso:

```go
writer, err := kafka.NewWriter([]string{"localhost:9092"})
```

As well as Reader:

```go
reader, err := kafka.NewReader([]string{"localhost:9092"}, "my-group", "my-topic")
```
