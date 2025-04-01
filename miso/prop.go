package miso

// misoapi-config-section: Common Configuration
const (

	// misoapi-config: name of the application
	PropAppName = "app.name"

	// misoapi-config: whether production mode is turned on | true
	PropProdMode = "mode.production"

	// misoapi-config: extra config files that should be loaded
	PropConfigExtraFiles = "config.extra.files"
)

// misoapi-config-section: Web Server Configuration
const (

	// misoapi-config: enable http server | true
	PropServerEnabled = "server.enabled"

	// misoapi-config: http server host | 127.0.0.1
	PropServerHost = "server.host"

	// misoapi-config: http server port | 8080
	PropServerPort = "server.port"

	// misoapi-config: http server bearer authorization token for all endpoints |
	PropServerAuthBearer = "server.auth.bearer"

	// misoapi-config: time wait (in second) before whole app server shutdown (previously, before `v0.1.12`, it only applies to the http server) | 30
	PropServerGracefulShutdownTimeSec = "server.gracefulShutdownTimeSec"

	// misoapi-config: logs time duration for each inbound http request | false
	PropServerPerfEnabled = "server.perf.enabled"

	// misoapi-config: propagate trace info from inbound requests | true
	PropServerPropagateInboundTrace = "server.trace.inbound.propagate"

	// misoapi-config: enable inbound request parameter validation | true
	PropServerRequestValidateEnabled = "server.validate.request.enabled"

	// misoapi-config: enable server request log | false
	PropServerRequestLogEnabled = "server.request-log.enabled"

	// misoapi-config: enable pprof (exposed using endpoint '/debug/pprof'); in non-prod mode, it's always enabled | false
	PropServerPprofEnabled = "server.pprof.enabled"

	// misoapi-config: enable bearer authentication for pprof endpoints | false
	PropServerPprofAuthEnabled = "server.pprof.auth.enabled"

	// misoapi-config: bearer token for pprof endpoints' authentication
	PropServerPprofAuthBearer = "server.pprof.auth.bearer"

	// misoapi-config: generate endpoint documentation | true
	PropServerGenerateEndpointDocEnabled = "server.generate-endpoint-doc.enabled"

	// misoapi-config: build webpage for the generated endpoint documentation | true
	PropServerGenerateEndpointDocApiEnabled = "server.generate-endpoint-doc.web.enabled"

	// misoapi-config: generate markdown endpoint documentation and save the doc to the specified file
	PropServerGenerateEndpointDocFile = "server.generate-endpoint-doc.file"

	// misoapi-config: whether the generated endpoint documentation should include app name as the path prefix | true
	PropServerGenerateEndpointDocInclPrefix = "server.generate-endpoint-doc.path-prefix-app"

	// misoapi-config: automatically map header values to request struct | true
	PropServerRequestAutoMapHeader = "server.request.mapping.header"

	// misoapi-config: disable gin's builtin validation | true
	PropServerGinValidationDisabled = "server.gin.validation.disabled"

	PropServerActualPort = "server.actual-port"
)

// misoapi-config-section: Consul Configuration
const (

	// misoapi-config: enable Consul client, service registration and service discovery | false
	PropConsulEnabled = "consul.enabled"

	// misoapi-config: 	registered service name | `${app.name}`
	PropConsuleRegisterName = "consul.registerName"

	// misoapi-config: registered service address | `${server.host}:${server.port}`
	PropConsulRegisterAddress = "consul.registerAddress"

	// misoapi-config: consul server address | `localhost:8500`
	PropConsulAddress = "consul.consulAddress"

	// misoapi-config: health check url | `/health`
	PropConsulHealthcheckUrl = "consul.healthCheckUrl"

	// misoapi-config: health check interval | 5s
	PropConsulHealthCheckInterval = "consul.healthCheckInterval"

	// misoapi-config: health check timeout | 3s
	PropConsulHealthcheckTimeout = "consul.healthCheckTimeout"

	// misoapi-config:  for how long the current instance is deregistered after first health check failure | 30m
	PropConsulHealthCheckFailedDeregAfter = "consul.healthCheckFailedDeregisterAfter"

	// misoapi-config: 	register default health check endpoint on startup | true
	PropConsulRegisterDefaultHealthcheck = "consul.registerDefaultHealthCheck"

	// misoapi-config: fetch server list from Consul in ever N seconds | 30
	PropConsulFetchServerInterval = "consul.fetchServerInterval"

	// misoapi-config: enable endpoint for manual Consul service deregistration | false
	PropConsulEnableDeregisterUrl = "consul.enableDeregisterUrl"

	// misoapi-config: endpoint url for manual Consul service deregistration | `/consul/deregister`
	PropConsulDeregisterUrl = "consul.deregisterUrl"

	// misoapi-config: instance metadata (`map[string]string`)
	PropConsulMetadata = "consul.metadata"
)

