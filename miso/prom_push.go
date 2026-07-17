package miso

import (
	"fmt"
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

func init() {
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Prometheus Push Gateway",
		Bootstrap: prometheusPushGatewayBootstrap,
		Condition: prometheusPushGatewayBootstrapCondition,
		Order:     BootstrapOrderL2,
	})
}

func prometheusPushGatewayBootstrapCondition(rail Rail) (bool, error) {
	return GetPropBool(PropMetricsEnabled) && GetPropBool(PropMetricsPushGatewayEnabled), nil
}

func prometheusPushGatewayBootstrap(rail Rail) error {
	url := GetPropStr(PropMetricsPushGatewayUrl)
	if strutil.IsBlankStr(url) {
		return errs.NewErrf("prometheus pushgateway enabled, but url is missing, please configure property '%v'",
			PropMetricsPushGatewayUrl)
	}

	job := GetPropStr(PropMetricsPushGatewayJob)
	instance := fmt.Sprintf("%s:%s", ResolveServerHost(GetPropStr(PropServerHost)), GetPropStr(PropServerPort))
	intervalSec := GetPropInt(PropMetricsPushGatewayIntervalSec)

	pusher := push.New(url, job).Gatherer(prometheus.DefaultGatherer).Grouping("instance", instance)

	if GetPropBool(PropMetricsPushGatewayAuthEnabled) {
		username := GetPropStr(PropMetricsPushGatewayAuthUsername)
		password := GetPropStr(PropMetricsPushGatewayAuthPassword)
		if strutil.IsBlankStr(username) || strutil.IsBlankStr(password) {
			return errs.NewErrf("prometheus pushgateway basic auth enabled, but username or password is missing, please configure properties '%v' and '%v'",
				PropMetricsPushGatewayAuthUsername, PropMetricsPushGatewayAuthPassword)
		}
		pusher = pusher.BasicAuth(username, password)
		rail.Info("Enabled pushgateway basic auth")
	}

	rail.Infof("Pushing metrics to Pushgateway %v, job=%v, instance=%v, interval=%vs", url, job, instance, intervalSec)

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := pusher.Push(); err != nil {
					rail.Errorf("failed to push metrics to pushgateway, %v", err)
				}
			case <-done:
				return
			}
		}
	}()

	AddShutdownHook(func() {
		close(done)
		ticker.Stop()
		if err := pusher.Push(); err != nil {
			rail.Errorf("failed to push metrics to pushgateway, %v", err)
		}
	})

	return nil
}
