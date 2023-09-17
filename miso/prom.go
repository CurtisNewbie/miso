package miso

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	SetDefProp(PropMetricsEnabled, true)
	SetDefProp(PropPromRoute, "/metrics")
}

func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}
