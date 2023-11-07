package miso

import (
	"fmt"
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
//
// In Grafana, you can write the following query to measure the average ms each op takes.
//
//	rate(my_op_seconds_sum{job="my-job"}[$__rate_interval]) * 1000
func NewPromTimer(name string) *prometheus.Timer {
	if name == "" {
		panic("name is empty")
	}

	if !strings.HasSuffix(name, "_seconds") {
		name += "_seconds"
	}

	return prometheus.NewTimer(NewPromHistogram(name))
}

// Create new Histogram.
//
// The Histogram with this name is only created once and is automatically registered to the prometheus.DefaultRegisterer.
func NewPromHistogram(name string) prometheus.Histogram {
	histoBuck.RLock()
	if v, ok := histoBuck.buckets[name]; ok {
		defer histoBuck.RUnlock()
		return v
	}
	histoBuck.RUnlock()

	histoBuck.Lock()
	defer histoBuck.Unlock()

	if v, ok := histoBuck.buckets[name]; ok {
		return v
	}

	hist := prometheus.NewHistogram(prometheus.HistogramOpts{Name: name})
	e := prometheus.DefaultRegisterer.Register(hist)
	if e != nil {
		panic(fmt.Sprintf("failed to register histogram %v, %v", name, e))
	}
	histoBuck.buckets[name] = hist
	return hist
}
