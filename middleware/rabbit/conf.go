package rabbit

import "github.com/curtisnewbie/miso/miso"

// misoapi-config-section: RabbitMQ Configuration
const (
	// misoapi-config: enable RabbitMQ client | false
	PropRabbitMqEnabled = "rabbitmq.enabled"

	// misoapi-config: RabbitMQ server host | `localhost`
	PropRabbitMqHost = "rabbitmq.host"

	// misoapi-config: RabbitMQ server port | 5672
	PropRabbitMqPort = "rabbitmq.port"

	// misoapi-config: username used to connect to server
	PropRabbitMqUsername = "rabbitmq.username"

	// misoapi-config: password used to connect to server
	PropRabbitMqPassword = "rabbitmq.password"

	// misoapi-config: virtual host
	PropRabbitMqVhost = "rabbitmq.vhost"

	// misoapi-config: consumer QOS | 68
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
