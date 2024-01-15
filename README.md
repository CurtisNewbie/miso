# miso

Miso, yet another simple application framework. It's mainly a <i>learn-by-doing</i> project for me to understand how things work :D

Miso provides an opinionated way to write application, common functionalities such as configuration, service discovery, load balancing, log tracing, log rotation, task scheduling, message queue and so on, are all implemented in an opinionated way. You can use miso to write almost any kind of application, feel free to read the code.

The overall target is to make it as small and simple as possible. The backward compatibility may break in future releases.

**How a miso app may look like:**

```go
func main() {

	miso.PreServerBootstrap(func(rail miso.Rail) error {

		// prepare some event bus declaration
		if err := miso.NewEventBus(demoEventBusName); err != nil {
			return err
		}

		// register some cron scheduling job (not distributed)
		err := miso.ScheduleCron(miso.Job{
			Name:            "MyJob",
			Cron:            "0 0/15 * * * *",
			CronWithSeconds: true,
			Run:             myJob,
		})
		if err != nil {
			panic(err) // for demo only
		}

		// register some distributed tasks
		err = miso.ScheduleDistributedTask(miso.Job{
			Cron:            "*/15 * * * *",
			CronWithSeconds: false,
			Name:            "MyDistributedTask",
			Run: func(miso miso.Rail) error {
				return jobDoSomething(rail)
			},
		})
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
  enabled: true
  user: root
  password: 123456
  database: mydb
  host: localhost
  port: 3306
```

## Include miso in your project

Get the latest release of miso:

```
go get -u github.com/curtisnewbie/miso
```

Or get the specific release of miso:

```
go get github.com/curtisnewbie/miso@v0.0.11
```

## Documentations

- [configuration](./doc/config.md)
- [application lifecycle](./doc/lifecycle.md)
- [distributed task scheduling](./doc/dtask.md)
- [validation](./doc/validate.md)
- [service healthcheck](./doc/health.md)
- [customize build](./doc/customize_build.md)
- [json processing](./doc/json.md)
- [pprof](./doc/pprof.md)
- More in the future (maybe) :D