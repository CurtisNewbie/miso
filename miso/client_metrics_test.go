package miso

import (
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

const (
	clientMetricsTestApp = "client-metrics-test-app"

	testMetricClientRequests      = "miso_client_requests_total"
	testMetricClientRequestBytes  = "miso_client_request_bytes_total"
	testMetricClientResponseBytes = "miso_client_response_bytes_total"
)

// withProp sets a property for the test and restores the previous value when it ends.
func withProp(t *testing.T, key string, value any) {
	t.Helper()
	prev := GetPropStr(key)
	SetProp(key, value)
	t.Cleanup(func() {
		SetProp(key, prev)
	})
}

// withClientMetricsApp sets the app name the counters read.
func withClientMetricsApp(t *testing.T) {
	t.Helper()
	withProp(t, PropAppName, clientMetricsTestApp)
}

// forEachClientMetricSample walks every sample of a metric family.
func forEachClientMetricSample(t *testing.T, name string, fn func(m *dto.Metric)) {
	t.Helper()
	fams, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	for _, mf := range fams {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			fn(m)
		}
	}
}

// clientMetricSample returns the value of one label set, or 0 when it was not recorded.
func clientMetricSample(t *testing.T, name, clientName, host, app string) float64 {
	t.Helper()
	var value float64
	forEachClientMetricSample(t, name, func(m *dto.Metric) {
		if metricLabelValue(m, "client_name") != clientName ||
			metricLabelValue(m, "host") != host ||
			metricLabelValue(m, "app") != app {
			return
		}
		if c := m.GetCounter(); c != nil {
			value = c.GetValue()
		}
	})
	return value
}

// clientMetricByHost sums every sample of a metric whose host label matches, whatever the
// remaining labels are.
func clientMetricByHost(t *testing.T, name, host string) float64 {
	t.Helper()
	var sum float64
	forEachClientMetricSample(t, name, func(m *dto.Metric) {
		if metricLabelValue(m, "host") != host {
			return
		}
		if c := m.GetCounter(); c != nil {
			sum += c.GetValue()
		}
	})
	return sum
}

func metricLabelValue(m *dto.Metric, name string) string {
	for _, lp := range m.GetLabel() {
		if lp.GetName() == name {
			return lp.GetValue()
		}
	}
	return ""
}

func TestClientMetricsRecordsRequestAndBytes(t *testing.T) {
	withClientMetricsApp(t)

	const payload = "payload"
	const reply = "pong"
	srv := newClientMetricsServer(t, reply)
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	const clientName = "ping"

	requestsBefore := clientMetricSample(t, testMetricClientRequests, clientName, host, clientMetricsTestApp)
	reqBytesBefore := clientMetricSample(t, testMetricClientRequestBytes, clientName, host, clientMetricsTestApp)
	respBytesBefore := clientMetricSample(t, testMetricClientResponseBytes, clientName, host, clientMetricsTestApp)

	resp := NewClient(EmptyRail(), srv.URL).EnableMetrics(clientName).PostBytes([]byte(payload))
	if resp.Err != nil {
		t.Fatalf("unexpected error: %v", resp.Err)
	}
	body, err := resp.Str()
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if body != reply {
		t.Fatalf("response body = %q, want %q", body, reply)
	}

	if got := clientMetricSample(t, testMetricClientRequests, clientName, host, clientMetricsTestApp) - requestsBefore; got != 1 {
		t.Errorf("requests delta = %v, want 1", got)
	}
	if got := clientMetricSample(t, testMetricClientRequestBytes, clientName, host, clientMetricsTestApp) - reqBytesBefore; got != float64(len(payload)) {
		t.Errorf("request bytes delta = %v, want %v", got, len(payload))
	}
	if got := clientMetricSample(t, testMetricClientResponseBytes, clientName, host, clientMetricsTestApp) - respBytesBefore; got != float64(len(reply)) {
		t.Errorf("response bytes delta = %v, want %v", got, len(reply))
	}
}

