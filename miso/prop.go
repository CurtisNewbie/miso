package miso

// misoconfig-section: Common Configuration
const (

	// misoconfig-prop: name of the application
	PropAppName = "app.name"

	// misoconfig-prop: whether production mode is turned on | true
	PropProdMode = "mode.production"

	// misoconfig-prop: extra config files that should be loaded
	PropConfigExtraFiles = "config.extra.files"
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
	PropHealthCheckUrl = "server.health-check-url"

	// misoconfig-prop: log all http server routes in INFO level | false
	PropServerLogRoutes = "server.log-routes"

	// misoconfig-prop: http server bearer authorization token for all endpoints |
	PropServerAuthBearer = "server.auth.bearer"

	// misoconfig-prop: time wait (in second) before whole app server shutdown (previously, before `v0.1.12`, it only applies to the http server) | 30
	PropServerGracefulShutdownTimeSec = "server.gracefulShutdownTimeSec"

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

	// misoconfig-prop: bearer token for pprof endpoints' authentication
	PropServerPprofAuthBearer = "server.pprof.auth.bearer"

	// misoconfig-prop: generate api doc | true
	PropServerGenerateEndpointDocEnabled = "server.generate-endpoint-doc.enabled"

	// misoconfig-prop: build webpage for the generated api doc | true
	PropServerGenerateEndpointDocApiEnabled = "server.generate-endpoint-doc.web.enabled"

	// misoconfig-prop: generate markdown api doc to the specified file
	PropServerGenerateEndpointDocFile = "server.generate-endpoint-doc.file"

	// misoconfig-prop: whether the markdown api doc should exclude miso.TClient demo | false
	PropServerGenerateEndpointDocFileExclTClientDemo = "server.generate-endpoint-doc.file-excl-tclient-demo"

	// misoconfig-prop: whether the markdown api doc should exclude Angular HttpClient demo | false
	PropServerGenerateEndpointDocFileExclNgClientDemo = "server.generate-endpoint-doc.file-excl-ng-client-demo"

	// misoconfig-prop: whether the markdown api doc should exclude openapi json for each endpoint | true
	PropServerGenerateEndpointDocFileExclOpenApi = "server.generate-endpoint-doc.file-excl-openapi-spec"

	// misoconfig-prop: whether the generated endpoint documentation should include app name as the path prefix | true
	PropServerGenerateEndpointDocInclPrefix = "server.generate-endpoint-doc.path-prefix-app"

	// misoconfig-prop: server address specified in openapi json doc |
	PropServerGenerateEndpointDocOpenApiSpecServer = "server.generate-endpoint-doc.openapi-spec.server"

	// misoconfig-prop: path to generated openapi json for all endpoints |
	PropServerGenerateEndpointDocOpenApiSpecFile = "server.generate-endpoint-doc.openapi-spec.file"

	// misoconfig-prop: path patterns for endpoints in openapi json (`slice of string`) |
	PropServerGenerateEndpointDocOpenApiSpecPathPatterns = "server.generate-endpoint-doc.openapi-spec.path-patterns"

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
	PropConsuleRegisterName = "consul.registerName"

	// misoconfig-prop: registered service address | `"${server.host}"`
	PropConsulRegisterAddress = "consul.registerAddress"

	// misoconfig-prop: consul server address | localhost:8500
	PropConsulAddress = "consul.consulAddress"

	// deprecated: changed to "server.health-check-url"
	//
	// misoconfig-prop: health check url. (deprecated since v0.1.23, use `server.health-check-url` instead) |
	PropConsulHealthCheckUrl = "consul.healthCheckUrl"

	// misoconfig-prop: health check interval | 5s
	PropConsulHealthCheckInterval = "consul.healthCheckInterval"

	// misoconfig-prop: health check timeout | 3s
	PropConsulHealthcheckTimeout = "consul.healthCheckTimeout"

	// misoconfig-prop: for how long the current instance is deregistered after first health check failure | 30m
	PropConsulHealthCheckFailedDeregAfter = "consul.healthCheckFailedDeregisterAfter"

	// misoconfig-prop: fetch server list from Consul in ever N seconds | 30
	PropConsulFetchServerInterval = "consul.fetchServerInterval"

	// misoconfig-prop: enable endpoint for manual Consul service deregistration | false
	PropConsulEnableDeregisterUrl = "consul.enableDeregisterUrl"

	// misoconfig-prop: endpoint url for manual Consul service deregistration | /consul/deregister
	PropConsulDeregisterUrl = "consul.deregisterUrl"

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
	SetDefProp(PropProdMode, true)
	SetDefProp(PropConsulEnabled, false)
	SetDefProp(PropConsuleRegisterName, "${app.name}")
	SetDefProp(PropConsulRegisterAddress, "${server.host}")
	SetDefProp(PropConsulAddress, "localhost:8500")
	SetDefProp(PropConsulHealthCheckInterval, "5s")
	SetDefProp(PropConsulHealthcheckTimeout, "3s")
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
	SetDefProp(PropServerRequestAutoMapHeader, true)
	SetDefProp(PropServerGinValidationDisabled, true)
}

// misoconfig-default-end
