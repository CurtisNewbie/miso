# Prometheus Metrics

Miso auto-bootstraps a `/metrics` endpoint (prometheus scrape target) when `metrics.enabled=true` and `server.enabled=true` (both default true). All metric types are auto-registered to `prometheus.DefaultRegisterer`.

## Metric Constructors

```go
// Histogram — durations, distributions
hist := miso.NewPromHisto("my_op_seconds")

// Histogram with labels
vec := miso.NewPromHistoVec("my_op_seconds", []string{"status", "method"})

// Counter — monotonically increasing totals
counter := miso.NewPromCounter("my_events_total")

// Counter with labels
vec := miso.NewPromCounterVec("my_events_total", []string{"status", "method"})

// Histogram with custom bucket boundaries (upper bounds, in seconds).
// Defaults to prometheus.DefBuckets + 20, 30, 45, 60 when omitted.
hist := miso.NewPromHisto("db_query_seconds", .005, .01, .05, .1, .5, 1, 2.5, 5, 10, 30)

// Or use a predefined bucket set for a common use case:
//   miso.HttpRequestBuckets() — HTTP request latency, 5ms-30s
//   miso.DBQueryBuckets()     — DB query latency, 1ms-30s
//   miso.LongTaskBuckets()    — long-running tasks, 1s-2h
//   miso.LLMResponseBuckets() — LLM response latency, 5s-5m
hist := miso.NewPromHisto("http_request_seconds", miso.HttpRequestBuckets()...)
```

Call these at package init or once at startup — panics on duplicate registration.

## Timers

Timers measure duration in **seconds** (`ObserveDuration` records `d.Seconds()`) and observe into the underlying histogram.

```go
// Single histogram timer
hist := miso.NewPromHisto("db_query_seconds")

func DoQuery(rail miso.Rail) error {
    t := miso.NewHistTimer(hist)
    defer t.ObserveDuration()
    // ... query ...
}

// HistogramVec timer (with labels)
vec := miso.NewPromHistoVec("http_request_seconds", []string{"route", "status"})

func HandleRequest(rail miso.Rail, route string) {
    t := miso.NewVecTimer(vec)
    defer t.ObserveDuration(route, "200")
}
```

Timer methods:
- `ObserveDuration() time.Duration` — records elapsed seconds, returns duration
- `ObserveDuration(labels ...string) time.Duration` — labeled variant for `VecTimer`
- `Reset()` — restarts the timer without creating a new one

## Naming Conventions

Unit suffix is **optional** (Prometheus SHOULD-level best practice, not enforced at ingestion). If used, it must be a **base unit** (`_seconds`, `_bytes`) — never `_ms` or `_milliseconds`, both fail `promtool check metrics` (promlint: "abbreviated units" / "use base unit seconds"). Counters should end with `_total` (`my_events_total`).

## Client Metrics

miso clients can record their outbound traffic. Enabling is per client and off by default, so a client that never calls `EnableMetrics` records nothing.

```go
client := miso.NewClient(rail, "/api/users").
    EnableServiceDiscovery("user-service").
    EnableMetrics("user-api")
```

The client name is whatever identifies that client in your service. An empty name is allowed and recorded as an empty label value. `EnableMetrics` may be chained in any order with the other client options.

Three counters share the same label set.

| Metric | Counts |
|---|---|
| `miso_client_requests_total` | attempts, including failed connections, non-2xx responses, and each redirect hop |
| `miso_client_request_bytes_total` | request body bytes read by the HTTP transport |
| `miso_client_response_bytes_total` | response body bytes read by the caller |

| Label | Value |
|---|---|
| `client_name` | the name passed to `EnableMetrics` |
| `host` | the resolved service name when the client uses service discovery, otherwise the URL host and port |
| `app` | the `app.name` property of the emitting process |

QPS is derived from the attempt counter rather than recorded:

```promql
rate(miso_client_requests_total{client_name="user-api"}[1m])
```

Byte counts come from the body streams, so they are payload bytes after transport decompression, not bytes on the wire. A response body that is never read is not counted; `Str`, `Bytes`, `WriteTo`, and `WriteToFile` read it in full, and `Require2xx` reads it when it builds an error.

Keep `client_name` to a small, stable set chosen in code, one name per logical client. The number of time series grows with the number of clients and targets, so a name built per request would multiply them.

## Configuration

| Property | Default | Description |
|---|---|---|
| `metrics.enabled` | `true` | Enable prometheus support |
| `metrics.route` | `/metrics` | Scrape endpoint path |
| `metrics.auth.enabled` | `false` | Require Bearer auth on `/metrics` |
| `metrics.auth.bearer` | — | Bearer token secret (required when auth enabled) |

## Push Gateway

Push metrics to a Prometheus Pushgateway instead of (or in addition to) scraping:

| Property | Default | Description |
|---|---|---|
| `metrics.push-gateway.enabled` | `false` | Enable push gateway integration |
| `metrics.push-gateway.url` | — | Pushgateway URL (required) |
| `metrics.push-gateway.job` | `${app.name}` | Job label |
| `metrics.push-gateway.push-interval-sec` | `30` | Push interval in seconds |
| `metrics.push-gateway.auth.enabled` | `false` | Enable Basic auth to pushgateway |
| `metrics.push-gateway.auth.username` | — | Basic auth username |
| `metrics.push-gateway.auth.password` | — | Basic auth password |

Grouping label `instance` is auto-set to `host:port`.

## Advanced

```go
// Custom handler (e.g. mount on a different path manually)
handler := miso.PrometheusHandler() // respects auth config
http.Handle("/custom-metrics", handler)

// Disable auto-mounted /metrics route (before bootstrap)
miso.DisablePrometheusBootstrap()
```

## Memory Stats Logging

| Property | Default | Description |
|---|---|---|
| `metrics.memstat.log.job.enabled` | `false` | Periodically log Go runtime memory stats |
| `metrics.memstat.log.job.cron` | `0/30 * * * * *` | Cron expression for mem stats logging |
