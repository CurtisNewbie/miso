# miso

> **_main branch is unstable, install miso with tags instead_**

Miso, yet another simple application framework. It's mainly a fun project for me to prove: **_'yes, we can just write a framework ourselves.'_**.

Miso provides an opinionated way to write application, common functionalities such as configuration, service discovery, load balancing, log tracing, log rotation, task scheduling, message queue and so on, are all implemented in an opinionated way. You can use miso to write _almost_ any kind of application, but it's mainly a backend framework.

The overall target is to make it as small and simple as possible, backward compatibility may break in future releases.

## Include miso in your project

Install a specific release of miso:

```
go get github.com/curtisnewbie/miso@v0.1.21
```

## Documentations

- [CLI Tools](./doc/tools.md)
- [Configuration](./doc/config.md)
- [Application Lifecycle](./doc/lifecycle.md)
- [HTTP Api Declaration](./doc/web.md)
- [Tracing](./doc/trace.md)
- [Distributed Task Scheduling](./doc/dtask.md)
- [Validation](./doc/validate.md)
- [Service Healthcheck](./doc/health.md)
- [Json Processing Behaviour](./doc/json.md)
- [Rabbitmq and Event Bus](./doc/rabbitmq.md)
- [API Documentation Generation](./doc/api_doc_gen.md)
- [Using pprof](./doc/pprof.md)

## Projects that use miso

The following are projects that use miso (mine tho), see also [moon-monorepo](https://github.com/curtisnewbie/moon-monorepo).

- [gatekeeper](https://github.com/curtisnewbie/gatekeeper)
- [event-pump](https://github.com/curtisnewbie/event-pump)
- [mini-fstore](https://github.com/curtisnewbie/mini-fstore)
- [vfm](https://github.com/curtisnewbie/vfm)
- [user-vault](https://github.com/curtisnewbie/user-vault)
- [hammer](https://github.com/curtisnewbie/hammer)
- [goauth](https://github.com/curtisnewbie/goauth)
- [logbot](https://github.com/curtisnewbie/logbot)
- [doc-indexer](https://github.com/curtisnewbie/doc-indexer)
