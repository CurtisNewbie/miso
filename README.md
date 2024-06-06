# miso

> ***main branch is unstable, install miso with tags instead***

Miso, yet another simple application framework. It's mainly a fun project for me to prove: ***'yes, we can just write a framework ourselves.'***.

Miso provides an opinionated way to write application, common functionalities such as configuration, service discovery, load balancing, log tracing, log rotation, task scheduling, message queue and so on, are all implemented in an opinionated way. You can use miso to write *almost* any kind of application, but it's mainly a backend framework.

The overall target is to make it as small and simple as possible, backward compatibility may break in future releases.

## Include miso in your project

Install a specific release of miso:

```
go get github.com/curtisnewbie/miso@v0.0.34
```

## Generate miso project

Install latest `misogen` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misogen@v0.0.34
```

Use `misogen` to generate new projects, e.g.,

```sh
$ misogen -h
Usage of misogen:
  -cli
        Generate CLI style project
  -disable-web
        Disable web server
  -name string
        Module name
  -static
        Generate code to embed and statically host frontend project
  -svc
        Generate code to integrate svc for automatic schema migration

$ mkdir myapp && cd myapp && misogen -name "myapp" -svc
misogen, current miso version: v0.0.34

Initialized module 'myapp'
Installing dependency: github.com/curtisnewbie/miso/miso@v0.0.34
Initializing conf.yml
Initializing internal/schema/scripts/schema.sql
Initializing internal/schema/migrate.go
Initializing main.go
```

## Documentations

- [Configuration](./doc/config.md)
- [Application Lifecycle](./doc/lifecycle.md)
- [Distributed Task Scheduling](./doc/dtask.md)
- [Validation](./doc/validate.md)
- [Service Healthcheck](./doc/health.md)
- [Json Processing Behaviour](./doc/json.md)
- [Using pprof](./doc/pprof.md)
- [Rabbitmq and Event Bus](./doc/rabbitmq.md)
- [API Documentation Generation](./doc/api_doc_gen.md)
- [Tracing](./doc/trace.md)
- More in the future (maybe) :D

## Projects that use miso

The following are projects that use miso (mine tho):

- [gatekeeper](https://github.com/curtisnewbie/gatekeeper)
- [event-pump](https://github.com/curtisnewbie/event-pump)
- [mini-fstore](https://github.com/curtisnewbie/mini-fstore)
- [vfm](https://github.com/curtisnewbie/vfm)
- [user-vault](https://github.com/curtisnewbie/user-vault)
- [hammer](https://github.com/curtisnewbie/hammer)
- [goauth](https://github.com/curtisnewbie/goauth)
- [logbot](https://github.com/curtisnewbie/logbot)
- [doc-indexer](https://github.com/curtisnewbie/doc-indexer)
