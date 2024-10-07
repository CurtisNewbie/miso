package miso

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	ServiceStatusUp   = "UP"
	ServiceStatusDown = "DOWN"
)

// Indicator of health status
type HealthIndicator struct {
	Name        string               // name of the indicator
	CheckHealth func(rail Rail) bool // Check health
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
		start := time.Now()
		hs = append(hs, HealthStatus{
			Healthy: indi.CheckHealth(rail),
			Name:    indi.Name,
		})
		rail.Debugf("HealthIndicator %v took %v", indi.Name, time.Since(start))
	}
	return hs
}

// Create a default health check endpoint that simply doesn't nothing except returing 200
func DefaultHealthCheck(ctx *gin.Context) {
	rail := EmptyRail()
	hs := CheckHealth(rail)
	for i := range hs {
		s := hs[i]
		if !s.Healthy {
			rail.Warnf("Component %s is down, healthcheck failed", s.Name)
			ctx.String(http.StatusServiceUnavailable, ServiceStatusDown)
			return
		}
	}
	rail.Debugf("Service healthcheck pass")
	ctx.String(http.StatusOK, ServiceStatusUp)
}

// Check health status, return true if all health check pass
func IsHealthcheckPass(rail Rail) bool {
	hs := CheckHealth(rail)
	for i := range hs {
		s := hs[i]
		if !s.Healthy {
			rail.Warnf("Component %s is down, healthcheck failed", s.Name)
			return false
		}
	}
	rail.Debugf("Service healthcheck pass")
	return true
}
