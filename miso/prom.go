package miso

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	histoBuck = NewRWMap[prometheus.Histogram](func(name string) prometheus.Histogram {
		hist := prometheus.NewHistogram(prometheus.HistogramOpts{Name: name})
		e := prometheus.DefaultRegisterer.Register(hist)
		if e != nil {
			panic(fmt.Sprintf("failed to register histogram %v, %v", name, e))
		}
		return hist
	})

	counterBuck = NewRWMap[prometheus.Counter](func(name string) prometheus.Counter {
		counter := prometheus.NewCounter(prometheus.CounterOpts{Name: name})
		e := prometheus.DefaultRegisterer.Register(counter)
		if e != nil {
			panic(fmt.Sprintf("failed to register counter %v, %v", name, e))
		}
		return counter
	})
)

func init() {
	SetDefProp(PropMetricsEnabled, true)
	SetDefProp(PropPromRoute, "/metrics")

	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Prometheus",
		Bootstrap: PrometheusBootstrap,
		Condition: PrometheusBootstrapCondition,
	})
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
	return histoBuck.Get(name)
}

// Create new Counter.
//
// The Counter with this name is only created once and is automatically registered to the prometheus.DefaultRegisterer.
func NewPromCounter(name string) prometheus.Counter {
	return counterBuck.Get(name)
}

func PrometheusBootstrapCondition(rail Rail) (bool, error) {
	return GetPropBool(PropMetricsEnabled) && GetPropBool(PropServerEnabled), nil
}

func PrometheusBootstrap(rail Rail) error {
	handler := PrometheusHandler()
	RawGet(GetPropStr(PropPromRoute), func(c *gin.Context, rail Rail) {
		handler.ServeHTTP(c.Writer, c.Request)
	}).Build()
	return nil
}
