package rabbit

import "github.com/curtisnewbie/miso/miso"

// Configuration Properties for RabbitMQ
const (
	PropRabbitMqEnabled     = "rabbitmq.enabled"
	PropRabbitMqHost        = "rabbitmq.host"
	PropRabbitMqPort        = "rabbitmq.port"
	PropRabbitMqUsername    = "rabbitmq.username"
	PropRabbitMqPassword    = "rabbitmq.password"
	PropRabbitMqVhost       = "rabbitmq.vhost"
	PropRabbitMqConsumerQos = "rabbitmq.consumer.qos"
)

func init() {
	miso.SetDefProp(PropRabbitMqEnabled, false)
	miso.SetDefProp(PropRabbitMqHost, "localhost")
	miso.SetDefProp(PropRabbitMqPort, 5672)
	miso.SetDefProp(PropRabbitMqUsername, "guest")
	miso.SetDefProp(PropRabbitMqPassword, "guest")
	miso.SetDefProp(PropRabbitMqVhost, "")
	miso.SetDefProp(PropRabbitMqConsumerQos, DefaultQos)
}
