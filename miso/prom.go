package miso

import (
	"fmt"
	"net/http"
	"time"

	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	prometheusBootstrapDisabled = false
)

func init() {
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Prometheus",
		Bootstrap: prometheusBootstrap,
		Condition: prometheusBootstrapCondition,
		Order:     BootstrapOrderL2,
	})
}

// Default handler for prometheus metrics.
func PrometheusHandler() http.Handler {
	if !GetPropBool(PropMetricsAuthEnabled) {
		return promhttp.Handler()
	}
	return BearerAuth(promhttp.Handler(), func() string { return GetPropStr(PropMetricsAuthBearer) })
}

// Timer based on prometheus.Histogram.
//
// Duration is measured in seconds.
//
// Use NewHistTimer to create a new one, and each timer can only be used for once.
type HistTimer struct {
	hist  prometheus.Histogram
	begin time.Time
}

func (t *HistTimer) Reset() {
	t.begin = time.Now()
}

func (t *HistTimer) ObserveDuration() time.Duration {
	d := time.Since(t.begin)
	t.hist.Observe(d.Seconds())
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

// Predefined bucket boundaries (upper bounds, in seconds) for HTTP request latency.
//
// 5ms to 30s, with a tail for slow endpoints.
func HttpRequestBuckets() []float64 {
	return []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 20, 30}
}

// Predefined bucket boundaries (upper bounds, in seconds) for DB query latency.
//
// Typical queries start around 1ms, slow ones can reach 10-30s (1ms - 30s).
func DBQueryBuckets() []float64 {
	return []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30}
}

// Predefined bucket boundaries (upper bounds, in seconds) for long-running task
// durations, e.g. cron jobs and batch processing (1s - 2h).
func LongTaskBuckets() []float64 {
	return []float64{1, 5, 15, 30, 60, 300, 600, 1800, 3600, 7200}
}

// Predefined bucket boundaries (upper bounds, in seconds) for LLM response
// latency, e.g. chat completions and other inference calls (5s - 5m).
func LLMResponseBuckets() []float64 {
	return []float64{5, 10, 15, 30, 45, 60, 80, 100, 120, 150, 300}
}

// Create new Histogram.
//
// The created Histogram is automatically registered to the prometheus.DefaultRegisterer.
//
// Custom bucket boundaries (upper bounds) can be provided as seconds, otherwise the
// default buckets (prometheus.DefBuckets + 20, 30, 45, 60) are used.
func NewPromHisto(name string, buckets ...float64) prometheus.Histogram {
	if len(buckets) == 0 {
		buckets = append(prometheus.DefBuckets, 20, 30, 45, 60)
	}
	hist := prometheus.NewHistogram(prometheus.HistogramOpts{Name: name, Buckets: buckets})
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

func prometheusBootstrapCondition(rail Rail) (bool, error) {
	return GetPropBool(PropMetricsEnabled) && GetPropBool(PropServerEnabled), nil
}

func prometheusBootstrap(rail Rail) error {

	if GetPropBool(PropMetricsAuthEnabled) {
		if strutil.IsBlankStr(GetPropStr(PropMetricsAuthBearer)) {
			return fmt.Errorf("metrics authorization enabled, but secret is missing, please configure property '%v'",
				PropMetricsAuthBearer)
		}
		rail.Info("Enabled metrics authorization")
	}

	if !prometheusBootstrapDisabled {
		handler := PrometheusHandler()
		metricsRoute := GetPropStr(PropMetricsRoute)
		HttpGet(metricsRoute,
			RawHandler(func(inb *Inbound) { handler.ServeHTTP(inb.Unwrap()) })).
			Desc("Collect prometheus metrics information").
			DocHeader("Authorization", "Basic authorization if enabled")
	}
	return nil
}

// Disable prometheus endpoint handler bootstrap.
func DisablePrometheusBootstrap() {
	prometheusBootstrapDisabled = true
}

// Timer based on prometheus.HistogramVec.
//
// Duration is measured in seconds.
//
// Use NewVecTimer to create a new one, and each timer can only be used for once.
type VecTimer struct {
	histVec *prometheus.HistogramVec
	begin   time.Time
}

func (t *VecTimer) Reset() {
	t.begin = time.Now()
}

func (t *VecTimer) ObserveDuration(labels ...string) time.Duration {
	d := time.Since(t.begin)
	t.histVec.WithLabelValues(labels...).Observe(d.Seconds())
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
//
// Custom bucket boundaries (upper bounds) can be provided as seconds, otherwise the
// default buckets (prometheus.DefBuckets + 20, 30, 45, 60) are used.
func NewPromHistoVec(name string, labels []string, buckets ...float64) *prometheus.HistogramVec {
	if len(buckets) == 0 {
		buckets = append(prometheus.DefBuckets, 20, 30, 45, 60)
	}
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name, Buckets: buckets}, labels)
	if e := prometheus.DefaultRegisterer.Register(vec); e != nil {
		panic(fmt.Errorf("failed to register HistogramVec %v, %v", name, e))
	}
	return vec
}
