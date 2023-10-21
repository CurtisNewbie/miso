package miso

import "sync"

// Indicator of health status
type HealthIndicator interface {
	Name() string               // name of the indicator
	CheckHealth(rail Rail) bool // Check health
}

type HealthStatus struct {
	Name    string
	Healthy bool
}

type aggregatedHealthIndicator struct {
	sync.RWMutex
	indicators []HealthIndicator
}

var (
	aggIndi = aggregatedHealthIndicator{indicators: make([]HealthIndicator, 0, 10)}
)

// Add health indicator.
func AddHealthIndicator(hi HealthIndicator) {
	aggIndi.Lock()
	defer aggIndi.Unlock()
	aggIndi.indicators = append(aggIndi.indicators, hi)
}

// Check health status.
func CheckHealth(rail Rail) []HealthStatus {
	aggIndi.RLock()
	defer aggIndi.RUnlock()
	hs := make([]HealthStatus, 0, len(aggIndi.indicators))
	for i := range aggIndi.indicators {
		indi := aggIndi.indicators[i]
		hs = append(hs, HealthStatus{
			Healthy: indi.CheckHealth(rail),
			Name:    indi.Name(),
		})
	}
	return hs
}
