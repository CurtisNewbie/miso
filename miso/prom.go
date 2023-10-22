package miso

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	histoBuck = &histogramBucket{buckets: make(map[string]prometheus.Histogram)}
)

type histogramBucket struct {
	sync.RWMutex
	buckets map[string]prometheus.Histogram
}

func init() {
	SetDefProp(PropMetricsEnabled, true)
	SetDefProp(PropPromRoute, "/metrics")
}

func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}

func NewPromTimer(name string) *prometheus.Timer {
	histoBuck.RLock()
	if v, ok := histoBuck.buckets[name]; ok {
		defer histoBuck.RUnlock()
		return prometheus.NewTimer(v)
	}
	histoBuck.RUnlock()

	histoBuck.Lock()
	defer histoBuck.Unlock()

	if v, ok := histoBuck.buckets[name]; ok {
		return prometheus.NewTimer(v)
	}

	hist := prometheus.NewHistogram(prometheus.HistogramOpts{Name: name})
	prometheus.DefaultRegisterer.MustRegister(hist)
	histoBuck.buckets[name] = hist
	return prometheus.NewTimer(hist)
}
