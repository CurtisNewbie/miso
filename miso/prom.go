package miso

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	SetDefProp(PROP_METRICS_ENABLED, true)
	SetDefProp(PROP_PROM_ROUTE, "/metrics")
}

func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}
