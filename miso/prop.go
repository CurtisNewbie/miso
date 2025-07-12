package miso

// misoconfig-section: Common Configuration
const (

	// misoconfig-prop: name of the application
	PropAppName = "app.name"

	// misoconfig-prop: warning threshold for slow ComponentBootstrap | 5s
	PropAppSlowBoostrapThresohold = "app.slow-bootstrap-threshold"

	// misoconfig-prop: whether production mode is turned on | true
	PropProdMode = "mode.production"

	// misoconfig-prop: extra config files that should be loaded
	PropConfigExtraFiles = "config.extra.files"

	PropAppTestEnv = "app.test-env"
)

// misoconfig-section: Web Server Configuration
const (

	// misoconfig-prop: enable http server | true
	PropServerEnabled = "server.enabled"

	// misoconfig-prop: http server host | 127.0.0.1
	PropServerHost = "server.host"

	// misoconfig-prop: http server port | 8080
	PropServerPort = "server.port"

	// misoconfig-prop: health check url | /health
	// misoconfig-alias: consul.healthCheckUrl | v0.2.0
	PropHealthCheckUrl = "server.health-check-url"

	// misoconfig-prop: health check interval, it's only used for service discovery, e.g., Consul | 5s
	// misoconfig-alias: consul.healthCheckInterval | v0.2.0
	PropHealthCheckInterval = "server.health-check-interval"

	// misoconfig-prop: health check timeout, it's only used for service discovery, e.g., Consul | 3s
	// misoconfig-alias: consul.healthCheckTimeout | v0.2.0
	PropHealthcheckTimeout = "server.health-check-timeout"

	// misoconfig-prop: log all http server routes in INFO level | false
	PropServerLogRoutes = "server.log-routes"

	// misoconfig-prop: http server bearer authorization token for all endpoints |
	PropServerAuthBearer = "server.auth.bearer"

	// misoconfig-prop: time wait (in second) before whole app server shutdown (previously, before `v0.1.12`, it only applies to the http server) | 30
	// misoconfig-alias: server.gracefulShutdownTimeSec | v0.2.0
	PropServerGracefulShutdownTimeSec = "server.graceful-shutdown-time-sec"

	// misoconfig-prop: logs time duration for each inbound http request | false
	PropServerPerfEnabled = "server.perf.enabled"

	// misoconfig-prop: propagate trace info from inbound requests | true
	PropServerPropagateInboundTrace = "server.trace.inbound.propagate"

	// misoconfig-prop: enable inbound request parameter validation | true
	PropServerRequestValidateEnabled = "server.validate.request.enabled"

	// misoconfig-prop: enable server request log | false
	PropServerRequestLogEnabled = "server.request-log.enabled"

	// misoconfig-prop: enable pprof (exposed using endpoint '/debug/pprof'); in non-prod mode, it's always enabled | false
	PropServerPprofEnabled = "server.pprof.enabled"

	// misoconfig-prop: bearer token for pprof endpoints' authentication. however, if `server.auth.bearer` is set, this prop is ignored.
	PropServerPprofAuthBearer = "server.pprof.auth.bearer"

	// misoconfig-prop: generate api doc | true
	// misoconfig-alias: server.generate-endpoint-doc.enabled | v0.2.0
	PropServerGenerateEndpointDocEnabled = "server.api-doc.enabled"

	// misoconfig-prop: build webpage for the generated api doc | true
	// misoconfig-alias: server.generate-endpoint-doc.web.enabled | v0.2.0
	PropServerGenerateEndpointDocApiEnabled = "server.api-doc.web.enabled"

	// misoconfig-prop: generate markdown api doc to the specified file
	// misoconfig-alias: server.generate-endpoint-doc.file | v0.2.0
	PropServerGenerateEndpointDocFile = "server.api-doc.file"

	// misoconfig-prop: the generated markdown api doc should exclude miso.TClient demo | false
	// misoconfig-alias: server.generate-endpoint-doc.file-excl-tclient-demo | v0.2.0
	PropServerGenerateEndpointDocFileExclTClientDemo = "server.api-doc.file-excl-tclient-demo"

	// misoconfig-prop: the generated markdown api doc should exclude Angular HttpClient demo | false
	// misoconfig-alias: server.generate-endpoint-doc.file-excl-ng-client-demo | v0.2.0
	PropServerGenerateEndpointDocFileExclNgClientDemo = "server.api-doc.file-excl-ngclient-demo"

	// misoconfig-prop: the generated markdown api doc should exclude openapi json for each endpoint | true
	// misoconfig-alias: server.generate-endpoint-doc.file-excl-openapi-spec | v0.2.0
	PropServerGenerateEndpointDocFileExclOpenApi = "server.api-doc.file-excl-openapi-spec"

	// misoconfig-prop: the generated endpoint documentation should include app name as the path prefix | true
	// misoconfig-alias: server.generate-endpoint-doc.path-prefix-app | v0.2.0
	PropServerGenerateEndpointDocInclPrefix = "server.api-doc.path-prefix-app"

	// misoconfig-prop: server address specified in openapi json doc |
	// misoconfig-alias: server.generate-endpoint-doc.openapi-spec.server | v0.2.0
	PropServerGenerateEndpointDocOpenApiSpecServer = "server.api-doc.openapi-spec.server"

	// misoconfig-prop: path to generated openapi json for all endpoints |
	// misoconfig-alias: server.generate-endpoint-doc.openapi-spec.file | v0.2.0
	PropServerGenerateEndpointDocOpenApiSpecFile = "server.api-doc.openapi-spec.file"

	// misoconfig-prop: path patterns for endpoints in openapi json (`slice of string`) |
	// misoconfig-alias: server.generate-endpoint-doc.openapi-spec.path-patterns | v0.2.0
	PropServerGenerateEndpointDocOpenApiSpecPathPatterns = "server.api-doc.openapi-spec.path-patterns"

	// misoconfig-prop: file that contains the generated api doc golang demo
	PropServerApiDocGoFile = "server.api-doc.go.file"

	// misoconfig-prop: whether the generated api-doc golang demo file should compile | false
	PropServerApiDocGoCompileFile = "server.api-doc.go.compile-file"

	// misoconfig-prop: path patterns for endpoints that are written to api doc golang demo file
	PropServerApiDocGoPathPatterns = "server.api-doc.go.path-patterns"

	// misoconfig-prop: automatically map header values to request struct | true
	PropServerRequestAutoMapHeader = "server.request.mapping.header"

	// misoconfig-prop: disable gin's builtin validation | true
	PropServerGinValidationDisabled = "server.gin.validation.disabled"

	PropServerActualPort = "server.actual-port"
)

