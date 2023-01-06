# gocommon

Common stuff for go

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

## Examples

To bootstrap the server:

```go
// Read yml config file
common.DefaultReadConfig(os.Args)

// Add route registar
server.AddRoutesRegistar(func(engine *gin.Engine) {
    engine.GET("/some/path", func(ctx *gin.Context) {
        logrus.Info("Received request")
    })
})

// Bootstrap server, may also initialize connections to MySQL, Consul and Redis based on the loaded configuration
server.BootstrapServer()
```


## Properties-Based Configuration

### Common Properties

| property | description | default value |
| --- | --- | --- | 
| profile | name of the profile used | dev |
| mode.production | whether production mode is turned on | false |

### Web Server Properties

| property | description | default value |
| --- | --- | --- | 
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
| sqlite.file | path to SQLite database file | 


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
````