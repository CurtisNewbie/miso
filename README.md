# miso

Miso, yet another simple application framework (formerly named gocommon).

*This is not really a general library for everyone, it's mainly developed for my personal projects :D. You are very welcome to read the code tho.*

Miso provides a universal configuration loading mechanism (wrapping Viper) and integrates with various components and libraries to make life hopefully a bit easier:

List of integration and functionalities provided:

- MySQL
- Consul
- Redis
- SQLite
- RabbitMQ
- JWT Encoding / Decoding
- Gin
- Http Client
- Logrus & Lumberjack (for rotating log files)
- Prometheus
- Tracing (based on context.Context, it's not integrated with anything like Zipkin, only the logs)
- Cron job scheduling (non-distributed)
- Distributed task scheduling (based on cron job scheduler)

**How a miso app may look like:**

```go
func main() {

	server.PreServerBootstrap(func(rail core.Rail) error {

		// prepare some event bus declaration
		if err := bus.DeclareEventBus(demoEventBusName); err != nil {
			return err
		}

		// register some cron scheduling job (not distributed)
		miso.ScheduleCron("0 0/15 * * * *", true, myJob)

		// register some distributed tasks
		err := task.ScheduleNamedDistributedTask("*/15 * * * *", false, "MyDistributedTask",
			func(rail core.Rail) error {
				return doSomething(rail)
			}
		)
		if err != nil {
			panic(err) // for demo only
		}

		// register http routes and handlers
		server.IPost("/open/api/demo", func(c *gin.Context, rail core.Rail, req DoSomethingReq) (any, error) {
				rail.Infof("Received request, %+v", req)
				return doSomething(rail, req)
			})
		})

	// bootstrap server
	server.BootstrapServer(os.Args)
}
```

## Initialize Project

Convenient way to initialize a new project:

```
mkdir myapp \
	&& cd myapp \
	&& curl https://raw.githubusercontent.com/CurtisNewbie/miso/main/init.sh \
	| bash
```

## Command Line Arguments

- To specify profile: `profile=${PROFILE_NAME}`
- To specify where the config file is: `configFile=${PATH_TO_CONFIG_FILE}`

By convention, without specifiying where the configuration file is, it looks for the file `app-conf-${PROFILE_NAME}.yml` and load the configuration properties from it.

e.g.,

```sh
# both profile and configFile are specified
./main profile='prod' configFile=/myapp/my-conf.yml

# only profile is specified, the configFile will be 'app-conf-prod.yml'
./main profile='prod'

# using default profile 'dev', the configFile will be 'app-conf-dev.yml'
./main
```

Properties loaded from configuration file can also be overriden by cli arguments (e.g., `KEY=VALUE`) and environment variables in `server.BootstrapServer(...)` method.

e.g.,

```sh
./main mode.production=true
```

## Configuration

### Common Configuration

| property        | description                          | default value |
|-----------------|--------------------------------------|---------------|
| app.name        | name of the application              |               |
| profile         | name of the profile used             | dev           |
| mode.production | whether production mode is turned on | false         |

### Web Server Configuration

| property                       | description                                           | default value |
|--------------------------------|-------------------------------------------------------|---------------|
| server.enabled                 | enable http server                                    | true          |
| server.host                    | http server host                                      | 0.0.0.0       |
| server.port                    | http server port                                      | 8080          |
| server.gracefulShutdownTimeSec | time wait (in second) before server shutdown          | 30            |
| server.perf.enabled            | enable logging time took for each http server request | false         |
| server.trace.inbound.propagate | propagate trace info from inbound requests            | true          |

### Consul Configuration

| property                                | description                                                          | default value                   |
|-----------------------------------------|----------------------------------------------------------------------|---------------------------------|
| consul.enabled                          | whether Consul is enabled                                            | false                           |
| consul.registerName                     | registered service name                                              | `${app.name}`                   |
| consul.registerAddress                  | registered service address                                           | `${server.host}:${server.port}` |
| consul.consulAddress                    | address of the Consul server                                         | `localhost:8500`                |
| consul.healthCheckUrl                   | health check url                                                     | `/health`                       |
| consul.healthCheckInterval              | health check interval                                                | 15s                             |
| consul.healthCheckTimeout               | health check timeout                                                 | 3s                              |
| consul.healthCheckFailedDeregisterAfter | timeout for current service to deregister after health check failure | 120s                            |
| consul.registerDefaultHealthCheck       | register default health check endpoint on startup                    | true                            |

### MySQL Configuration

| property                    | description                                 | default value                                                                          |
|-----------------------------|---------------------------------------------|----------------------------------------------------------------------------------------|
| mysql.enabled               | whether MySQL is enabled                    | false                                                                                  |
| mysql.user                  | username                                    | root                                                                                   |
| mysql.password              | password                                    |                                                                                        |
| mysql.database              | database                                    |                                                                                        |
| mysql.host                  | host                                        | `localhost`                                                                            |
| mysql.port                  | port                                        | 3306                                                                                   |
| mysql.connection.parameters | query parameters declared on connection url | `charset=utf8mb4&parseTime=True&loc=Local&readTimeout=30s&writeTimeout=30s&timeout=3s` |

### Redis Configuration

| property       | description              | default value |
|----------------|--------------------------|---------------|
| redis.enabled  | whether Redis is enabled | false         |
| redis.address  | address of Redis server  | `localhost`   |
| redis.port     | port of Redis server     | 6379          |
| redis.username | username                 |               |
| redis.password | password                 |               |
| redis.database | 0                        |               |

### RabbitMQ Configuration

| property              | description                        | default value |
|-----------------------|------------------------------------|---------------|
| rabbitmq.enabled      | whether RabbitMQ client is enabled | false         |
| rabbitmq.host         | host of the RabbitMQ server        | `localhost`   |
| rabbitmq.port         | port of the RabbitMQ server        | 5672          |
| rabbitmq.username     | username used to connect to server |               |
| rabbitmq.password     | password used to connect to server |               |
| rabbitmq.vhost        | virtual host                       |               |
| rabbitmq.consumer.qos | consumer QOS                       | 68            |

Miso's integration with RabbitMQ supports delayed message redelivery (messages that can't be handled without error), the delay is currently 10 seconds. This is to prevent server being flooded with redelivered messages, this is not configurable though.

