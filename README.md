# gocommon
Common stuff for go

## Properties-Based Configuration

### Common Properties

| property | description | default value |
| --- | --- | --- | 
| profile | name of the profile used | dev |

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
| consul.registerAddress | registered service address | \${server.host}:${server.port} |  
| consul.consulAddress | address of the Consul server | localhost:8500 | 
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
| mysql.host | host | localhost |
| mysql.port | port | 3306 |

### Redis Properties

| property | description | default value |
| --- | --- | --- | 
| redis.enabled | whether Redis is enabled | false |
| redis.address | address of Redis server | localhost |
| redis.port | port of Redis server | 6379 |
| redis.username | username | |
| redis.password | password | | 
| redis.database | 0 | |  

### SQLite Properties

| property | description | default value |
| --- | --- | --- | 
| sqlite.file | path to SQLite database file | 