// misoapi-config-section: Service Discovery Configuration
const (

	// misoapi-config: slice of service names that should be subcribed on startup
	PropSDSubscrbe = "service-discovery.subscribe"
)

// misoapi-config-section: Tracing Configuration
const (

	// misoapi-config: propagation keys in trace (string slice) | `X-B3-TraceId`, `X-B3-SpanId`
	PropTracingPropagationKeys = "tracing.propagation.keys"
)

// misoapi-config-section: Metrics Configuration
const (

	// misoapi-config:  enable metrics collection using prometheus | true
	PropMetricsEnabled = "metrics.enabled"

	// misoapi-config: route used to expose collected metrics | /metrics
	PropMetricsRoute = "metrics.route"

	// misoapi-config: enable authorization for metrics endpoint | false
	PropMetricsAuthEnabled = "metrics.auth.enabled"

	// misoapi-config: bearer token for metrics endpoint authorization
	PropMetricsAuthBearer = "metrics.auth.bearer"

	// misoapi-config: enable job that logs memory stats periodically (using `runtime/metrics`) | false
	PropMetricsEnableMemStatsLogJob = "metrics.memstat.log.job.enabled"

	// misoapi-config: job cron expresson for memory stats log job | `0 */1 * * * *`
	PropMetricsMemStatsLogJobCron = "metrics.memstat.log.job.cron"
)

// misoapi-config-section: Logging Configuration
const (

	// misoapi-config: log level | info
	PropLoggingLevel = "logging.level"

	// misoapi-config: path to rolling log file
	PropLoggingRollingFile = "logging.rolling.file"

	// misoapi-config: max age of log files in days | 0 (files are retained forever)
	PropLoggingRollingFileMaxAge = "logging.file.max-age"

	// misoapi-config: max size of each log file (in mb) | 50
	PropLoggingRollingFileMaxSize = "logging.file.max-size"

	// misoapi-config: max number of backup log files | 10
	PropLoggingRollingFileMaxBackups = "logging.file.max-backups"

	// misoapi-config: rotate log file at every day 00:00 (local) | true
	PropLoggingRollingFileRotateDaily = "logging.file.rotate-daily"
)

func init() {
	SetDefProp(PropProdMode, true)

	SetDefProp(PropServerEnabled, true)
	SetDefProp(PropServerHost, "127.0.0.1")
	SetDefProp(PropServerPort, 8080)
	SetDefProp(PropServerPerfEnabled, false)
	SetDefProp(PropServerPropagateInboundTrace, true)
	SetDefProp(PropServerRequestValidateEnabled, true)
	SetDefProp(PropServerPprofEnabled, false)
	SetDefProp(PropServerRequestAutoMapHeader, true)
	SetDefProp(PropServerGinValidationDisabled, true)

	SetDefProp(PropLoggingRollingFileMaxAge, 0)
	SetDefProp(PropLoggingRollingFileMaxSize, 50)
	SetDefProp(PropLoggingRollingFileMaxBackups, 10)
	SetDefProp(PropLoggingRollingFileRotateDaily, true)

	SetDefProp(PropMetricsEnabled, true)
	SetDefProp(PropMetricsRoute, "/metrics")

	SetDefProp(PropMetricsEnableMemStatsLogJob, false)
	SetDefProp(PropMetricsMemStatsLogJobCron, "0 */1 * * * *")

	SetDefProp(PropConsulEnabled, false)
	SetDefProp(PropConsulAddress, "localhost:8500")
	SetDefProp(PropConsulHealthcheckUrl, "/health")
	SetDefProp(PropConsulHealthCheckInterval, "5s")
	SetDefProp(PropConsulHealthcheckTimeout, "3s")
	SetDefProp(PropConsulHealthCheckFailedDeregAfter, "30m")
	SetDefProp(PropConsulRegisterDefaultHealthcheck, true)
	SetDefProp(PropConsulFetchServerInterval, 30)
	SetDefProp(PropConsulDeregisterUrl, "/consul/deregister")
	SetDefProp(PropConsulEnableDeregisterUrl, false)
	SetDefProp(PropConsuleRegisterName, "${app.name}")

	SetDefProp(PropServerGracefulShutdownTimeSec, 30)

	SetDefProp(PropServerGenerateEndpointDocEnabled, true)
	SetDefProp(PropServerGenerateEndpointDocApiEnabled, true)
	SetDefProp(PropServerGenerateEndpointDocInclPrefix, true)
}
