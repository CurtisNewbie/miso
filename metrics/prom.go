package metrics

import (
	"net/http"

	"github.com/curtisnewbie/miso/core"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	core.SetDefProp(core.PROP_METRICS_ENABLED, true)
	core.SetDefProp(core.PROP_PROM_ROUTE, "/metrics")
}

func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}
