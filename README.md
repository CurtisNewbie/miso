# gocommon

Common stuff for Go. **This is not a general library for everyone, it's developed for my personal projects :D You are very welcome to read the code tho.**

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

Properties loaded from configuration file can also be overriden by cli arguments (e.g., `KEY=VALUE`) in `config.DefaultReadConfig(...)` or `server.DefaultBootstrapServer(...)`.

e.g.,

```sh
./main mode.production=true
```

## Properties-Based Configuration

### Common Properties

| property        | description                          | default value |
|-----------------|--------------------------------------|---------------|
| app.name        | name of the application              |               |
| profile         | name of the profile used             | dev           |
| mode.production | whether production mode is turned on | false         |

### Web Server Properties

| property                       | description                                           | default value |
|--------------------------------|-------------------------------------------------------|---------------|
| server.enabled                 | enable http server                                    | true          |
| server.host                    | http server host                                      | 0.0.0.0       |
| server.port                    | http server port                                      | 8080          |
| server.gracefulShutdownTimeSec | time wait (in second) before server shutdown          | 30            |
| server.perf.enabled            | enable logging time took for each http server request | false         |

### Consul Properties

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

### MySQL Properties

| property                    | description                                                                   | default value                                                                                                   |
|-----------------------------|-------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------|
| mysql.enabled               | whether MySQL is enabled                                                      | false                                                                                                           |
| mysql.user                  | username                                                                      | root                                                                                                            |
| mysql.password              | password                                                                      |                                                                                                                 |
| mysql.database              | database                                                                      |                                                                                                                 |
| mysql.host                  | host                                                                          | `localhost`                                                                                                     |
| mysql.port                  | port                                                                          | 3306                                                                                                            |
| mysql.connection.parameters | query parameters declared on connection url (a single string joined with `&`) | `charset=utf8mb4`<br>`parseTime=True`<br>`loc=Local`<br>`readTimeout=30s`<br>`writeTimeout=30s`<br>`timeout=3s` |

### Redis Properties

| property       | description              | default value |
|----------------|--------------------------|---------------|
| redis.enabled  | whether Redis is enabled | false         |
| redis.address  | address of Redis server  | `localhost`   |
| redis.port     | port of Redis server     | 6379          |
| redis.username | username                 |               |
| redis.password | password                 |               |
| redis.database | 0                        |               |

### RabbitMQ Properties

| property              | description                        | default value |
|-----------------------|------------------------------------|---------------|
| rabbitmq.enabled      | whether RabbitMQ client is enabled | false         |
| rabbitmq.host         | host of the RabbitMQ server        | `localhost`   |
| rabbitmq.port         | port of the RabbitMQ server        | 5672          |
| rabbitmq.username     | username used to connect to server |               |
| rabbitmq.password     | password used to connect to server |               |
| rabbitmq.vhost        | virtual host                       |               |
| rabbitmq.consumer.qos | consumer QOS                       | 68            |

### SQLite Properties

| property    | description                  | default value |
|-------------|------------------------------|---------------|
| sqlite.file | path to SQLite database file |               |

### Logger Properties

| property             | description                                                                                    | default value |
|----------------------|------------------------------------------------------------------------------------------------|---------------|
| logging.rolling.file | path to rolling log file, if not set, logs are written to stdout/stderr                        |               |
| logging.level        | logging level (handled by `server.ConfigureLogging`), the configured value is case-insensitive |               |

### Distributed Task Scheduling Properties

| property                | description                                                                                                                                                                                                                   | default value |
|-------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------|
| task.scheduling.enabled | whether distributed task scheduling is enabled, this is mainly used for developement purpose, e.g., not running the tasks locally                                                                                             | true          |
| task.scheduling.group   | group name of current node. By default, it will attempt to read this property as the proposed group name. If absent, it will then read and use `app.name` property intead. If both of them are absent, then `default` is used | default       |

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

`gocommon` supports integrating with Redis, MySQL, Consul, RabbitMQ and so on. It's basically written for web application. `server.go` handles the server bootstraping, in which it helps by managing the lifecycle of the clients based on the loaded configuration.

```go
func main() {
	c := common.EmptyExecContext()

	// load configuration from 'myconf.yml'
	common.LoadConfigFromFile("myconf.yml", c)

	// add GET request handler
	server.RawGet("/some/path", func(c *gin.Context, ec common.ExecContext) {
		logrus.Info("Received request")
	})

	// bootstrap server
	server.BootstrapServer(c)
}
```

Since `gocommon` is mainly written for my personal projects, it indeed provides a very opinionated way to configure and startup the application. This follows the convention mentioned in the above sections.

```go
func main() {
	// ...

	// maybe some scheduling (not distributed)
	common.ScheduleCron("0 0/15 * * * *", myJob)

	// register routes and handlers
	server.IPost(server.OpenApiPath("/path"), myHandler)

	// bootstrap server
	server.DefaultBootstrapServer(os.Args, common.EmptyExecContext())
}
```

### validation.go

`validation.go` is used for validating parameters against some pre-defined rules. This is enabled by adding tag "validation" on the fields.

For example,

```go
type Dummy struct {
  Favourite string `validation:"notEmpty"`
}
```

To validate a struct, just call `common.Validate(...)` as follows:

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
	task.SetScheduleGroup("gocommon")

	// add task
	task.ScheduleDistributedTask("0/1 * * * * ?", func(c common.ExecContext) {
		// ...
	})

	// start task scheduler
	task.StartTaskSchedulerAsync()

	// stop task scheduler gracefully
	defer task.StopTaskScheduler()
}
```

If `server.go` is used, this is automatically handled by `DefaultBootstrapServer(...)` func.

```go
func main() {
	// add tasks
	task.ScheduleDistributedTask("0 0/15 * * * *", func(c common.ExecContext) {
	})

	// bootstrap server
	server.DefaultBootstrapServer(os.Args, common.EmptyExecContext())
}
```
