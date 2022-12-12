package common

const (
	// name of the profile used
	PROP_PROFILE = "profile"
	// whether production mode is turned on (true/false)
	PROP_PRODUCTION_MODE = "mode.production"

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
	PROP_MYSQL_ENABLED  = "mysql.enabled"
	PROP_MYSQL_USER     = "mysql.user"
	PROP_MYSQL_PASSWORD = "mysql.password"
	PROP_MYSQL_DATABASE = "mysql.database"
	PROP_MYSQL_HOST     = "mysql.host"
	PROP_MYSQL_PORT     = "mysql.port"

	/*
		------------------------------------

		Prop for Server

		------------------------------------
	*/
	PROP_SERVER_HOST                       = "server.host"
	PROP_SERVER_PORT                       = "server.port"
	PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC = "server.gracefulShutdownTimeSec"

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
	PROP_RABBITMQ_HOST     = "rabbitmq.host"
	PROP_RABBITMQ_PORT     = "rabbitmq.port"
	PROP_RABBITMQ_USERNAME = "rabbitmq.username"
	PROP_RABBITMQ_PASSWORD = "rabbitmq.password"
	PROP_RABBITMQ_VHOST    = "rabbitmq.vhost"

	/*
		durable, non auto-delete queue name (slice)

			{
				"rabbitmq": {
					"declaration": {
						"queue": [
							"my-first-queue",
							"my-second-queue"
						]
					}
				}
			}
	*/
	PROP_RABBITMQ_DEC_QUEUE = "rabbitmq.declaration.queue"

	/*
		durable, non auto-delete, 'direct' exchange name (slice)

			{
				"rabbitmq": {
					"declaration": {
						"exchange": [
							"my-exchange-one",
							"my-exchange-two"
						]
					}
				}
			}
	*/
	PROP_RABBITMQ_DEC_EXCHANGE = "rabbitmq.declaration.exchange"

	/*
		Binding between queue and exchange, it's always queue -> exchange

		So, in our json configuration:

			{
				"rabbitmq": {
					"declaration": {
						"binding": {
							"my-first-queue": {
								"key": "myKey1",
								"exchange": "my-exchange-one"
							},
							"my-second-queue": {
								"key": "mykey2",
								"exchange": "my-exchange-two"
							}
						}
					}
				}
			}
	*/
	PROP_RABBITMQ_DEC_BINDING = "rabbitmq.declaration.binding"
)
