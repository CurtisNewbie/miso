# Configuration

## Command Line Arguments

- To specify where the config file is: `configFile=${PATH_TO_CONFIG_FILE}`

By convention, without specifiying where the configuration file is, it looks for the file `conf.yml` and load the configuration properties from it.

e.g.,

```sh
# the configFile is specified, file '/myapp/my-conf.yml' is loaded.
./main configFile=/myapp/my-conf.yml

# using default `conf.yml` file
./main
```

Properties loaded from configuration file can also be overriden by cli arguments (e.g., `KEY=VALUE`) and environment variables in `server.BootstrapServer(...)` method.

e.g.,

```sh
./main mode.production=true
```

Notice that if you have more than one configuration file to load, you can use `config.extra.files` configuration property.

## Common Configuration

| property           | description                                               | default value |
| ------------------ | --------------------------------------------------------- | ------------- |
| app.name           | name of the application                                   |               |
| mode.production    | whether production mode is turned on                      | false         |
| config.extra.files | config files that should be loaded beside the default one |               |

## Web Server Configuration

| property                        | description                                           | default value |
| ------------------------------- | ----------------------------------------------------- | ------------- |
| server.enabled                  | enable http server                                    | true          |
| server.host                     | http server host                                      | 0.0.0.0       |
| server.port                     | http server port                                      | 8080          |
| server.gracefulShutdownTimeSec  | time wait (in second) before server shutdown          | 30            |
| server.perf.enabled             | enable logging time took for each http server request | false         |
| server.trace.inbound.propagate  | propagate trace info from inbound requests            | true          |
| server.validate.request.enabled | enable server request parameter validation            | true          |
| server.request-log.enabled      | enable server request log enabled                     | false         |

## Consul Configuration

| property                                | description                                                          | default value                   |
| --------------------------------------- | -------------------------------------------------------------------- | ------------------------------- |
| consul.enabled                          | whether Consul is enabled                                            | false                           |
| consul.registerName                     | registered service name                                              | `${app.name}`                   |
| consul.registerAddress                  | registered service address                                           | `${server.host}:${server.port}` |
| consul.consulAddress                    | address of the Consul server                                         | `localhost:8500`                |
| consul.healthCheckUrl                   | health check url                                                     | `/health`                       |
| consul.healthCheckInterval              | health check interval                                                | 15s                             |
| consul.healthCheckTimeout               | health check timeout                                                 | 3s                              |
| consul.healthCheckFailedDeregisterAfter | timeout for current service to deregister after health check failure | 120s                            |
| consul.registerDefaultHealthCheck       | register default health check endpoint on startup                    | true                            |
| consul.fetchServerInterval              | fetch server list from consul in ever N seconds                      | 15                              |
| consul.enableDeregisterUrl              | enable endpoint for manual consul service deregistration             | false                           |
| consul.deregisterUrl                    | endpoint url for manual consul service deregistration                | `/consul/deregister`            |


## MySQL Configuration

| property                    | description                                 | default value                                                                          |
| --------------------------- | ------------------------------------------- | -------------------------------------------------------------------------------------- |
| mysql.enabled               | whether MySQL is enabled                    | false                                                                                  |
| mysql.user                  | username                                    | root                                                                                   |
| mysql.password              | password                                    |                                                                                        |
| mysql.database              | database                                    |                                                                                        |
| mysql.host                  | host                                        | `localhost`                                                                            |
| mysql.port                  | port                                        | 3306                                                                                   |
| mysql.connection.parameters | query parameters declared on connection url | `charset=utf8mb4&parseTime=True&loc=Local&readTimeout=30s&writeTimeout=30s&timeout=3s` |

## Redis Configuration

| property       | description              | default value |
| -------------- | ------------------------ | ------------- |
| redis.enabled  | whether Redis is enabled | false         |
| redis.address  | address of Redis server  | `localhost`   |
| redis.port     | port of Redis server     | 6379          |
| redis.username | username                 |               |
| redis.password | password                 |               |
| redis.database | 0                        |               |

## RabbitMQ Configuration

| property              | description                        | default value |
| --------------------- | ---------------------------------- | ------------- |
| rabbitmq.enabled      | whether RabbitMQ client is enabled | false         |
| rabbitmq.host         | host of the RabbitMQ server        | `localhost`   |
| rabbitmq.port         | port of the RabbitMQ server        | 5672          |
| rabbitmq.username     | username used to connect to server |               |
| rabbitmq.password     | password used to connect to server |               |
| rabbitmq.vhost        | virtual host                       |               |
| rabbitmq.consumer.qos | consumer QOS                       | 68            |

Miso's integration with RabbitMQ supports delayed message redelivery (messages that can't be handled without error), the delay is currently 10 seconds. This is to prevent server being flooded with redelivered messages, this is not configurable though.

## SQLite Configuration

| property    | description                  | default value |
| ----------- | ---------------------------- | ------------- |
| sqlite.file | path to SQLite database file |               |

## Logging Configuration

| property                 | description                    | default value                  |
| ------------------------ | ------------------------------ | ------------------------------ |
| logging.level            | log level                      | info                           |
| logging.rolling.file     | path to rolling log file       |                                |
| logging.file.max-age     | max age of log files in days   | 0 (files are retained forever) |
| logging.file.max-size    | max size of each log file      | 100                            |
| logging.file.max-backups | max number of backup log files | 0 (all files are retained)     |

## Distributed Task Scheduling Configuration

| property                | description                                                    | default value |
| ----------------------- | -------------------------------------------------------------- | ------------- |
| task.scheduling.enabled | enabled distributed task scheduling                            | true          |
| task.scheduling.group   | name of the cluster, if absent, `${app.name}` is used instead. | default       |

## Client Package Configuration

| property      | description                             | default value |
| ------------- | --------------------------------------- | ------------- |
| client.host.* | static hostname and port of the service |               |


## JWT Configuration

| property        | description                            | default value |
| --------------- | -------------------------------------- | ------------- |
| jwt.key.public  | public key for verifying the JWT token |               |
| jwt.key.private | private key for signing the JWT token  |               |
| jwt.key.issuer  | issuer of the token                    |               |


## Metrics Configuration

| property        | description                                | default value |
| --------------- | ------------------------------------------ | ------------- |
| metrics.enabled | enable metrics collection using prometheus | true          |
| metrics.route   | route used to expose collected metrics     | /metrics      |


## Yaml Configuration File Example

```yml
mode.production: true

mysql:
  enabled: false
  user: root
  password: 123456
  database: fileServer
  host: localhost
  port: 3306
```