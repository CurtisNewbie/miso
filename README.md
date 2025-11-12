# miso

> **_main branch is unstable, install miso with tags instead_**

miso, yet another simple application framework.

Initially, it was a fun project for me to prove: **_'yes, we can just write a framework ourselves.'_**. Surprisingly, it does work very well.

miso provides an opinionated way to write applications (mainly backend web services). It's convenient enough with reasonable code complexity.
Do not expect it to be a full-fledged framework, it's surely not.

The overall target is to make it as small and simple as possible, backward compatibility may break in future releases.

## Include miso in your project

Install a specific release of miso:

```
go get github.com/curtisnewbie/miso@v0.3.9
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

> [!IMPORTANT]
>
> See [Upgrade & Migration](./doc/migration.md) for automatic migration.

- Since v0.2.0, 21 configuration properties are renamed, these property names are not backward compatible. If you are using deprecated configuration names, error log are printed as a warning.
- Since v0.3.0, a bunch of breaking changes are introduced, e.g., refactoring package structure, removing deprecated code and so on.
  Lots of methods and types are moved from `util` pkg to pkgs such as `cli`, `csv`, `errs`, `expr`, `flags`, `hash`, `heap`, `pair`, `queue`, `rfutil`, `slutil`, `stack`, and `strutil`.
- Since v0.3.5, a few funcs in `errs` pkg are deprecated.
- Since v0.3.6, file (os) related funcs in pkg `util` are deprecated and moved to `util/osutil`; and `FindTestdata(..)` func in pkg `util` is deprecated and moved to `util/testutil`.
- Since v0.3.7, async code in pkg `util` is deprecated and moved to `util/async`, while previous code may continue to work, it will be deleted in later release.
- Since v0.4.0,
  - The default lowercase camel case json field naming strategy has been removed.
  - a lot of deprecated code is removed. All code directly under `util` pkg is moved to more dedicated pkgs: `util/strutil`, `util/snowflake`, `util/randutil`, `util/iputil`, `util/pool`, `util/profile`, `util/constraint`, `util/atom`, `util/must`, `util/cmputil`.

> [!WARNING]
>
> Since [Jsoniter](https://github.com/json-iterator/go) is nolonger actively maintained, last commit was 3 years ago. Jsoniter will be removed from this repository in later release. The default [json processing behaviour](./doc/json.md) has been removed. Previously, all struct fields are by default serialized/deserialized using lowercase camel case style without needing to add any json tag. **Since v0.4.0**, json tags must be added manually to maintain compatibility.