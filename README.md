# miso

Miso, yet another simple application framework. Learn by doing is great :D

Miso provides a universal configuration management mechanism and integrates with various components and libraries to make life hopefully a bit easier.

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
- Tracing (based on context.Context, it's not integrated with anything like Zipkin)
- Cron job scheduling (non-distributed)
- Distributed task scheduling (based on cron job scheduler & Redis)
- Convenient JSON processing configuration (e.g., lowercase json key naming)
- and so on.

**How a miso app may look like:**

```go
func main() {

	miso.PreServerBootstrap(func(rail miso.Rail) error {

		// prepare some event bus declaration
		if err := miso.NewEventBus(demoEventBusName); err != nil {
			return err
		}

		// register some cron scheduling job (not distributed)
		miso.ScheduleCron("0 0/15 * * * *", true, myJob)

		// register some distributed tasks
		err := miso.ScheduleDistributedTask("*/15 * * * *", false, "MyDistributedTask",
			func(miso miso.Rail) error {
				return jobDoSomething(rail)
			},
		)
		if err != nil {
			panic(err) // for demo only
		}

		// register http routes and handlers
		miso.IPost[MyReq]("/open/api/demo/post", doSomethingEndpoint).Build()

		// register grouped routes that share the same base url
		miso.BaseRoute("/open/api/demo/grouped").
			Group(
				miso.IPost[MyReq]("/post1", doSomethingEndpoint),
				miso.IPost[MyReq]("/post2", doSomethingEndpoint),
				miso.IPost[MyReq]("/post3", doSomethingEndpoint),
			)

		return nil
	})

	// bootstrap server
	miso.BootstrapServer(os.Args)
}
```

**Example of configuration file:**

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

## Documentations

- [Configuration](./doc/config.md)
- [Application Lifecycle](./doc/lifecycle.md)
- [Distributed Task Scheduling](./doc/dtask.md)
- [Validation](./doc/validate.md)
- [Service Healthcheck](./doc/health.md)
- [Customize Build](./doc/customize_build.md)

## Different Behaviour

### Default JSON Field Naming Strategy

In Golang, we export fields by capitalizing the first letter. This leads to a problem where we may have to add json tag for literally every exported fields. Miso internally uses `jsoniter`, it configures the naming strategy
that always use lowercase for the first letter of the field name unless sepcified explicitly. Whenever Miso Marshal/Unmarshal JSON values, Miso uses the configured `jsoniter` instead of the standard one. This can be reverted by registering
`PreServerBootstrap` callback to change the naming strategy back to the default one.