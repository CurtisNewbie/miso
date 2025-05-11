package kafka

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: Kafka Configuration
const (
	// misoconfig-prop: Enable kafka client | false
	PropKafkaEnabled = "kafka.enabled"

	// misoconfig-prop: list of kafka server addresses | localhost:9092
	PropKafkaServerAddr = "kafka.server.addr"
)

// misoconfig-default-start
func init() {
	miso.SetDefProp(PropKafkaEnabled, false)
	miso.SetDefProp(PropKafkaServerAddr, "localhost:9092")
}

// misoconfig-default-end
