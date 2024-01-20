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

The tables shown below list all configuration that you can tune. You can also read [example_conf.yml](./example_conf.yml) to get a better understanding on how these configuration properties are mapped in a yaml file.

## Common Configuration

| property           | description                              | default value |
| ------------------ | ---------------------------------------- | ------------- |
| app.name           | name of the application                  |               |
| mode.production    | whether production mode is turned on     | false         |
| config.extra.files | extra config files that should be loaded |               |

## Web Server Configuration

| property                        | description                                          | default value |
| ------------------------------- | ---------------------------------------------------- | ------------- |
| server.enabled                  | enable http server                                   | true          |
| server.host                     | http server host                                     | 0.0.0.0       |
| server.port                     | http server port                                     | 8080          |
| server.gracefulShutdownTimeSec  | time wait (in second) before http server shutdown    | 30            |
| server.perf.enabled             | logs time duration for each inbound http request     | false         |
| server.trace.inbound.propagate  | propagate trace info from inbound requests           | true          |
| server.validate.request.enabled | enable inbound request parameter validation          | true          |
| server.request-log.enabled      | enable server request log                            | false         |
| server.pprof.enabled            | enable pprof (exposed using endpoint '/debug/pprof') | false         |

## Consul Configuration

| property                                | description                                                                        | default value                   |
| --------------------------------------- | ---------------------------------------------------------------------------------- | ------------------------------- |
| consul.enabled                          | enable Consul client, service registration and service discovery                   | false                           |
| consul.registerName                     | registered service name                                                            | `${app.name}`                   |
| consul.registerAddress                  | registered service address                                                         | `${server.host}:${server.port}` |
| consul.consulAddress                    | consul server address                                                              | `localhost:8500`                |
| consul.healthCheckUrl                   | health check url                                                                   | `/health`                       |
| consul.healthCheckInterval              | health check interval                                                              | 5s                              |
| consul.healthCheckTimeout               | health check timeout                                                               | 3s                              |
| consul.healthCheckFailedDeregisterAfter | for how long the current instance is deregistered after first health check failure | 30m                             |
| consul.registerDefaultHealthCheck       | register default health check endpoint on startup                                  | true                            |
| consul.fetchServerInterval              | fetch server list from Consul in ever N seconds                                    | 30                              |
| consul.enableDeregisterUrl              | enable endpoint for manual Consul service deregistration                           | false                           |
| consul.deregisterUrl                    | endpoint url for manual Consul service deregistration                              | `/consul/deregister`            |
| consul.metadata                         | instance metadata (`map[string]string`)                                            |                                 |


## MySQL Configuration

| property                    | description                               | default value                                                                                                   |
| --------------------------- | ----------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| mysql.enabled               | enable MySQL client                       | false                                                                                                           |
| mysql.user                  | username                                  | root                                                                                                            |
| mysql.password              | password                                  |                                                                                                                 |
| mysql.database              | database                                  |                                                                                                                 |
| mysql.host                  | host                                      | `localhost`                                                                                                     |
| mysql.port                  | port                                      | 3306                                                                                                            |
| mysql.connection.parameters | connection parameters (slices of strings) | "charset=utf8mb4"<br>"parseTime=True"<br>"loc=Local"<br>"readTimeout=30s"<br>"writeTimeout=30s"<br>"timeout=3s" |
| mysql.connection.lifetime   | connection lifetime in minutes            | 30                                                                                                              |
| mysql.connection.open.max   | max number of open connections            | 10                                                                                                              |
| mysql.connection.idle.max   | max number of idle connections            | 10                                                                                                              |



## Redis Configuration

| property       | description         | default value |
| -------------- | ------------------- | ------------- |
| redis.enabled  | enable Redis client | false         |
| redis.address  | Redis server host   | `localhost`   |
| redis.port     | Redis server port   | 6379          |
| redis.username | username            |               |
| redis.password | password            |               |
| redis.database | 0                   |               |

## RabbitMQ Configuration

| property              | description                        | default value |
| --------------------- | ---------------------------------- | ------------- |
| rabbitmq.enabled      | enable RabbitMQ client             | false         |
| rabbitmq.host         | RabbitMQ server host               | `localhost`   |
| rabbitmq.port         | RabbitMQ server port               | 5672          |
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

| property                  | description                                | default value                  |
| ------------------------- | ------------------------------------------ | ------------------------------ |
| logging.level             | log level                                  | info                           |
| logging.rolling.file      | path to rolling log file                   |                                |
| logging.file.max-age      | max age of log files in days               | 0 (files are retained forever) |
| logging.file.max-size     | max size of each log file                  | 100                            |
| logging.file.max-backups  | max number of backup log files             | 0 (all files are retained)     |
| logging.file.rotate-daily | rotate log file at every day 00:00 (local) | true                           |

## Distributed Task Scheduling Configuration

| property                | description                                                    | default value |
| ----------------------- | -------------------------------------------------------------- | ------------- |
| task.scheduling.enabled | enable distributed task scheduling                             | true          |
| task.scheduling.group   | name of the cluster, if absent, `${app.name}` is used instead. |               |

## Client Package Configuration

| property                         | description         | default value |
| -------------------------------- | ------------------- | ------------- |
| client.addr.${SERVICE_NAME}.host | client service host |               |
| client.addr.${SERVICE_NAME}.port | client service port |               |


## JWT Configuration

| property        | description                            | default value |
| --------------- | -------------------------------------- | ------------- |
| jwt.key.public  | public key for verifying the JWT token |               |
| jwt.key.private | private key for signing the JWT token  |               |
| jwt.key.issuer  | issuer of the token                    |               |


## Metrics Configuration

| property                        | description                                                              | default value   |
| ------------------------------- | ------------------------------------------------------------------------ | --------------- |
| metrics.enabled                 | enable metrics collection using prometheus                               | true            |
| metrics.route                   | route used to expose collected metrics                                   | /metrics        |
| metrics.auth.enabled            | enable authorization for metrics endpoint                                | false           |
| metrics.auth.bearer             | bearer token for metrics endpoint authorization                          |                 |
| metrics.memstat.log.job.enabled | enable job that logs memory stats periodically (using `runtime/metrics`) | false           |
| metrics.memstat.log.job.cron    | job cron expresson for memory stats log job                              | `0 */1 * * * *` |


## Yaml Configuration File Example

```yml
app.name: "myapp"

mode.production: true

mysql:
  enabled: true
  user: "root"
  password: "123456"
  database: "mydb"
  host: "localhost"
  port: 3306
  connection:
    parameters:
      - "charset=utf8mb4"
      - "parseTime=True"
      - "loc=Local"
      - "readTimeout=30s"
      - "writeTimeout=30s"
      - "timeout=3s"

server:
  host: localhost
  port: 8080
```