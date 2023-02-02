# gocommon v1.0.3

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

## Properties-Based Configuration

### Common Properties

| property | description | default value |
| --- | --- | --- | 
| app.name | name of the application, if `consul.registerName` is missing, this will be used for the service registration | |
| profile | name of the profile used | dev |
| mode.production | whether production mode is turned on | false |

### Web Server Properties

| property | description | default value |
| --- | --- | --- | 
| server.web.enabled | enable http server | true |   
| server.host | http server host | localhost |   
| server.port | http server port | 8080 |
| server.gracefulShutdownTimeSec | time wait (in second) before server shutdown | 5 | 

### Consul Properties

| property | description | default value |
| --- | --- | --- | 
| consul.enabled | whether Consul is enabled | false |
| consul.registerName | registered service name | | 
| consul.registerAddress | registered service address | `${server.host}:${server.port}` |  
| consul.consulAddress | address of the Consul server | `localhost:8500` | 
| consul.healthCheckUrl | health check url | /health |
| consul.healthCheckInterval | health check interval | 60s |
| consul.healthCheckTimeout | health check timeout | 3s |
| consul.healthCheckFailedDeregisterAfter | timeout for current service to deregister after health check failure | 130s |

### MySQL Properties

| property | description | default value |
| --- | --- | --- | 
| mysql.enabled | whether MySQL is enabled | false |
| mysql.user | username  | root |
| mysql.password | password |  |
| mysql.database | database | |  
| mysql.host | host | `localhost` |
| mysql.port | port | 3306 |
| mysql.connection.parameters | query parameters declared on connection url | `charset=utf8mb4&parseTime=True&loc=Local&readTimeout=30s&writeTimeout=30s&timeout=3s` |

### Redis Properties

| property | description | default value |
| --- | --- | --- | 
| redis.enabled | whether Redis is enabled | false |
| redis.address | address of Redis server | `localhost` |
| redis.port | port of Redis server | 6379 |
| redis.username | username | |
| redis.password | password | | 
| redis.database | 0 | |  

### RabbitMQ Properties

| property | description | default value |
| --- | --- | --- | 
| rabbitmq.enabled | whether RabbitMQ client is enabled | false | 
| rabbitmq.host | host of the RabbitMQ server | `localhost` | 
| rabbitmq.port | port of the RabbitMQ server | 5672 | 
| rabbitmq.username | username used to connect to server | | 
| rabbitmq.password | password used to connect to server | | 
| rabbitmq.vhost | virtual host | | 
| rabbitmq.consumer.qos | consumer QOS | 68 | 
| rabbitmq.consumer.parallism | consumer parallism (number of goroutines for each queue) | 2 | 
| rabbitmq.consumer.retry | maximum number of retry for message received by consumer: `-1` means never retry, and the message will simply be Nack(ed); if retry is set to greater than -1, then whenever all the retry is used, the message is acked to prevent infinite redelivery | -1 | 

### SQLite Properties

| property | description | default value |
| --- | --- | --- | 
| sqlite.file | path to SQLite database file |  | 

### Logger Properties

| property | description | default value |
| --- | --- | --- | 
| logging.rolling.file | path to rolling log file, if not set, logs are written to stdout/stderr |  | 


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
  // load configuration from 'myconf.yml'
  LoadConfigFromFile("myconf.yml")

  // add GET request handler 
  server.RawGet("/some/path", func(ctx *gin.Context) {
    logrus.Info("Received request")
  })

  // bootstrap server
  server.BootstrapServer()
}
```

Since `gocommon` is mainly written for my personal projects, it indeed provides a very opinionated way to configure and startup the application. This follows the convention mentioned in the above sections.

```go
func main() {
  // maybe some scheduling (not distributed)
  common.ScheduleCron("0 0/15 * * * *", myJob)

  // register routes and handlers
  server.PostJ(server.OpenApiPath("/path"), myHandler)

  // default way to determine profile used, find config file, load configuration, and bootstrap server
  server.DefaultBootstrapServer(os.Args)
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