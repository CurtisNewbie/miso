# miso

Miso, yet another simple application framework. It's mainly a <i>learn-by-doing</i> project for me to explore various ideas that come across my mind. It's just very fun to implement stuff and realize that things can be very easy and straight-forward. This project is open sourced for love, but contribution is not really expected :D

Miso provides an opinionated way to write application, common functionalities such as configuration, service discovery, load balancing, log tracing, log rotation, task scheduling, message queue and so on, are all implemented in an opinionated way. You can use miso to write almost any kind of application.

The overall target is to make it as small and simple as possible, backward compatibility may break in future releases.

**How a miso app may look like (for demonstration only):**

```go
func main() {

	// after configuration loaded, before server bootstrap
	miso.PreServerBootstrap(func(rail miso.Rail) error {

		// declare event bus (for mq)
		miso.NewEventBus(demoEventBusName)

		// register some distributed tasks
		err := miso.ScheduleDistributedTask(miso.Job{
			Cron:                   "*/15 * * * *",
			CronWithSeconds:        false,
			Name:                   "MyDistributedTask",
			LogJobExec:             true,
			TriggeredOnBoostrapped: false,
			Run: func(miso miso.Rail) error {
				rail.Infof("MyDistributedTask running, now: %v", time.Now())
				return nil
			},
		})
		if err != nil {
			panic(err) // for demo only
		}

		type MyReq struct {
			Name string
			Age  int
		}
		type MyRes struct {
			ResultId string
		}
		// register endpoints, api-doc are automatically  generated based on MyReq/MyRes
		miso.IPost("/open/api/demo/post",
			func(c *gin.Context, rail miso.Rail, req MyReq) (MyRes, error) {
				rail.Infof("Received request: %#v", req)
				return MyRes{ResultId: "1234"}, nil // serialized to json
			}).
			Desc("Post demo stuff").                           // add description to api-doc
			DocHeader("Authorization", "Bearer Authorization") // document header param in api-doc

		// group routes based on shared url path
		miso.BaseRoute("/open/api/demo/grouped").Group(

			// /open/api/demo/grouped/post1
			miso.IPost("/post1", doSomethingEndpoint),

			miso.BaseRoute("/subgroup").Group(

				// /open/api/demo/grouped/subgroup/post2
				miso.IPost("/post2", doSomethingEndpoint),
			),
		)

		return nil
	})

	// after server fully bootstrapped, do some stuff
	miso.PostServerBootstrapped(func(rail miso.Rail) error {

		type GetUserInfoReq struct {
			UserNo string
		}
		type GetUserInfoRes struct {
			Username     string
			RegisteredAt miso.ETime
		}

		// maybe send some requests to other backend services
		// (using consul-based service discovery)
		var res GetUserInfoRes
		err := miso.NewDynTClient(rail, "/open/api/user", "user-vault").
			PostJson(GetUserInfoReq{UserNo: "123"}).
			Json(&res)

		if err != nil {
			rail.Errorf("request failed, %v", err)
		} else {
			rail.Infof("request succeded, %#v", res)
		}

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
go get github.com/curtisnewbie/miso@v0.0.19
```

## Documentations

- [Configuration](./doc/config.md)
- [Application Lifecycle](./doc/lifecycle.md)
- [Distributed Task Scheduling](./doc/dtask.md)
- [Validation](./doc/validate.md)
- [Service Healthcheck](./doc/health.md)
- [Customize Build](./doc/customize_build.md)
- [Json Processing Behaviour](./doc/json.md)
- [pprof](./doc/pprof.md)
- [Rabbitmq and Event Bus](./doc/rabbitmq.md)
- [API Documentation Generation](./doc/api_doc_gen.md)
- More in the future (maybe) :D

## Projects that use miso

The following are some projects that use miso (mine tho):

- [gatekeeper](https://github.com/curtisnewbie/gatekeeper)
- [event-pump](https://github.com/curtisnewbie/event-pump)
- [vfm](https://github.com/curtisnewbie/vfm)
- [user-vault](https://github.com/curtisnewbie/user-vault)
- [hammer](https://github.com/curtisnewbie/hammer)
- [goauth](https://github.com/curtisnewbie/goauth)
- [logbot](https://github.com/curtisnewbie/logbot)
- [doc-indexer](https://github.com/curtisnewbie/doc-indexer)