### SQLite Configuration

| property    | description                  | default value |
|-------------|------------------------------|---------------|
| sqlite.file | path to SQLite database file |               |

### Logging Configuration

| property             | description              | default value |
|----------------------|--------------------------|---------------|
| logging.rolling.file | path to rolling log file |               |
| logging.level        | log level                | info          |

### Distributed Task Scheduling Configuration

| property                | description                                                    | default value |
|-------------------------|----------------------------------------------------------------|---------------|
| task.scheduling.enabled | enabled distributed task scheduling                            | true          |
| task.scheduling.group   | name of the cluster, if absent, `${app.name}` is used instead. | default       |

### Client Package Configuration

| property      | description                             | default value |
|---------------|-----------------------------------------|---------------|
| client.host.* | static hostname and port of the service |               |


### JWT Configuration

| property        | description                            | default value |
|-----------------|----------------------------------------|---------------|
| jwt.key.public  | public key for verifying the JWT token |               |
| jwt.key.private | private key for signing the JWT token  |               |
| jwt.key.issuer  | issuer of the token                    |               |


### Metrics Configuration

| property        | description                                | default value |
|-----------------|--------------------------------------------|---------------|
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

## More about the code

### server.go

Miso supports integrating with Redis, MySQL, Consul, RabbitMQ and so on. It's basically written for web application. `server.go` handles the server bootstraping, in which it helps by managing the lifecycle of the clients based on the loaded configuration.

Since miso is mainly written for my personal projects, it indeed provides a very opinionated way to configure and startup the application. This follows the convention mentioned in the above sections.

```go
func main() {
	// ...

	// maybe some scheduling (not distributed)
	miso.ScheduleCron("0 0/15 * * * *", true, myJob)

	// register routes and handlers
	server.IPost("/my/path", myHandler)

	// bootstrap server
	server.BootstrapServer(os.Args)
}
```

### validation.go

`validation.go` is used for validating parameters against some pre-defined rules. This is enabled by adding tag "validation" on the fields.

For example,

```go
type Dummy struct {
  Favourite string `valid:"notEmpty"`
}
```

To validate a struct, just call `core.Validate(...)` as follows:

```go
func TestValidate(t *testing.T) {
  v := Dummy{}
  e := Validate(v)
  if e != nil {
    t.Fatal(e)
  }
}
```

The rules available are (see constants and documentation in `validation.go`):

- maxLen
- notEmpty
- notNil
- positive
- positiveOrZero
- negative
- negativeOrZero
- notZero
- validated

A field can have more than one rule, these rules are sapareted using ',', and the rules are validated in the order in which they are declared, for example:

```go
type ValidatedDummy struct {
  DummyPtr *AnotherDummy `validation:"notNil,validated"`
}
```

The `DummyPtr` field is then validated against rule `notNil` first, and then the rule `validated`.

Some rules require parameters (only `maxLen` for now), these are specified in the format: `[RULE_NAME]:[PARAM]`, for example:

```go
type ValidatedDummy struct {
  Name string `validation:"maxLen:10,notEmpty"`
}
```

It's required that the `Name` field can at most have 10 characters, and it cannot be empty (blank).

Rule `validated` is very special. It doesn't actually check the value of the field, instead, it annotates that the field should be further analyzed recursively. If the field is a pointer and it's not nil, the actual value referred is validated. Else, if the field is just a simple struct, then the struct is scanned.

### task.go

`task.go` internally wraps `schedule.go` to support distributed task scheduling. A cluster is distinguished by a group name, each cluster of nodes can only have one master, and the master node is reponsible for running all the tasks.

```go
func main() {
	// set the group name
	task.SetScheduleGroup("myApp")

	// add task
	task.ScheduleDistributedTask("0/1 * * * * ?", true, func(c core.Rail) {
		// ...
	})

	// start task scheduler
	task.StartTaskSchedulerAsync()

	// stop task scheduler gracefully
	defer task.StopTaskScheduler()
}
```

If `server.go` is used, this is automatically handled by `BootstrapServer(...)` func.

```go
func main() {
	// add tasks
	task.ScheduleDistributedTask("0 0/15 * * * *", true, func(c core.Rail) {
	})

	// bootstrap server
	server.BootstrapServer(os.Args)
}
```

#### More

A lot more stuff is written but not documented here, it may not be useful for you, but feel free to read the code :D.