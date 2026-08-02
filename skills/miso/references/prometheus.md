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
