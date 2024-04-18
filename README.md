# miso

> ***main branch is unstable, install miso with tags instead***

Miso, yet another simple application framework. It's mainly a <i>learn-by-doing</i> project for me to explore various ideas that come across my mind. This project is open sourced for love, but contribution is not really expected :D

Miso provides an opinionated way to write application, common functionalities such as configuration, service discovery, load balancing, log tracing, log rotation, task scheduling, message queue and so on, are all implemented in an opinionated way. You can use miso to write almost any kind of application.

The overall target is to make it as small and simple as possible, backward compatibility may break in future releases.

**How a miso app may look like (for demonstration only):**

```go
func main() {
  // register callbacks that are invoked after configuration loaded, before server bootstrap
  miso.PreServerBootstrap(PrepareServer)

  // register callbacks that are invoked after server fully bootstrapped
  miso.PostServerBootstrapped(TriggerWorkflowOnBootstrapped)

  // start the bootstrap process
  miso.BootstrapServer(os.Args)
}

func PrepareServer(rail miso.Rail) error {

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

  // register endpoints, api-doc are automatically generated
  // routes can also be grouped based on shared url path
  miso.BaseRoute("/open/api/demo/grouped").Group(

    // /open/api/demo/grouped/post
    miso.IPost("/open/api/demo/post",
      func(inb *miso.Inbound, req PostReq) (PostRes, error) {
        rail := inb.Rail()
        rail.Infof("Received request: %#v", req)

        // e.g., read some table
        var res PostRes
        err := miso.GetMySQL().
          Raw(`SELECT result_id FROM post_result WHERE request_id = ?`,
            req.RequestId).
          Scan(&res).Error
        if err != nil {
          return PostRes{}, err
        }

        return res, nil // serialized to json
      }).
      Desc("Post demo stuff").                            // describe endpoint in api-doc
      DocHeader("Authorization", "Bearer Authorization"), // document request header

    miso.BaseRoute("/subgroup").Group(

      // /open/api/demo/grouped/subgroup/post
      miso.IPost("/post1", doSomethingEndpoint),
    ),
  )
  return nil
}

func TriggerWorkflowOnBootstrapped(rail miso.Rail) error {
  // maybe send some requests to other backend services
  // (using consul-based service discovery)
  var res TriggerResult
  err := miso.NewDynTClient(rail, "/open/api/engine", "workflow-engine" /* service name */).
    PostJson(TriggerWorkFlow{WorkFlowId: "123"}).
    Json(&res)

  if err != nil {
    rail.Errorf("request failed, %v", err)
  } else {
    rail.Infof("request succeded, %#v", res)
  }
  return nil
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

Install a specific release of miso:

```
go get github.com/curtisnewbie/miso@v0.0.29
```

## Documentations

- [Configuration](./doc/config.md)
- [Application Lifecycle](./doc/lifecycle.md)
- [Distributed Task Scheduling](./doc/dtask.md)
- [Validation](./doc/validate.md)
- [Service Healthcheck](./doc/health.md)
- [Customize Build](./doc/customize_build.md)
- [Json Processing Behaviour](./doc/json.md)
- [Using pprof](./doc/pprof.md)
- [Rabbitmq and Event Bus](./doc/rabbitmq.md)
- [API Documentation Generation](./doc/api_doc_gen.md)
- [Tracing](./doc/trace.md)
- More in the future (maybe) :D

## Projects that use miso

The following are some projects that use miso (mine tho):

- [gatekeeper](https://github.com/curtisnewbie/gatekeeper)
- [event-pump](https://github.com/curtisnewbie/event-pump)
- [mini-fstore](https://github.com/curtisnewbie/mini-fstore)
- [vfm](https://github.com/curtisnewbie/vfm)
- [user-vault](https://github.com/curtisnewbie/user-vault)
- [hammer](https://github.com/curtisnewbie/hammer)
- [goauth](https://github.com/curtisnewbie/goauth)
- [logbot](https://github.com/curtisnewbie/logbot)
- [doc-indexer](https://github.com/curtisnewbie/doc-indexer)