func TestClientMetricsGetHasNoRequestBytes(t *testing.T) {
	withClientMetricsApp(t)

	const reply = "pong"
	srv := newClientMetricsServer(t, reply)
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	const clientName = "get-only"

	requestsBefore := clientMetricSample(t, testMetricClientRequests, clientName, host, clientMetricsTestApp)
	reqBytesBefore := clientMetricSample(t, testMetricClientRequestBytes, clientName, host, clientMetricsTestApp)
	respBytesBefore := clientMetricSample(t, testMetricClientResponseBytes, clientName, host, clientMetricsTestApp)

	resp := NewClient(EmptyRail(), srv.URL).EnableMetrics(clientName).Get()
	if resp.Err != nil {
		t.Fatalf("unexpected error: %v", resp.Err)
	}
	if _, err := resp.Str(); err != nil {
		t.Fatalf("read response: %v", err)
	}

	if got := clientMetricSample(t, testMetricClientRequests, clientName, host, clientMetricsTestApp) - requestsBefore; got != 1 {
		t.Errorf("requests delta = %v, want 1", got)
	}
	if got := clientMetricSample(t, testMetricClientRequestBytes, clientName, host, clientMetricsTestApp) - reqBytesBefore; got != 0 {
		t.Errorf("request bytes delta for a bodyless GET = %v, want 0", got)
	}
	if got := clientMetricSample(t, testMetricClientResponseBytes, clientName, host, clientMetricsTestApp) - respBytesBefore; got != float64(len(reply)) {
		t.Errorf("response bytes delta = %v, want %v", got, len(reply))
	}
}

func TestClientMetricsLabelsEmptyClientName(t *testing.T) {
	withClientMetricsApp(t)

	srv := newClientMetricsServer(t, "ok")
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	before := clientMetricSample(t, testMetricClientRequests, "", host, clientMetricsTestApp)

	resp := NewClient(EmptyRail(), srv.URL).EnableMetrics("").Get()
	if resp.Err != nil {
		t.Fatalf("unexpected error: %v", resp.Err)
	}
	if _, err := resp.Str(); err != nil {
		t.Fatalf("read response: %v", err)
	}

	if got := clientMetricSample(t, testMetricClientRequests, "", host, clientMetricsTestApp) - before; got != 1 {
		t.Errorf("requests delta with empty client name = %v, want 1", got)
	}
}

func TestClientMetricsUsesServiceNameForDiscoveredTarget(t *testing.T) {
	withClientMetricsApp(t)
	withProp(t, "client.addr.metrics-svc.host", "10.0.0.1")
	withProp(t, "client.addr.metrics-svc.port", 8080)

	const clientName = "discovered"
	before := clientMetricSample(t, testMetricClientRequests, clientName, "metrics-svc", clientMetricsTestApp)

	resp := NewClient(EmptyRail(), "/ping").
		EnableServiceDiscovery("metrics-svc").
		EnableMetrics(clientName).
		UseClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     http.Header{},
			}, nil
		})}).
		Get()
	if resp.Err != nil {
		t.Fatalf("unexpected error: %v", resp.Err)
	}
	if _, err := resp.Str(); err != nil {
		t.Fatalf("read response: %v", err)
	}

	if got := clientMetricSample(t, testMetricClientRequests, clientName, "metrics-svc", clientMetricsTestApp) - before; got != 1 {
		t.Errorf("requests delta for discovered target = %v, want 1", got)
	}
	if got := clientMetricSample(t, testMetricClientRequests, clientName, "10.0.0.1:8080", clientMetricsTestApp); got != 0 {
		t.Errorf("resolved address recorded as host unexpectedly, sample = %v", got)
	}
}

func TestClientMetricsOffByDefault(t *testing.T) {
	withClientMetricsApp(t)

	srv := newClientMetricsServer(t, "ok")
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")

	resp := NewClient(EmptyRail(), srv.URL).Get()
	if resp.Err != nil {
		t.Fatalf("unexpected error: %v", resp.Err)
	}
	if _, err := resp.Str(); err != nil {
		t.Fatalf("read response: %v", err)
	}

	if got := clientMetricByHost(t, testMetricClientRequests, host); got != 0 {
		t.Errorf("requests recorded for a client without EnableMetrics, sample = %v", got)
	}
}

