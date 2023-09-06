package miso

const (
	// name of the profile used
	PROP_PROFILE = "profile"
	// whether production mode is turned on (true/false)
	PROP_PRODUCTION_MODE = "mode.production"

	/*
		------------------------------------

		Prop for App

		------------------------------------
	*/
	PROP_APP_NAME = "app.name"

	/*
		------------------------------------

		Prop for Consul

		------------------------------------
	*/
	PROP_CONSUL_ENABLED                        = "consul.enabled"
	PROP_CONSUL_REGISTER_NAME                  = "consul.registerName"
	PROP_CONSUL_REGISTER_ADDRESS               = "consul.registerAddress"
	PROP_CONSUL_CONSUL_ADDRESS                 = "consul.consulAddress"
	PROP_CONSUL_HEALTHCHECK_URL                = "consul.healthCheckUrl"
	PROP_CONSUL_HEALTHCHECK_INTERVAL           = "consul.healthCheckInterval"
	PROP_CONSUL_HEALTHCHECK_TIMEOUT            = "consul.healthCheckTimeout"
	PROP_CONSUL_HEALTHCHECK_FAILED_DEREG_AFTER = "consul.healthCheckFailedDeregisterAfter"
	PROP_CONSUL_REGISTER_DEFAULT_HEALTHCHECK   = "consul.registerDefaultHealthCheck"

	/*
		------------------------------------

		Prop for Redis

		------------------------------------
	*/
	PROP_REDIS_ENABLED  = "redis.enabled"
	PROP_REDIS_ADDRESS  = "redis.address"
	PROP_REDIS_PORT     = "redis.port"
	PROP_REDIS_USERNAME = "redis.username"
	PROP_REDIS_PASSWORD = "redis.password"
	PROP_REDIS_DATABASE = "redis.database"

	/*
		------------------------------------

		Prop for MySQL

		------------------------------------
	*/
	PROP_MYSQL_ENABLED    = "mysql.enabled"
	PROP_MYSQL_USER       = "mysql.user"
	PROP_MYSQL_PASSWORD   = "mysql.password"
	PROP_MYSQL_DATABASE   = "mysql.database"
	PROP_MYSQL_HOST       = "mysql.host"
	PROP_MYSQL_PORT       = "mysql.port"
	PROP_MYSQL_CONN_PARAM = "mysql.connection.parameters"

	/*
		------------------------------------

		Prop for Server

		------------------------------------
	*/
	PROP_SERVER_ENABLED                    = "server.enabled"
	PROP_SERVER_HOST                       = "server.host"
	PROP_SERVER_PORT                       = "server.port"
	PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC = "server.gracefulShutdownTimeSec"
	PROP_SERVER_PERF_ENABLED               = "server.perf.enabled"
	PROP_SERVER_PROPAGATE_INBOUND_TRACE    = "server.trace.inbound.propagate"

	/*
		------------------------------------

		Prop for SQLite

		------------------------------------
	*/
	PROP_SQLITE_FILE = "sqlite.file"

	/*
		------------------------------------

		Prop for RabbitMQ

		------------------------------------
	*/
	PROP_RABBITMQ_ENABLED      = "rabbitmq.enabled"
	PROP_RABBITMQ_HOST         = "rabbitmq.host"
	PROP_RABBITMQ_PORT         = "rabbitmq.port"
	PROP_RABBITMQ_USERNAME     = "rabbitmq.username"
	PROP_RABBITMQ_PASSWORD     = "rabbitmq.password"
	PROP_RABBITMQ_VHOST        = "rabbitmq.vhost"
	PROP_RABBITMQ_CONSUMER_QOS = "rabbitmq.consumer.qos"

	/*
		------------------------------------

		Prop for Tracing

		------------------------------------
	*/
	PROP_TRACING_PROPAGATION_KEYS = "tracing.propagation.keys"

	/*
		------------------------------------

		Prop for Logging

		------------------------------------
	*/
	PROP_LOGGING_ROLLING_FILE = "logging.rolling.file"
	PROP_LOGGING_LEVEL        = "logging.level"

	/*
		------------------------------------

		Prop for distributed task scheduling

		------------------------------------
	*/
	PROP_TASK_SCHEDULING_ENABLED = "task.scheduling.enabled"
	PROP_TASK_SCHEDULING_GROUP   = "task.scheduling.group"

	/*
		------------------------------------

		Prop for JWT

		------------------------------------
	*/
	PROP_JWT_PUBLIC_KEY  = "jwt.key.public"
	PROP_JWT_PRIVATE_KEY = "jwt.key.private"
	PROP_JWT_ISSUER      = "jwt.key.issuer"

	/*
		------------------------------------

		Prop for Metrics & Prometheus

		------------------------------------
	*/
	PROP_METRICS_ENABLED = "metrics.enabled"
	PROP_PROM_ROUTE      = "metrics.route"
)
