package miso

const (
	// whether production mode is turned on (true/false)
	PropProdMode = "mode.production"

	/*
		------------------------------------

		Prop for App

		------------------------------------
	*/
	PropAppName = "app.name"

	/*
		------------------------------------

		Prop for Consul

		------------------------------------
	*/
	PropConsulEnabled                     = "consul.enabled"
	PropConsuleRegisterName               = "consul.registerName"
	PropConsulRegisterAddress             = "consul.registerAddress"
	PropConsulAddress                     = "consul.consulAddress"
	PropConsulHealthcheckUrl              = "consul.healthCheckUrl"
	PropConsulHealthCheckInterval         = "consul.healthCheckInterval"
	PropConsulHealthcheckTimeout          = "consul.healthCheckTimeout"
	PropConsulHealthCheckFailedDeregAfter = "consul.healthCheckFailedDeregisterAfter"
	PropConsulRegisterDefaultHealthcheck  = "consul.registerDefaultHealthCheck"
	PropConsulFetchServerInterval         = "consul.fetchServerInterval"
	PropConsulDeregisterUrl               = "consul.deregisterUrl"
	PropConsulEnableDeregisterUrl         = "consul.enableDeregisterUrl"
	PropConsulMetadata                    = "consul.metadata"

	/*
		------------------------------------

		Prop for ServiceDiscovery

		------------------------------------
	*/
	PropSDSubscrbe = "service-discovery.subscribe"

	/*
		------------------------------------

		Prop for Server

		------------------------------------
	*/

	PropServerEnabled                    = "server.enabled"
	PropServerHost                       = "server.host"
	PropServerPort                       = "server.port"
	PropServerActualPort                 = "server.actual-port"
	PropServerGracefulShutdownTimeSec    = "server.gracefulShutdownTimeSec"
	PropServerPerfEnabled                = "server.perf.enabled"
	PropServerRequestLogEnabled          = "server.request-log.enabled"
	PropServerPropagateInboundTrace      = "server.trace.inbound.propagate"
	PropServerRequestValidateEnabled     = "server.validate.request.enabled"
	PropServerPprofEnabled               = "server.pprof.enabled"
	PropServerGenerateEndpointDocEnabled = "server.generate-endpoint-doc.enabled"
	PropServerGenerateEndpointDocFile    = "server.generate-endpoint-doc.file"
	PropServerRequestAutoMapHeader       = "server.request.mapping.header"
	PropServerGinValidationDisabled      = "server.gin.validation.disabled"

	/*
		------------------------------------

		Prop for Tracing

		------------------------------------
	*/

	PropTracingPropagationKeys = "tracing.propagation.keys"

	/*
		------------------------------------

		Prop for Logging

		------------------------------------
	*/

	PropLoggingLevel                  = "logging.level"
	PropLoggingRollingFile            = "logging.rolling.file"
	PropLoggingRollingFileMaxAge      = "logging.file.max-age"
	PropLoggingRollingFileMaxSize     = "logging.file.max-size"
	PropLoggingRollingFileMaxBackups  = "logging.file.max-backups"
	PropLoggingRollingFileRotateDaily = "logging.file.rotate-daily"

	/*
		------------------------------------

		Prop for Metrics & Prometheus

		------------------------------------
	*/

	PropMetricsEnabled              = "metrics.enabled"
	PropMetricsRoute                = "metrics.route"
	PropMetricsAuthEnabled          = "metrics.auth.enabled"
	PropMetricsAuthBearer           = "metrics.auth.bearer"
	PropMetricsEnableMemStatsLogJob = "metrics.memstat.log.job.enabled"
	PropMetricsMemStatsLogJobCron   = "metrics.memstat.log.job.cron"

	/*
		------------------------------------

		Prop for Configuration

		------------------------------------
	*/

	PropConfigExtraFiles = "config.extra.files"
)