// misoconfig-section: Consul Configuration
const (

	// misoconfig-prop: enable Consul client, service registration and service discovery | false
	PropConsulEnabled = "consul.enabled"

	// misoconfig-prop: registered service name | `"${app.name}"`
	// misoconfig-alias: consul.registerName | v0.2.0
	PropConsuleRegisterName = "consul.register-name"

	// misoconfig-prop: registered service address | `"${server.host}"`
	// misoconfig-alias: consul.registerAddress | v0.2.0
	PropConsulRegisterAddress = "consul.register-address"

	// misoconfig-prop: consul server address | localhost:8500
	// misoconfig-alias: consul.consulAddress | v0.2.0
	PropConsulAddress = "consul.consul-address"

	// misoconfig-prop: for how long the current instance is deregistered after first health check failure | 30m
	// misoconfig-alias: consul.healthCheckFailedDeregisterAfter | v0.2.0
	PropConsulHealthCheckFailedDeregAfter = "consul.health-check-failed-deregister-time"

	// misoconfig-prop: fetch server list from Consul in ever N seconds | 30
	// misoconfig-alias: consul.fetchServerInterval | v0.2.0
	PropConsulFetchServerInterval = "consul.fetch-server-interval"

	// misoconfig-prop: enable endpoint for manual Consul service deregistration | false
	// misoconfig-alias: consul.enableDeregisterUrl | v0.2.0
	PropConsulEnableDeregisterUrl = "consul.enable-deregister-url"

	// misoconfig-prop: endpoint url for manual Consul service deregistration | /consul/deregister
	// misoconfig-alias: consul.deregisterUrl | v0.2.0
	PropConsulDeregisterUrl = "consul.deregister-url"

	// misoconfig-prop: instance metadata (`map[string]string`)
	PropConsulMetadata = "consul.metadata"
)

// misoconfig-section: Service Discovery Configuration
const (

	// misoconfig-prop: slice of service names that should be subcribed on startup
	PropSDSubscrbe = "service-discovery.subscribe"
)

// misoconfig-section: Tracing Configuration
const (

	// misoconfig-prop: propagation keys in trace (string slice) |
	PropTracingPropagationKeys = "tracing.propagation.keys"
)

