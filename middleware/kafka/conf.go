package kafka

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: Kafka Configuration
const (
	// misoconfig-prop: Enable kafka client | false
	PropKafkaEnabled = "kafka.enabled"

	// misoconfig-prop: list of kafka server addresses | localhost:9092
	PropKafkaServerAddr = "kafka.server.addr"

	// misoconfig-prop: start offset for new consumer groups, first or last | last
	PropKafkaStartOffset = "kafka.start.offset"

	// misoconfig-prop: timeout for a single kafka write | 5s
	PropKafkaWriteTimeout = "kafka.write.timeout"
)

// misoconfig-default-start
func init() {
	miso.SetDefProp(PropKafkaEnabled, false)
	miso.SetDefProp(PropKafkaServerAddr, "localhost:9092")
	miso.SetDefProp(PropKafkaStartOffset, "last")
	miso.SetDefProp(PropKafkaWriteTimeout, "5s")
}

// misoconfig-default-end
