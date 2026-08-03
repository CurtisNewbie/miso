# Bootstrap (Component Lifecycle)

Ordered initialization of framework components with dependency management. Components register bootstrap callbacks in `init()`, and the framework runs them in order during server startup.

**Package:** `github.com/curtisnewbie/miso`

**Table of Contents:**
- Order Levels
- Registering Components
- Lifecycle Callbacks
- Bootstrap Lifecycle
- Shutdown Hooks
- Graceful Shutdown

## Order Levels

```go
const (
    BootstrapOrderL1      = -30  // Essential components, must be ready first (DB, cache)
    BootstrapOrderL2      = -20  // Bootstrapped before the web server (metrics)
    BootstrapOrderL3      = -10  // Web server (starting it doesn't mean inbound requests flow yet)
    BootstrapOrderDefault = 0    // Default order
    BootstrapOrderL4      = 10   // Inbound traffic / job scheduling: service registration,
                                // MQ connections — server considered truly running after this
)
```

Components with the same order run in registration order in practice (not strictly guaranteed). Middleware packages (mysql, redis, task, kafka, etc.) register their own bootstrap callbacks automatically when imported — no manual registration needed.

## Registering Components

```go
import (
    "github.com/curtisnewbie/miso"
    "github.com/curtisnewbie/miso/flow"
)

func init() {
    miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
        Name:      "Initialize Database",
        Bootstrap: dbBootstrap,
        Condition: dbBootstrapCondition,
        Order:     miso.BootstrapOrderL1,
    })
}

func dbBootstrap(rail flow.Rail) error {
    // initialization logic
    rail.Infof("Database initialized")
    return nil
}

func dbBootstrapCondition(rail flow.Rail) (bool, error) {
    // return true to bootstrap, false to skip
    return miso.GetPropBool("db.enabled"), nil
}
```

`Condition` is optional — omit it to always run. See [configuration.md](configuration.md) for config-driven conditional bootstrap.

## Lifecycle Callbacks

```go
// Runs after config is loaded, before any bootstrap (used by Nacos config center
// to load remote config)
miso.RegisterConfigLoader(func(rail miso.Rail) error { return nil })

// Runs before component bootstrap
miso.PreServerBootstrap(func(rail miso.Rail) error {
    rail.Infof("Preparing server")
    return nil
})

// Runs after all components are bootstrapped (post-server)
miso.PostServerBootstrap(func(rail miso.Rail) error {
    rail.Infof("Server started")
    return nil
})

// Runs after all PostServerBootstrap callbacks complete
miso.OnAppReady(func(rail miso.Rail) error {
    rail.Infof("Application ready")
    return nil
})
```

## Bootstrap Lifecycle

1. `BootstrapServer()` called
2. Configuration loaded from `conf.yml`
3. Logging configured
4. `RegisterConfigLoader` callbacks run
5. `PreServerBootstrap` callbacks run
6. Components bootstrapped in order (L1 → L2 → L3 → L4)
7. `PostServerBootstrap` callbacks run
8. `OnAppReady` callbacks triggered
9. Server ready signal
10. Wait for shutdown signal (SIGTERM/SIGINT)
11. Shutdown hooks run in ascending order value

## Shutdown Hooks

```go
import "github.com/curtisnewbie/miso/flow"

miso.AddShutdownHook(func() {
    flow.Infof("Cleaning up...")
})

miso.AddOrderedShutdownHook(1, func() {
    flow.Infof("Closing DB connections...")
})
```

Async variants run each hook in its own goroutine (concurrently rather than serially). Shutdown still waits for all hooks, up to the global `server.graceful-shutdown-time-sec` timeout:

```go
miso.AddAsyncShutdownHook(func() { /* runs in its own goroutine */ })
miso.AddOrderedAsyncShutdownHook(miso.DefShutdownOrder-1, func() { /* ... */ })
```

Hooks run in **ascending order value** — use `AddOrderedShutdownHook(order, ...)` to sequence them; within the same order, execution order is unspecified.

## Graceful Shutdown

```go
// Graceful-shutdown awareness for long-running code
if miso.IsShuttingDown() {
    // stop accepting new work
}
<-miso.IsShuttingDownCh() // blocks until shutdown begins

// Programmatic shutdown
miso.Shutdown()
```

`server.graceful-shutdown-time-sec` (default 30s) sets the global timeout for shutdown hooks + HTTP graceful shutdown:

```yaml
server:
  graceful-shutdown-time-sec: 30
```