// misoconfig-section: Metrics Configuration
const (

	// misoconfig-prop: enable metrics collection using prometheus | true
	PropMetricsEnabled = "metrics.enabled"

	// misoconfig-prop: route used to expose collected metrics | /metrics
	PropMetricsRoute = "metrics.route"

	// misoconfig-prop: enable authorization for metrics endpoint | false
	PropMetricsAuthEnabled = "metrics.auth.enabled"

	// misoconfig-prop: bearer token for metrics endpoint authorization
	PropMetricsAuthBearer = "metrics.auth.bearer"

	// misoconfig-prop: enable job that logs memory and cpu stats periodically (using `runtime/metrics`) | false
	PropMetricsEnableMemStatsLogJob = "metrics.memstat.log.job.enabled"

	// misoconfig-prop: job cron expresson for memory stats log job | 0/30 * * * * *
	PropMetricsMemStatsLogJobCron = "metrics.memstat.log.job.cron"
)

// misoconfig-section: Logging Configuration
const (

	// misoconfig-prop: log level | info
	PropLoggingLevel = "logging.level"

	// misoconfig-prop: path to rolling log file
	PropLoggingRollingFile = "logging.rolling.file"

	// misoconfig-prop: logs are written to log file only | false
	PropLoggingRollingFileOnly = "logging.file.log-file-only"

	// misoconfig-prop: max age of log files in days, 0 means files are retained forever | 0
	PropLoggingRollingFileMaxAge = "logging.file.max-age"

	// misoconfig-prop: max size of each log file (in mb) | 50
	PropLoggingRollingFileMaxSize = "logging.file.max-size"

	// misoconfig-prop: max number of backup log files | 10
	PropLoggingRollingFileMaxBackups = "logging.file.max-backups"

	// misoconfig-prop: rotate log file at every day 00:00 (local) | true
	PropLoggingRollingFileRotateDaily = "logging.file.rotate-daily"
)

