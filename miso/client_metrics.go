package miso

import (
	"fmt"
	"io"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricClientRequests      = "miso_client_requests_total"
	metricClientRequestBytes  = "miso_client_request_bytes_total"
	metricClientResponseBytes = "miso_client_response_bytes_total"

	clientMetricsLabelName = "client_name"
	clientMetricsLabelHost = "host"
	clientMetricsLabelApp  = "app"
)

// The three outbound request counters are process wide: every client that enables metrics
// shares them. They are registered once at package initialization.
var (
	clientRequestsMetric  *prometheus.CounterVec
	clientReqBytesMetric  *prometheus.CounterVec
	clientRespBytesMetric *prometheus.CounterVec
)

func init() {
	labels := []string{clientMetricsLabelName, clientMetricsLabelHost, clientMetricsLabelApp}
	clientRequestsMetric = registerClientCounterVec(metricClientRequests, labels)
	clientReqBytesMetric = registerClientCounterVec(metricClientRequestBytes, labels)
	clientRespBytesMetric = registerClientCounterVec(metricClientResponseBytes, labels)
}

// registerClientCounterVec registers a counter vector, reusing an already registered
// collector of the same name. Registration happens at startup, so a conflicting collector
// is a programming error and panics there, never inside a request.
func registerClientCounterVec(name string, labels []string) *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{Name: name}, labels)
	err := prometheus.DefaultRegisterer.Register(vec)
	if err == nil {
		return vec
	}
	if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
		if existing, ok := are.ExistingCollector.(*prometheus.CounterVec); ok {
			return existing
		}
	}
	panic(fmt.Errorf("failed to register CounterVec %v, %w", name, err))
}

// countingReadCloser adds the bytes read from the wrapped reader to a counter.
type countingReadCloser struct {
	inner   io.ReadCloser
	counter prometheus.Counter
}

func (c *countingReadCloser) Read(p []byte) (int, error) {
	n, err := c.inner.Read(p)
	if n > 0 {
		c.counter.Add(float64(n))
	}
	return n, err
}

func (c *countingReadCloser) Close() error {
	return c.inner.Close()
}

// clientMetricsTransport records requests and payload bytes for a single client.
type clientMetricsTransport struct {
	inner  http.RoundTripper
	client *Client
	name   string
}

func (t *clientMetricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	labels := []string{t.name, t.targetHost(req), GetPropStr(PropAppName)}
	clientRequestsMetric.WithLabelValues(labels...).Inc()

	if req.Body != nil {
		req.Body = &countingReadCloser{inner: req.Body, counter: clientReqBytesMetric.WithLabelValues(labels...)}
	}

	resp, err := t.inner.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		return resp, err
	}

	resp.Body = &countingReadCloser{inner: resp.Body, counter: clientRespBytesMetric.WithLabelValues(labels...)}
	return resp, err
}

// targetHost is the service name when the request target is resolved through service
// discovery, and the URL host (including port) otherwise.
func (t *clientMetricsTransport) targetHost(req *http.Request) string {
	if t.client.discoverService && t.client.serviceName != "" {
		return t.client.serviceName
	}
	return req.URL.Host
}

// meteredClient returns a shallow copy of the client's http.Client whose transport
// records request metrics. The base client, its transport, and its connection pool are
// shared, never mutated.
func (t *Client) meteredClient() *http.Client {
	inner := t.client.Transport
	if inner == nil {
		inner = http.DefaultTransport
	}
	clone := *t.client
	clone.Transport = &clientMetricsTransport{inner: inner, client: t, name: t.metricsName}
	return &clone
}
