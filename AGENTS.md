# Miso Framework Agent Instructions

Go microservices framework with component-based architecture and ordered bootstrap lifecycle.

## Project Structure

- `miso/` - Core framework (bootstrap, web server, config, client)
- `middleware/` - Plugin components (mysql, redis, rabbit, kafka, nacos, etc.)
- `util/` - Utility packages (strutil, json, osutil, testutil, etc.)
- `flow/` - Distributed tracing context (Rail)
- `errs/` - Structured error types with stack traces
- `patch/` - Version migration patches for gopatch
- `cmd/` - CLI tools (misogen, misoapi, misocurl, misoconfig, misopatch)
- `demo/appdemo/` - Reference application
- `skills/miso/` - Agent skill with complete framework usage docs

## Framework Usage

For framework usage — bootstrap lifecycle, Rail tracing, web development, configuration, database operations, error handling, HTTP client, health checks, validation, performance monitoring, caching — use the **miso skill** (`skills/miso/SKILL.md`), which contains verified reference docs in `skills/miso/references/`:

- **rail.md** - Rail creation, span lifecycle, trace propagation, logging methods
- **bootstrap.md** - Order levels, component registration, lifecycle callbacks, shutdown hooks
- **web-development.md** - Routing, API patterns, middleware, misoapi code generation
- **database.md** - GORM usage, transactions, migrations, dbquery API
- **configuration.md** - Viper-based config, property constants, default values
- **error-handling.md** - Error types, wrapping, logging patterns with Rail
- **distributed-tasks.md** - Cron scheduling, master-worker pattern, task queues, hooks
- **service-discovery.md** - Nacos/Consul service registration, HTTP client with service discovery
- **cmd-tools.md** - misogen, misoapi, misocurl, misopatch, misoconfig
- **health-checks.md** - Health indicators, status monitoring
- **validation.md** - Struct validation rules, custom error messages, recursive validation
- **performance-monitoring.md** - pprof profiling, FlightRecorder traces, gops inspection
- **caching.md** - Local cache, TTL cache, Redis cache, cache patterns
- **redis.md** - Raw Redis operations, hash/list/set, distributed locks, scripting
- **kafka.md** - Produce/consume messages, listener semantics, partition ordering, trace propagation
- **middleware-utils.md** - crypto, JWT, expr, Lua, money, ZooKeeper leader election
- **util.md** - atom.Time (always use instead of time.Time), json, strutil, slutil, retry, randutil, IDs, async, osutil, testutil, flags, excel, copyutil
- **prometheus.md** - Histogram/Counter/HistogramVec constructors, HistTimer/VecTimer, scrape endpoint config, Push Gateway

## Quick Start Commands

Generate new project:
```bash
go install github.com/curtisnewbie/miso/cmd/misogen@v0.4.13
mkdir myapp && cd myapp && misogen -name "myapp"
```

Generate API endpoints from code:
```bash
go install github.com/curtisnewbie/miso/cmd/misoapi@v0.4.13
misoapi              # Generate endpoint registration code
```

Generate HTTP client from curl:
```bash
go install github.com/curtisnewbie/miso/cmd/misocurl@v0.4.13
misocurl  # Reads curl command from clipboard
```

Apply version migration:
```bash
go install github.com/uber-go/gopatch@latest
go install github.com/curtisnewbie/miso/cmd/misopatch@v0.4.13
misopatch  # Auto-applies all patches using gopatch
```

## Repo-Specific Notes

- **Main branch unstable** - Install via tags: `go get github.com/curtisnewbie/miso@vX.Y.Z`
- **Go workspace** - Demo apps use go.work for local development
- **Version migration** - Breaking changes are common between versions; `misopatch` auto-applies patches from `patch/` via gopatch (manual: `gopatch -p /path/to/miso/patch/v0.4.0.patch ./...`)
- **Test environment** - `miso.PrepareTestEnv(t)` bootstraps the full framework; finds `conf.yml` by walking up from the test file directory
