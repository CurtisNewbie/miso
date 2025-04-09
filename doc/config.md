# Configuration

## Command Line Arguments

By convention, without specifiying where the configuration file is, it looks for the file `conf.yml` and load the configuration properties from it. You can also specify where the config file is using builtin cli arguments: `configFile=${PATH_TO_CONFIG_FILE}`.

E.g.,

```sh
# the configFile is specified, file '/myapp/my-conf.yml' is loaded.
./main configFile=/myapp/my-conf.yml

# using default `conf.yml` file
./main
```

Properties loaded from configuration file can also be overriden by cli arguments (e.g., `KEY=VALUE`) and environment variables in `server.BootstrapServer(...)` method.

E.g.,

```sh
./main mode.production=true
```

Notice that if you have more than one configuration file to load, you can use `config.extra.files` configuration property. You can also use `${}` placeholder to borrow the value of other properties from the same configuration file, cli arguments, or even environment variables.

E.g.,

```yaml
app.name: "vfm"

mysql:
  database: "${app.name}"
```

Say that you have an environment variable `MYSQL_USERNAME=root` and `MYSQL_PASSWORD=123456`, then in your configuration file, you can refer to these values as follows:

```yaml
mysql:
  username: "${MYSQL_USERNAME}"
  password: "${MYSQL_PASSWORD}"
```

The tables shown below list all configuration that you can tune. You can also read [example_conf.yml](./example_conf.yml) to get a better understanding on how these configuration properties are mapped in a yaml file.

<!-- misoconfig-table-start -->

## Common Configuration

| property           | description                              | default value |
| ------------------ | ---------------------------------------- | ------------- |
| app.name           | name of the application                  |               |
| mode.production    | whether production mode is turned on     | true          |
| config.extra.files | extra config files that should be loaded |               |

## Consul Configuration

| property                                | description                                                                        | default value      |
| --------------------------------------- | ---------------------------------------------------------------------------------- | ------------------ |
| consul.enabled                          | enable Consul client, service registration and service discovery                   | false              |
| consul.registerName                     | registered service name                                                            | `"${app.name}"`    |
| consul.registerAddress                  | registered service address                                                         | `"${server.host}"` |
| consul.consulAddress                    | consul server address                                                              | localhost:8500     |
| consul.healthCheckUrl                   | health check url                                                                   | /health            |
| consul.healthCheckInterval              | health check interval                                                              | 5s                 |
| consul.healthCheckTimeout               | health check timeout                                                               | 3s                 |
| consul.healthCheckFailedDeregisterAfter | for how long the current instance is deregistered after first health check failure | 30m                |
| consul.registerDefaultHealthCheck       | register default health check endpoint on startup                                  | true               |
| consul.fetchServerInterval              | fetch server list from Consul in ever N seconds                                    | 30                 |
| consul.enableDeregisterUrl              | enable endpoint for manual Consul service deregistration                           | false              |
| consul.deregisterUrl                    | endpoint url for manual Consul service deregistration                              | /consul/deregister |
| consul.metadata                         | instance metadata (`map[string]string`)                                            |                    |

## Distributed Task Scheduling Configuration

| property                | description                        | default value   |
| ----------------------- | ---------------------------------- | --------------- |
| task.scheduling.enabled | enable distributed task scheduling | true            |
| task.scheduling.group   | name of the cluster                | `"${app.name}"` |

## JWT Configuration

| property        | description                            | default value |
| --------------- | -------------------------------------- | ------------- |
| jwt.key.public  | public key for verifying the JWT token |               |
| jwt.key.private | private key for signing the JWT token  |               |
| jwt.key.issuer  | issuer of the token                    |               |

## Logging Configuration

| property                  | description                                                      | default value |
| ------------------------- | ---------------------------------------------------------------- | ------------- |
| logging.level             | log level                                                        | info          |
| logging.rolling.file      | path to rolling log file                                         |               |
| logging.file.max-age      | max age of log files in days, 0 means files are retained forever | 0             |
| logging.file.max-size     | max size of each log file (in mb)                                | 50            |
| logging.file.max-backups  | max number of backup log files                                   | 10            |
| logging.file.rotate-daily | rotate log file at every day 00:00 (local)                       | true          |

## Metrics Configuration

| property                        | description                                                              | default value |
| ------------------------------- | ------------------------------------------------------------------------ | ------------- |
| metrics.enabled                 | enable metrics collection using prometheus                               | true          |
| metrics.route                   | route used to expose collected metrics                                   | /metrics      |
| metrics.auth.enabled            | enable authorization for metrics endpoint                                | false         |
| metrics.auth.bearer             | bearer token for metrics endpoint authorization                          |               |
| metrics.memstat.log.job.enabled | enable job that logs memory stats periodically (using `runtime/metrics`) | false         |
| metrics.memstat.log.job.cron    | job cron expresson for memory stats log job                              | 0 */1 * * * * |

## MySQL Configuration

