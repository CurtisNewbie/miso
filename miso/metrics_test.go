package miso

import (
	"testing"
)

func TestMetricsCollector(t *testing.T) {
	var collector MetricsCollector = NewMetricsCollector(DefaultMetricDesc(nil))
	collector.Read()
	Info("\n\n" + SprintMemStats(collector.MemStats()))
}
