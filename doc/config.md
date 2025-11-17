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

The `${}` expression also support default value. For example, if `my-schema` property is missing, `my_db` is provided as the default value.

```yaml
mysql:
  database: "${my-schema:my_db}"
```

You can even overwrite configurations without using the `${}` syntax. Export environment variables that starts with `'MISO_'` and use '_' as the delimiter, e.g., `'MISO_MYSQL_DATABASE=xxx'`, which will then be converted to `'mysql.database=xxx'` and loaded into the miso app.

The tables shown below list all configuration that you can tune. You can also read [example_conf.yml](./example_conf.yml) to get a better understanding on how these configuration properties are mapped in a yaml file.

<!-- misoconfig-table-start -->

## Common Configuration

| property                     | description                                                                 | default value |
| ---------------------------- | --------------------------------------------------------------------------- | ------------- |
| app.name                     | name of the application                                                     |               |
| app.profile                  | profile name, it's only a flag used to identify which environment we are in |               |
| app.slow-bootstrap-threshold | warning threshold for slow ComponentBootstrap                               | 5s            |
| mode.production              | whether production mode is turned on                                        | true          |
| config.extra.files           | extra config files that should be loaded                                    |               |

## Consul Configuration

| property                                   | description                                                                        | default value      |
| ------------------------------------------ | ---------------------------------------------------------------------------------- | ------------------ |
| consul.enabled                             | enable Consul client, service registration and service discovery                   | false              |
| consul.register-name                       | registered service name                                                            | `"${app.name}"`    |
| consul.register-address                    | registered service address                                                         | `"${server.host}"` |
| consul.consul-address                      | consul server address                                                              | localhost:8500     |
| consul.health-check-failed-deregister-time | for how long the current instance is deregistered after first health check failure | 30m                |
| consul.fetch-server-interval               | fetch server list from Consul in ever N seconds                                    | 30                 |
| consul.enable-deregister-url               | enable endpoint for manual Consul service deregistration                           | false              |
| consul.deregister-url                      | endpoint url for manual Consul service deregistration                              | /consul/deregister |
| consul.metadata                            | instance metadata (`map[string]string`)                                            |                    |

## Distributed Task Scheduling Configuration

| property                             | description                        | default value   |
| ------------------------------------ | ---------------------------------- | --------------- |
| task.scheduling.enabled              | enable distributed task scheduling | true            |
| task.scheduling.group                | name of the cluster                | `"${app.name}"` |
| task.scheduling.${taskName}.disabled | disable specific task by it's name | false           |

## JWT Configuration

| property        | description                            | default value |
| --------------- | -------------------------------------- | ------------- |
| jwt.key.public  | public key for verifying the JWT token |               |
| jwt.key.private | private key for signing the JWT token  |               |
| jwt.key.issuer  | issuer of the token                    |               |

## Job Scheduler Configuration

| property                          | description                                                     | default value |
| --------------------------------- | --------------------------------------------------------------- | ------------- |
| scheduler.api.trigger-job.enabled | enable API to manually trigger jobs (and tasks on current node) | false         |

## Kafka Configuration

| property          | description                    | default value  |
| ----------------- | ------------------------------ | -------------- |
| kafka.enabled     | Enable kafka client            | false          |
| kafka.server.addr | list of kafka server addresses | localhost:9092 |

## Logging Configuration

| property                      | description                                                      | default value |
| ----------------------------- | ---------------------------------------------------------------- | ------------- |
| logging.level                 | log level                                                        | info          |
| logging.rolling.file          | path to rolling log file                                         |               |
| logging.file.append-ip-suffix | append ip suffix to log file, e.g., myapp-192.168.1.1.log        | false         |
| logging.file.log-file-only    | logs are written to log file only                                | false         |
| logging.file.max-age          | max age of log files in days, 0 means files are retained forever | 0             |
| logging.file.max-size         | max size of each log file (in mb)                                | 50            |
| logging.file.max-backups      | max number of backup log files, 0 means INF                      | 0             |
| logging.file.rotate-daily     | rotate log file at every day 00:00 (local)                       | true          |

## Metrics Configuration

| property                        | description                                                                      | default value  |
| ------------------------------- | -------------------------------------------------------------------------------- | -------------- |
| metrics.enabled                 | enable metrics collection using prometheus                                       | true           |
| metrics.route                   | route used to expose collected metrics                                           | /metrics       |
| metrics.auth.enabled            | enable authorization for metrics endpoint                                        | false          |
| metrics.auth.bearer             | bearer token for metrics endpoint authorization                                  |                |
| metrics.memstat.log.job.enabled | enable job that logs memory and cpu stats periodically (using `runtime/metrics`) | false          |
| metrics.memstat.log.job.cron    | job cron expresson for memory stats log job                                      | 0/30 * * * * * |

## MySQL Configuration

