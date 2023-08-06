package metrics

import (
	"net/http"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	common.SetDefProp(common.PROP_METRICS_ENABLED, true)
	common.SetDefProp(common.PROP_PROM_ROUTE, "/metrics")
}

func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}
