package rabbit

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: RabbitMQ Configuration
const (
	// misoconfig-prop: enable RabbitMQ client | false
	PropRabbitMqEnabled = "rabbitmq.enabled"

	// misoconfig-prop: RabbitMQ server host | localhost
	PropRabbitMqHost = "rabbitmq.host"

	// misoconfig-prop: RabbitMQ server port | 5672
	PropRabbitMqPort = "rabbitmq.port"

	// misoconfig-prop: username used to connect to server | guest
	PropRabbitMqUsername = "rabbitmq.username"

	// misoconfig-prop: password used to connect to server | guest
	PropRabbitMqPassword = "rabbitmq.password"

	// misoconfig-prop: virtual host
	PropRabbitMqVhost = "rabbitmq.vhost"

	// misoconfig-prop: consumer QOS | 68
	PropRabbitMqConsumerQos = "rabbitmq.consumer.qos"

	// misoconfig-prop: publisher channel pool size | 20
	PropRabbitMqPublisherChanPoolSize = "rabbitmq.publisher.channel-pool-size"
)

// misoconfig-default-start
func init() {
	miso.SetDefProp(PropRabbitMqEnabled, false)
	miso.SetDefProp(PropRabbitMqHost, "localhost")
	miso.SetDefProp(PropRabbitMqPort, 5672)
	miso.SetDefProp(PropRabbitMqUsername, "guest")
	miso.SetDefProp(PropRabbitMqPassword, "guest")
	miso.SetDefProp(PropRabbitMqConsumerQos, 68)
	miso.SetDefProp(PropRabbitMqPublisherChanPoolSize, 20)
}

// misoconfig-default-end
