package miso

import (
	"net/http"
	"strings"
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

// Default handler for prometheus metrics.
func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// Create new Prometheus timer (in seconds).
//
// The timer is backed by a Histogram, and the histogram is named by
//
//	name + "_seconds"
//
// The Histogram with this name is only created once and is automatically registered to the prometheus.DefaultRegisterer.
func NewPromTimer(name string) *prometheus.Timer {
	if name == "" {
		panic("name is empty")
	}

	if !strings.HasSuffix(name, "_seconds") {
		name += "_seconds"
	}

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