// misoconfig-default-start
func init() {
	PostServerBootstrap(func(rail Rail) error {
		deprecatedProps := [][]string{}
		deprecatedProps = append(deprecatedProps, []string{"consul.registerName", "v0.2.0", PropConsuleRegisterName})
		deprecatedProps = append(deprecatedProps, []string{"consul.registerAddress", "v0.2.0", PropConsulRegisterAddress})
		deprecatedProps = append(deprecatedProps, []string{"consul.consulAddress", "v0.2.0", PropConsulAddress})
		deprecatedProps = append(deprecatedProps, []string{"consul.healthCheckFailedDeregisterAfter", "v0.2.0", PropConsulHealthCheckFailedDeregAfter})
		deprecatedProps = append(deprecatedProps, []string{"consul.fetchServerInterval", "v0.2.0", PropConsulFetchServerInterval})
		deprecatedProps = append(deprecatedProps, []string{"consul.enableDeregisterUrl", "v0.2.0", PropConsulEnableDeregisterUrl})
		deprecatedProps = append(deprecatedProps, []string{"consul.deregisterUrl", "v0.2.0", PropConsulDeregisterUrl})
		deprecatedProps = append(deprecatedProps, []string{"consul.healthCheckUrl", "v0.2.0", PropHealthCheckUrl})
		deprecatedProps = append(deprecatedProps, []string{"consul.healthCheckInterval", "v0.2.0", PropHealthCheckInterval})
		deprecatedProps = append(deprecatedProps, []string{"consul.healthCheckTimeout", "v0.2.0", PropHealthcheckTimeout})
		deprecatedProps = append(deprecatedProps, []string{"server.gracefulShutdownTimeSec", "v0.2.0", PropServerGracefulShutdownTimeSec})
		deprecatedProps = append(deprecatedProps, []string{"server.generate-endpoint-doc.enabled", "v0.2.0", PropServerGenerateEndpointDocEnabled})
		deprecatedProps = append(deprecatedProps, []string{"server.generate-endpoint-doc.web.enabled", "v0.2.0", PropServerGenerateEndpointDocApiEnabled})
		deprecatedProps = append(deprecatedProps, []string{"server.generate-endpoint-doc.file", "v0.2.0", PropServerGenerateEndpointDocFile})
		deprecatedProps = append(deprecatedProps, []string{"server.generate-endpoint-doc.file-excl-tclient-demo", "v0.2.0", PropServerGenerateEndpointDocFileExclTClientDemo})
		deprecatedProps = append(deprecatedProps, []string{"server.generate-endpoint-doc.file-excl-ng-client-demo", "v0.2.0", PropServerGenerateEndpointDocFileExclNgClientDemo})
		deprecatedProps = append(deprecatedProps, []string{"server.generate-endpoint-doc.file-excl-openapi-spec", "v0.2.0", PropServerGenerateEndpointDocFileExclOpenApi})
		deprecatedProps = append(deprecatedProps, []string{"server.generate-endpoint-doc.path-prefix-app", "v0.2.0", PropServerGenerateEndpointDocInclPrefix})
		deprecatedProps = append(deprecatedProps, []string{"server.generate-endpoint-doc.openapi-spec.server", "v0.2.0", PropServerGenerateEndpointDocOpenApiSpecServer})
		deprecatedProps = append(deprecatedProps, []string{"server.generate-endpoint-doc.openapi-spec.file", "v0.2.0", PropServerGenerateEndpointDocOpenApiSpecFile})
		deprecatedProps = append(deprecatedProps, []string{"server.generate-endpoint-doc.openapi-spec.path-patterns", "v0.2.0", PropServerGenerateEndpointDocOpenApiSpecPathPatterns})
		for _, p := range deprecatedProps {
			if HasProp(p[0]) {
				Errorf("Config prop: '%v' has been deprecated since '%v', please change to '%v'", p[0], p[1], p[2])
			}
		}
		return nil
	})

	SetDefProp(PropAppSlowBoostrapThresohold, "5s")
	SetDefProp(PropProdMode, true)
	SetDefProp(PropConsulEnabled, false)
	SetDefProp(PropConsuleRegisterName, "${app.name}")
	SetDefProp(PropConsulRegisterAddress, "${server.host}")
	SetDefProp(PropConsulAddress, "localhost:8500")
	SetDefProp(PropConsulHealthCheckFailedDeregAfter, "30m")
	SetDefProp(PropConsulFetchServerInterval, 30)
	SetDefProp(PropConsulEnableDeregisterUrl, false)
	SetDefProp(PropConsulDeregisterUrl, "/consul/deregister")
	SetDefProp(PropLoggingLevel, "info")
	SetDefProp(PropLoggingRollingFileOnly, false)
	SetDefProp(PropLoggingRollingFileMaxAge, 0)
	SetDefProp(PropLoggingRollingFileMaxSize, 50)
	SetDefProp(PropLoggingRollingFileMaxBackups, 10)
	SetDefProp(PropLoggingRollingFileRotateDaily, true)
	SetDefProp(PropMetricsEnabled, true)
	SetDefProp(PropMetricsRoute, "/metrics")
	SetDefProp(PropMetricsAuthEnabled, false)
	SetDefProp(PropMetricsEnableMemStatsLogJob, false)
	SetDefProp(PropMetricsMemStatsLogJobCron, "0/30 * * * * *")
	SetDefProp(PropServerEnabled, true)
	SetDefProp(PropServerHost, "127.0.0.1")
	SetDefProp(PropServerPort, 8080)
	SetDefProp(PropHealthCheckUrl, "/health")
	SetDefProp(PropHealthCheckInterval, "5s")
	SetDefProp(PropHealthcheckTimeout, "3s")
	SetDefProp(PropServerLogRoutes, false)
	SetDefProp(PropServerGracefulShutdownTimeSec, 30)
	SetDefProp(PropServerPerfEnabled, false)
	SetDefProp(PropServerPropagateInboundTrace, true)
	SetDefProp(PropServerRequestValidateEnabled, true)
	SetDefProp(PropServerRequestLogEnabled, false)
	SetDefProp(PropServerPprofEnabled, false)
	SetDefProp(PropServerGenerateEndpointDocEnabled, true)
	SetDefProp(PropServerGenerateEndpointDocApiEnabled, true)
	SetDefProp(PropServerGenerateEndpointDocFileExclTClientDemo, false)
	SetDefProp(PropServerGenerateEndpointDocFileExclNgClientDemo, false)
	SetDefProp(PropServerGenerateEndpointDocFileExclOpenApi, true)
	SetDefProp(PropServerGenerateEndpointDocInclPrefix, true)
	SetDefProp(PropServerApiDocGoCompileFile, false)
	SetDefProp(PropServerRequestAutoMapHeader, true)
	SetDefProp(PropServerGinValidationDisabled, true)
}

// misoconfig-default-end