| property                    | description                                                                         | default value                                                                                                     |
| --------------------------- | ----------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| mysql.enabled               | enable MySQL client                                                                 | false                                                                                                             |
| mysql.user                  | username                                                                            | root                                                                                                              |
| mysql.password              | password                                                                            |                                                                                                                   |
| mysql.database              | database                                                                            |                                                                                                                   |
| mysql.host                  | host                                                                                | localhost                                                                                                         |
| mysql.port                  | port                                                                                | 3306                                                                                                              |
| mysql.connection.parameters | connection parameters (slices of strings)                                           | `[]string{"charset=utf8mb4", "parseTime=True", "loc=Local", "readTimeout=30s", "writeTimeout=30s", "timeout=3s"}` |
| mysql.connection.lifetime   | connection lifetime in minutes (hikari recommends 1800000, so we do the same thing) | 30                                                                                                                |
| mysql.connection.open.max   | max number of open connections                                                      | 10                                                                                                                |
| mysql.connection.idle.max   | max number of idle connections                                                      | 10                                                                                                                |

## RabbitMQ Configuration

| property              | description                        | default value |
| --------------------- | ---------------------------------- | ------------- |
| rabbitmq.enabled      | enable RabbitMQ client             | false         |
| rabbitmq.host         | RabbitMQ server host               | localhost     |
| rabbitmq.port         | RabbitMQ server port               | 5672          |
| rabbitmq.username     | username used to connect to server | guest         |
| rabbitmq.password     | password used to connect to server | guest         |
| rabbitmq.vhost        | virtual host                       |               |
| rabbitmq.consumer.qos | consumer QOS                       | 68            |

## Redis Configuration

| property       | description         | default value |
| -------------- | ------------------- | ------------- |
| redis.enabled  | enable Redis client | false         |
| redis.address  | Redis server host   | localhost     |
| redis.port     | Redis server port   | 6379          |
| redis.username | username            |               |
| redis.password | password            |               |
| redis.database | database            | 0             |

## SQLite Configuration

| property           | description                  | default value |
| ------------------ | ---------------------------- | ------------- |
| sqlite.file        | path to SQLite database file |               |
| sqlite.wal.enabled | enable WAL mode              | true          |

## Service Discovery Configuration

| property                    | description                                                | default value |
| --------------------------- | ---------------------------------------------------------- | ------------- |
| service-discovery.subscribe | slice of service names that should be subcribed on startup |               |

## Tracing Configuration

| property                 | description                              | default value |
| ------------------------ | ---------------------------------------- | ------------- |
| tracing.propagation.keys | propagation keys in trace (string slice) |               |

## Web Server Configuration

| property                                                | description                                                                                                               | default value |
| ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- | ------------- |
| server.enabled                                          | enable http server                                                                                                        | true          |
| server.host                                             | http server host                                                                                                          | 127.0.0.1     |
| server.port                                             | http server port                                                                                                          | 8080          |
| server.auth.bearer                                      | http server bearer authorization token for all endpoints                                                                  |               |
| server.gracefulShutdownTimeSec                          | time wait (in second) before whole app server shutdown (previously, before `v0.1.12`, it only applies to the http server) | 30            |
| server.perf.enabled                                     | logs time duration for each inbound http request                                                                          | false         |
| server.trace.inbound.propagate                          | propagate trace info from inbound requests                                                                                | true          |
| server.validate.request.enabled                         | enable inbound request parameter validation                                                                               | true          |
| server.request-log.enabled                              | enable server request log                                                                                                 | false         |
| server.pprof.enabled                                    | enable pprof (exposed using endpoint '/debug/pprof'); in non-prod mode, it's always enabled                               | false         |
| server.pprof.auth.enabled                               | enable bearer authentication for pprof endpoints                                                                          | false         |
| server.pprof.auth.bearer                                | bearer token for pprof endpoints' authentication                                                                          |               |
| server.generate-endpoint-doc.enabled                    | generate api doc                                                                                                          | true          |
| server.generate-endpoint-doc.web.enabled                | build webpage for the generated api doc                                                                                   | true          |
| server.generate-endpoint-doc.file                       | generate markdown api doc to the specified file                                                                           |               |
| server.generate-endpoint-doc.file-excl-tclient-demo     | whether the markdown api doc should exclude miso.TClient demo                                                             | false         |
| server.generate-endpoint-doc.file-excl-ng-client-demo   | whether the markdown api doc should exclude Angular HttpClient demo                                                       | false         |
| server.generate-endpoint-doc.file-excl-openapi-spec     | whether the markdown api doc should exclude openapi json for each endpoint                                                | true          |
| server.generate-endpoint-doc.path-prefix-app            | whether the generated endpoint documentation should include app name as the path prefix                                   | true          |
| server.generate-endpoint-doc.openapi-spec.server        | server address specified in openapi json doc                                                                              |               |
| server.generate-endpoint-doc.openapi-spec.file          | path to generated openapi json for all endpoints                                                                          |               |
| server.generate-endpoint-doc.openapi-spec.path-patterns | path patterns for endpoints in openapi json (`slice of string`)                                                           |               |
| server.request.mapping.header                           | automatically map header values to request struct                                                                         | true          |
| server.gin.validation.disabled                          | disable gin's builtin validation                                                                                          | true          |

<!-- misoconfig-table-end -->

## Client Package Configuration

| property                         | description         | default value |
| -------------------------------- | ------------------- | ------------- |
| client.addr.${SERVICE_NAME}.host | client service host |               |
| client.addr.${SERVICE_NAME}.port | client service port |               |

## Yaml Configuration File Example

See [example_conf.yml](./example_conf.yml).

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