| property                                | description                                                                                                                                           | default value                                                                                                                                                                |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| mysql.enabled                           | enable MySQL client                                                                                                                                   | false                                                                                                                                                                        |
| mysql.user                              | username                                                                                                                                              | root                                                                                                                                                                         |
| mysql.password                          | password                                                                                                                                              |                                                                                                                                                                              |
| mysql.database                          | database                                                                                                                                              |                                                                                                                                                                              |
| mysql.host                              | host                                                                                                                                                  | localhost                                                                                                                                                                    |
| mysql.port                              | port                                                                                                                                                  | 3306                                                                                                                                                                         |
| mysql.log-sql                           | log sql statements                                                                                                                                    | false                                                                                                                                                                        |
| mysql.prepare-statement                 | enable prepared statement                                                                                                                             | true                                                                                                                                                                         |
| mysql.disable-nested-transaction        | disabled nested transaction                                                                                                                           | true                                                                                                                                                                         |
| mysql.connection.parameters             | connection parameters (slices of strings) (see [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql?tab=readme-ov-file#dsn-data-source-name)) | `[]string{"charset=utf8mb4", "parseTime=true", "loc=Local", "readTimeout=30s", "writeTimeout=30s", "timeout=3s", "collation=utf8mb4_general_ci", "interpolateParams=false"}` |
| mysql.connection.lifetime               | connection lifetime in minutes (hikari recommends 1800000, so we do the same thing)                                                                   | 30                                                                                                                                                                           |
| mysql.connection.open.max               | max number of open connections                                                                                                                        | 10                                                                                                                                                                           |
| mysql.connection.idle.max               | max number of idle connections                                                                                                                        | 10                                                                                                                                                                           |
| mysql.managed.${name}.user              | managed connection username                                                                                                                           | root                                                                                                                                                                         |
| mysql.managed.${name}.password          | managed connection password                                                                                                                           |                                                                                                                                                                              |
| mysql.managed.${name}.database          | managed connection database                                                                                                                           |                                                                                                                                                                              |
| mysql.managed.${name}.host              | managed connection host                                                                                                                               | localhost                                                                                                                                                                    |
| mysql.managed.${name}.port              | managed connection port                                                                                                                               | 3306                                                                                                                                                                         |
| mysql.managed.${name}.prepare-statement | managed connection enable prepared statement                                                                                                          | true                                                                                                                                                                         |

## Nacos Configuration

| property                              | description                                                                               | default value      |
| ------------------------------------- | ----------------------------------------------------------------------------------------- | ------------------ |
| nacos.enabled                         | enable nacos client                                                                       | false              |
| nacos.server.addr                     | nacos server address                                                                      | localhost          |
| nacos.server.scheme                   | nacos server address scheme                                                               | http               |
| nacos.server.port                     | nacos server port (by default it's either 80, 443 or 8848)                                |                    |
| nacos.server.context-path             | nacos server context path                                                                 |                    |
| nacos.server.namespace                | nacos server namespace                                                                    |                    |
| nacos.server.username                 | nacos server username                                                                     |                    |
| nacos.server.password                 | nacos server password                                                                     |                    |
| nacos.server.config.data-id           | nacos config data-id                                                                      | ${app.name}        |
| nacos.server.config.group             | nacos config group                                                                        | DEFAULT_GROUP      |
| nacos.server.config.watch             | extra watched nacos config, (slice of strings, format: `"${data-id}" + ":" + "${group}"`) |                    |
| nacos.discovery.enabled               | enable nacos client for service discovery                                                 | true               |
| nacos.discovery.register-instance     | register current instance on nacos for service discovery                                  | true               |
| nacos.discovery.register-address      | register service address                                                                  | `"${server.host}"` |
| nacos.discovery.register-name         | register service name                                                                     | `"${app.name}"`    |
| nacos.discovery.enable-deregister-url | enable endpoint for manual Nacos service deregistration                                   | false              |
| nacos.discovery.deregister-url        | endpoint url for manual Nacos service deregistration                                      | /nacos/deregister  |
| nacos.discovery.metadata              | instance metadata (`map[string]string`)                                                   |                    |
| nacos.cache-dir                       | nacos cache dir                                                                           | /tmp/nacos/cache   |

## RabbitMQ Configuration

| property                             | description                        | default value |
| ------------------------------------ | ---------------------------------- | ------------- |
| rabbitmq.enabled                     | enable RabbitMQ client             | false         |
| rabbitmq.host                        | RabbitMQ server host               | localhost     |
| rabbitmq.port                        | RabbitMQ server port               | 5672          |
| rabbitmq.username                    | username used to connect to server | guest         |
| rabbitmq.password                    | password used to connect to server | guest         |
| rabbitmq.vhost                       | virtual host                       |               |
| rabbitmq.consumer.qos                | consumer QOS                       | 68            |
| rabbitmq.publisher.channel-pool-size | publisher channel pool size        | 20            |

## Redis Configuration

| property            | description                                                                                                                                                            | default value |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| redis.enabled       | enable Redis client                                                                                                                                                    | false         |
| redis.address       | Redis server host                                                                                                                                                      | localhost     |
| redis.port          | Redis server port                                                                                                                                                      | 6379          |
| redis.username      | username                                                                                                                                                               |               |
| redis.password      | password                                                                                                                                                               |               |
| redis.database      | database                                                                                                                                                               | 0             |
| redis.max-pool-size | max connection pool size (Default is 10 connections per every available CPU as reported by runtime.GOMAXPROCS or 64 connections if the calculated one is less then 64) | 0             |

## SQLite Configuration

| property           | description                  | default value |
| ------------------ | ---------------------------- | ------------- |
| sqlite.file        | path to SQLite database file |               |
| sqlite.wal.enabled | enable WAL mode              | true          |
| sqlite.log-sql     | log sql statements           | false         |

## Service Discovery Configuration

| property                    | description                                                | default value |
| --------------------------- | ---------------------------------------------------------- | ------------- |
| service-discovery.subscribe | slice of service names that should be subcribed on startup |               |

## Tracing Configuration

| property                 | description                              | default value |
| ------------------------ | ---------------------------------------- | ------------- |
| tracing.propagation.keys | propagation keys in trace (string slice) |               |

## Web Server Configuration

| property                                  | description                                                                                                                                                                              | default value |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| server.enabled                            | enable http server                                                                                                                                                                       | true          |
| server.host                               | http server host                                                                                                                                                                         | 127.0.0.1     |
| server.port                               | http server port                                                                                                                                                                         | 8080          |
| server.health-check-url                   | health check url                                                                                                                                                                         | /health       |
| server.health-check-interval              | health check interval, it's only used for service discovery, e.g., Consul                                                                                                                | 5s            |
| server.health-check-timeout               | health check timeout, it's only used for service discovery, e.g., Consul                                                                                                                 | 3s            |
| server.log-routes                         | log all http server routes in INFO level                                                                                                                                                 | true          |
| server.auth.bearer                        | http server bearer authorization token for all endpoints                                                                                                                                 |               |
| server.graceful-shutdown-time-sec         | time wait (in second) before whole app server shutdown (previously, before `v0.1.12`, it only applies to the http server)                                                                | 30            |
| server.perf.enabled                       | logs time duration for each inbound http request                                                                                                                                         | false         |
| server.trace.inbound.propagate            | propagate trace info from inbound requests                                                                                                                                               | true          |
| server.validate.request.enabled           | enable inbound request parameter validation                                                                                                                                              | true          |
| server.request-log.enabled                | enable server request log                                                                                                                                                                | true          |
| server.pprof.enabled                      | enable apis for pprof (`/debug/pprof/**`) and flight recorder (`/debug/trace/**`), see [FlightRecorder Blog](https://go.dev/blog/flight-recorder); in non-prod mode, it's always enabled | false         |
| server.pprof.auth.bearer                  | bearer token for pprof and trace api authentication. If `server.auth.bearer` is set for all api, this prop is ignored.                                                                   |               |
| server.api-doc.enabled                    | generate api doc                                                                                                                                                                         | true          |
| server.api-doc.web.enabled                | build webpage for the generated api doc                                                                                                                                                  | true          |
| server.api-doc.file                       | generate markdown api doc to the specified file                                                                                                                                          |               |
| server.api-doc.file-excl-tclient-demo     | the generated markdown api doc should exclude miso.TClient demo                                                                                                                          | false         |
| server.api-doc.file-excl-ngclient-demo    | the generated markdown api doc should exclude Angular HttpClient demo                                                                                                                    | false         |
| server.api-doc.file-excl-openapi-spec     | the generated markdown api doc should exclude openapi json for each endpoint                                                                                                             | true          |
| server.api-doc.path-prefix-app            | the generated endpoint documentation should include app name as the path prefix                                                                                                          | true          |
| server.api-doc.openapi-spec.server        | server address specified in openapi json doc                                                                                                                                             |               |
| server.api-doc.openapi-spec.file          | path to generated openapi json for all endpoints                                                                                                                                         |               |
| server.api-doc.openapi-spec.path-patterns | path patterns for endpoints in openapi json (`slice of string`)                                                                                                                          |               |
| server.api-doc.go.file                    | file that contains the generated api doc golang demo                                                                                                                                     |               |
| server.api-doc.go.compile-file            | whether the generated api-doc golang demo file should compile                                                                                                                            | false         |
| server.api-doc.go.path-patterns           | path patterns for endpoints that are written to api doc golang demo file                                                                                                                 |               |
| server.api-doc.go.excl-path-patterns      | path patterns excluding for endpoints that should not be written to api doc golang demo file                                                                                             |               |
| server.request.mapping.header             | automatically map header values to request struct                                                                                                                                        | true          |
| server.gin.validation.disabled            | disable gin's builtin validation                                                                                                                                                         | true          |

## Zookeeper Configuration

| property           | description                         | default value |
| ------------------ | ----------------------------------- | ------------- |
| zk.enabled         | enable zk client                    | false         |
| zk.hosts           | zk server host (slice of string)    | localhost     |
| zk.session-timeout | zk server session timeout (seconds) | 5             |

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
