package miso

const (
	// name of the profile used
	PropProfile = "profile"
	// whether production mode is turned on (true/false)
	PropProductinMode = "mode.production"

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
	PropRedisDatabas  = "redis.database"

	/*
		------------------------------------

		Prop for MySQL

		------------------------------------
	*/
	PropMySqlEnabled   = "mysql.enabled"
	PropMySqlUser      = "mysql.user"
	PropMySqlPassword  = "mysql.password"
	PropMySqldatabase  = "mysql.database"
	PropMySqlHost      = "mysql.host"
	PropMySqlPort      = "mysql.port"
	PropMySqlConnParam = "mysql.connection.parameters"

	/*
		------------------------------------

		Prop for Server

		------------------------------------
	*/
	PropServerEnabled                 = "server.enabled"
	PropServerHost                    = "server.host"
	PropServerPort                    = "server.port"
	PropServerGracefulShutdownTimeSec = "server.gracefulShutdownTimeSec"
	PropServerPerfEnabled             = "server.perf.enabled"
	PropServerRequestLogEnabled       = "server.request-log.enabled"
	PropServerPropagateInboundTrace   = "server.trace.inbound.propagate"
	PropServerRequestValidateEnabled  = "server.validate.request.enabled"

	/*
		------------------------------------

		Prop for SQLite

		------------------------------------
	*/
	PropSqliteFile = "sqlite.file"

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
	PropLoggingFile                  = "logging.level"
	PropLoggingRollingFile           = "logging.rolling.file"
	PropLoggingRollingFileMaxAge     = "logging.file.max-age"
	PropLoggingRollingFileMaxSize    = "logging.file.max-size"
	PropLoggingRollingFileMaxBackups = "logging.file.max-backups"

	/*
		------------------------------------

		Prop for distributed task scheduling

		------------------------------------
	*/
	PropTaskSchedulingEnabled = "task.scheduling.enabled"
	ProptaskSchedulingGroup   = "task.scheduling.group"

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
	PropMetricsEnabled = "metrics.enabled"
	PropPromRoute      = "metrics.route"

	/*
		------------------------------------

		Prop for Configuration

		------------------------------------
	*/
	PropConfigExtraFiles = "config.extra.files"
)