func TestClientMetricsCountsFailedAttempt(t *testing.T) {
	withClientMetricsApp(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	host := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	const clientName = "unreachable"
	before := clientMetricSample(t, testMetricClientRequests, clientName, host, clientMetricsTestApp)

	resp := NewClient(EmptyRail(), "http://"+host).EnableMetrics(clientName).Get()
	if resp.Err == nil {
		t.Fatal("expected a connection error")
	}

	if got := clientMetricSample(t, testMetricClientRequests, clientName, host, clientMetricsTestApp) - before; got != 1 {
		t.Errorf("requests delta for failed attempt = %v, want 1", got)
	}
}

func TestClientMetricsSharedByMultipleClients(t *testing.T) {
	withClientMetricsApp(t)

	srv := newClientMetricsServer(t, "ok")
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	const firstName = "first"
	const secondName = "second"
	firstBefore := clientMetricSample(t, testMetricClientRequests, firstName, host, clientMetricsTestApp)
	secondBefore := clientMetricSample(t, testMetricClientRequests, secondName, host, clientMetricsTestApp)

	for _, name := range []string{firstName, secondName} {
		resp := NewClient(EmptyRail(), srv.URL).EnableMetrics(name).Get()
		if resp.Err != nil {
			t.Fatalf("unexpected error for client %q: %v", name, resp.Err)
		}
		if _, err := resp.Str(); err != nil {
			t.Fatalf("read response for client %q: %v", name, err)
		}
	}

	if got := clientMetricSample(t, testMetricClientRequests, firstName, host, clientMetricsTestApp) - firstBefore; got != 1 {
		t.Errorf("first client requests delta = %v, want 1", got)
	}
	if got := clientMetricSample(t, testMetricClientRequests, secondName, host, clientMetricsTestApp) - secondBefore; got != 1 {
		t.Errorf("second client requests delta = %v, want 1", got)
	}
}

func TestClientMetricsLeavesDefaultClientUntouched(t *testing.T) {
	withClientMetricsApp(t)

	srv := newClientMetricsServer(t, "ok")
	defer srv.Close()

	transportBefore := MisoDefaultClient.Transport
	if transportBefore == nil {
		transportBefore = http.DefaultTransport
	}

	resp := NewClient(EmptyRail(), srv.URL).EnableMetrics("shared-client").Get()
	if resp.Err != nil {
		t.Fatalf("unexpected error: %v", resp.Err)
	}
	if _, err := resp.Str(); err != nil {
		t.Fatalf("read response: %v", err)
	}

	transportAfter := MisoDefaultClient.Transport
	if transportAfter == nil {
		transportAfter = http.DefaultTransport
	}
	if transportAfter != transportBefore {
		t.Fatal("default client transport was replaced by a metered request")
	}
}

func TestClientMetricsCountsEachRedirectHop(t *testing.T) {
	withClientMetricsApp(t)

	const redirectBody = "moved"
	const finalBody = "final"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			w.Header().Set("Location", "/final")
			w.WriteHeader(http.StatusFound)
			if _, err := w.Write([]byte(redirectBody)); err != nil {
				t.Errorf("write redirect body: %v", err)
			}
			return
		}
		if _, err := w.Write([]byte(finalBody)); err != nil {
			t.Errorf("write final body: %v", err)
		}
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	const clientName = "redirect"

	requestsBefore := clientMetricSample(t, testMetricClientRequests, clientName, host, clientMetricsTestApp)
	respBytesBefore := clientMetricSample(t, testMetricClientResponseBytes, clientName, host, clientMetricsTestApp)

	resp := NewClient(EmptyRail(), srv.URL+"/start").EnableMetrics(clientName).Get()
	if resp.Err != nil {
		t.Fatalf("unexpected error: %v", resp.Err)
	}
	body, err := resp.Str()
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if body != finalBody {
		t.Fatalf("response body = %q, want %q", body, finalBody)
	}

	if got := clientMetricSample(t, testMetricClientRequests, clientName, host, clientMetricsTestApp) - requestsBefore; got != 2 {
		t.Errorf("requests delta for one redirect = %v, want 2", got)
	}
	wantBytes := float64(len(redirectBody) + len(finalBody))
	if got := clientMetricSample(t, testMetricClientResponseBytes, clientName, host, clientMetricsTestApp) - respBytesBefore; got != wantBytes {
		t.Errorf("response bytes delta = %v, want %v", got, wantBytes)
	}
}

func TestClientMetricsCountsDecompressedResponseBytes(t *testing.T) {
	withClientMetricsApp(t)

	const reply = "hello"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		if _, err := gz.Write([]byte(reply)); err != nil {
			t.Errorf("write gzip body: %v", err)
		}
		if err := gz.Close(); err != nil {
			t.Errorf("close gzip writer: %v", err)
		}
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	const clientName = "gzip"

	respBytesBefore := clientMetricSample(t, testMetricClientResponseBytes, clientName, host, clientMetricsTestApp)

	resp := NewClient(EmptyRail(), srv.URL).EnableMetrics(clientName).Get()
	if resp.Err != nil {
		t.Fatalf("unexpected error: %v", resp.Err)
	}
	body, err := resp.Str()
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if body != reply {
		t.Fatalf("response body = %q, want %q", body, reply)
	}

	if got := clientMetricSample(t, testMetricClientResponseBytes, clientName, host, clientMetricsTestApp) - respBytesBefore; got != float64(len(reply)) {
		t.Errorf("response bytes delta for a gzip response = %v, want %v", got, len(reply))
	}
}

func newClientMetricsServer(t *testing.T, reply string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Errorf("read request body: %v", err)
		}
		if _, err := w.Write([]byte(reply)); err != nil {
			t.Errorf("write response body: %v", err)
		}
	}))
}
