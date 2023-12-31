package miso

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	manualBootstrap = false
)

func init() {
	SetDefProp(PropMetricsEnabled, true)
	SetDefProp(PropMetricsRoute, "/metrics")

	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Prometheus",
		Bootstrap: PrometheusBootstrap,
		Condition: PrometheusBootstrapCondition,
		Order:     BootstrapOrderL2,
	})
}

// Default handler for prometheus metrics.
func PrometheusHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		if GetPropBool(PropMetricsAuthEnabled) {
			authorization := req.Header.Get("Authorization")
			secret, ok := ParseBearer(authorization)
			if !ok || secret != GetPropStr(PropMetricsAuthBearer) {
				Debug("metrics endpoint authorization failed")
				writer.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		promhttp.Handler().ServeHTTP(writer, req)
	}
}

// Timer based on prometheus.Histogram.
//
// Duration is measured in millisecond.
//
// Use NewHistTimer to create a new one, and each timer can only be used for once.
type HistTimer struct {
	hist  prometheus.Histogram
	begin time.Time
}

func (t *HistTimer) ObserveDuration() time.Duration {
	d := time.Since(t.begin)
	t.hist.Observe(float64(d.Milliseconds()))
	return d
}

// Create new timer that is backed by a prometheus.Histogram. Each timer can only be used for once.
func NewHistTimer(hist prometheus.Histogram) *HistTimer {
	if hist == nil {
		panic("prometheus.Histogram is nil")
	}
	return &HistTimer{
		hist:  hist,
		begin: time.Now(),
	}
}

// Create new Histogram.
//
// The created Histogram is automatically registered to the prometheus.DefaultRegisterer.
func NewPromHisto(name string) prometheus.Histogram {
	hist := prometheus.NewHistogram(prometheus.HistogramOpts{Name: name})
	if e := prometheus.DefaultRegisterer.Register(hist); e != nil {
		panic(fmt.Errorf("failed to register histogram %v, %w", name, e))
	}
	return hist
}

// Create new Counter.
//
// The Counter with this name is automatically registered to the prometheus.DefaultRegisterer.
func NewPromCounter(name string) prometheus.Counter {
	counter := prometheus.NewCounter(prometheus.CounterOpts{Name: name})
	if e := prometheus.DefaultRegisterer.Register(counter); e != nil {
		panic(fmt.Errorf("failed to register counter %v, %w", name, e))
	}
	return counter
}

func PrometheusBootstrapCondition(rail Rail) (bool, error) {
	return GetPropBool(PropMetricsEnabled) && GetPropBool(PropServerEnabled), nil
}

func PrometheusBootstrap(rail Rail) error {
	handler := PrometheusHandler()

	if GetPropBool(PropMetricsAuthEnabled) {
		if IsBlankStr(GetPropStr(PropMetricsAuthBearer)) {
			return fmt.Errorf("metrics authorization enabled, but secret is missing, please configure property '%v'",
				PropMetricsAuthBearer)
		}
		rail.Info("Enabled metrics authorization")
	}

	if !manualBootstrap {
		RawGet(GetPropStr(PropMetricsRoute), func(c *gin.Context, rail Rail) {
			handler.ServeHTTP(c.Writer, c.Request)
		}).Build()
	}
	return nil
}

// Caller wants to bootstrap prometheus manually.
//
// This is mainly used for gateway that implements handler for all endpoints.
func ManualBootstrapPrometheus() {
	manualBootstrap = true
}

// Timer based on prometheus.HistogramVec.
//
// Duration is measured in millisecond.
//
// Use NewVecTimer to create a new one, and each timer can only be used for once.
type VecTimer struct {
	histVec *prometheus.HistogramVec
	begin   time.Time
}

func (t *VecTimer) ObserveDuration(labels ...string) time.Duration {
	d := time.Since(t.begin)
	t.histVec.WithLabelValues(labels...).Observe(float64(d.Milliseconds()))
	return d
}

// Create new timer that is back by prometheus HistogramVec. Each timer can only be used for once.
func NewVecTimer(vec *prometheus.HistogramVec) *VecTimer {
	if vec == nil {
		panic("prometheus.HistogramVec is nil")
	}
	return &VecTimer{
		histVec: vec,
		begin:   time.Now(),
	}
}

// Create new HistogramVec.
//
// The HistogramVec is automatically registered to the prometheus.DefaultRegisterer.
func NewPromHistoVec(name string, labels []string) *prometheus.HistogramVec {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name}, labels)
	if e := prometheus.DefaultRegisterer.Register(vec); e != nil {
		panic(fmt.Errorf("failed to register HistogramVec %v, %v", name, e))
	}
	return vec
}
