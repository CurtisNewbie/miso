# miso

> **_main branch is unstable, install miso with tags instead_**

miso, yet another simple application framework. miso provides an opinionated way to write backend application, it _'subjectively'_ solves many common issues that we all have at some point during development.

It's mainly a fun project for me to prove: **_'yes, we can just write a framework ourselves.'_**. Surprisingly, it does work.

The overall target is to make it as small and simple as possible, backward compatibility may break in future releases.

## Include miso in your project

Install a specific release of miso:

```
go get github.com/curtisnewbie/miso@v0.3.4
```

Again, miso is an opinionated framework, it might not be suitable for mature codebase, but you are free to explore this framework in a new project.

You can generate a new project using `misogen` (see [CLI Tools](./doc/tools.md)).

## Documentations

- [CLI Tools](./doc/tools.md)
- [Configuration](./doc/config.md)
- [Application Lifecycle](./doc/lifecycle.md)
- [HTTP Client](./doc/http_client.md)
- [HTTP API Declaration](./doc/web.md)
- [Tracing](./doc/trace.md)
- [Database](./doc/database.md)
- [Cron Scheduler & Distributed Task Scheduling](./doc/dtask.md)
- [Validation](./doc/validate.md)
- [Service Healthcheck](./doc/health.md)
- [Json Processing Behaviour](./doc/json.md)
- [Rabbitmq and Event Bus](./doc/rabbitmq.md)
- [Kafka](./doc/kafka.md)
- [API Documentation Generation](./doc/api_doc_gen.md)
- [Debugging Performance: pprof, trace and gops](./doc/perf.md)
- [List of Supported Middlewares](./doc/middlewares.md)
- [Upgrade & Migration](./doc/migration.md)

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

# Updates

- Since v0.2.0, 21 configuration properties are renamed, these property names are not backward compatible. If you specify values for these configuration, make sure you update the property name before you upgrade miso.
- Since v0.3.0, a bunch of breaking changes are introduced, e.g., refactoring package structure, removing deprecated code and so on.
  Lots of methods and types are moved from `/util` pkg to pkgs such as `cli`, `csv`, `errs`, `expr`, `flags`, `hash`, `heap`, `pair`, `queue`, `rfutil`, `slutil`, `stack`, and `strutil`.
  Use [./patch/v0.3.0.patch](./patch/v0.3.0.patch) for automatica migration. See [Upgrade & Migration](./doc/migration.md).
- Since v0.3.5, a few funcs in errs pkg are deprecated, Use [./patch/v0.3.0.patch](./patch/v0.3.0.patch) for automatica migration. See [Upgrade & Migration](./doc/migration.md).
