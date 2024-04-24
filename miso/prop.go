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

		Prop for Redis

		------------------------------------
	*/
	PropRedisEnabled  = "redis.enabled"
	PropRedisAddress  = "redis.address"
	PropRedisPort     = "redis.port"
	PropRedisUsername = "redis.username"
	PropRedisPassword = "redis.password"
	PropRedisDatabase = "redis.database"

	/*
		------------------------------------

		Prop for MySQL

		------------------------------------
	*/
	PropMySQLEnabled      = "mysql.enabled"
	PropMySQLUser         = "mysql.user"
	PropMySQLPassword     = "mysql.password"
	PropMySQLSchema       = "mysql.database"
	PropMySQLHost         = "mysql.host"
	PropMySQLPort         = "mysql.port"
	PropMySQLConnParam    = "mysql.connection.parameters"
	PropMySQLConnLifetime = "mysql.connection.lifetime"
	PropMySQLMaxOpenConns = "mysql.connection.open.max"
	PropMySQLMaxIdleConns = "mysql.connection.idle.max"

	/*
		------------------------------------

		Prop for Server

		------------------------------------
	*/

	PropServerEnabled                    = "server.enabled"
	PropServerHost                       = "server.host"
	PropServerPort                       = "server.port"
	PropServerGracefulShutdownTimeSec    = "server.gracefulShutdownTimeSec"
	PropServerPerfEnabled                = "server.perf.enabled"
	PropServerRequestLogEnabled          = "server.request-log.enabled"
	PropServerPropagateInboundTrace      = "server.trace.inbound.propagate"
	PropServerRequestValidateEnabled     = "server.validate.request.enabled"
	PropServerPprofEnabled               = "server.pprof.enabled"
	PropServerGenerateEndpointDocEnabled = "server.generate-endpoint-doc.enabled"
	PropServerRequestAutoMapHeader       = "server.request.mapping.header"
	PropServerGinValidationDisabled      = "server.gin.validation.disabled"

	/*
		------------------------------------

		Prop for SQLite

		------------------------------------
	*/

	PropSqliteFile       = "sqlite.file"
	PropSqliteWalEnabled = "sqlite.wal.enabled"

	/*
		------------------------------------

		Prop for RabbitMQ

		------------------------------------
	*/

	PropRabbitMqEnabled     = "rabbitmq.enabled"
	PropRabbitMqHost        = "rabbitmq.host"
	PropRabbitMqPort        = "rabbitmq.port"
	PropRabbitMqUsername    = "rabbitmq.username"
	PropRabbitMqPassword    = "rabbitmq.password"
	PropRabbitMqVhost       = "rabbitmq.vhost"
	PropRabbitMqConsumerQos = "rabbitmq.consumer.qos"

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

		Prop for distributed task scheduling

		------------------------------------
	*/

	PropTaskSchedulingEnabled = "task.scheduling.enabled"
	PropTaskSchedulingGroup   = "task.scheduling.group"

	/*
		------------------------------------

		Prop for JWT

		------------------------------------
	*/

	PropJwtPublicKey  = "jwt.key.public"
	PropJwtPrivateKey = "jwt.key.private"
	PropJwtIssue      = "jwt.key.issuer"

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
