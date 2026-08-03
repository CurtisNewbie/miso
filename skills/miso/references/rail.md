# Rail (Tracing & Logging)

Rail is miso's primary abstraction for distributed tracing and structured logging. It wraps `context.Context` and carries trace/span/user information through the request lifecycle.

**Package:** `github.com/curtisnewbie/miso/flow` (also re-exported as `miso.Rail`, `miso.EmptyRail`, `miso.NewRail`, etc.)

**Table of Contents:**
- Creating Rail
- Passing Rail
- Async Operations
- Span & Trace Lifecycle
- Context Operations
- User Info
- Logging
- Global Logging
- Trace Propagation
- Timing

## Creating Rail

```go
import "github.com/curtisnewbie/miso/flow"

// Empty rail (no trace/span context)
rail := flow.EmptyRail()

// Rail from context — preserves existing trace/span values,
// otherwise generates a new trace id
rail := flow.NewRail(ctx)

// miso package re-exports the same functions
rail := miso.EmptyRail()
```

HTTP handlers and most framework callbacks (bootstrap, jobs, health checks) receive a Rail or an `*miso.Inbound` — use `inb.Rail()` there instead of creating one manually. See [web-development.md](web-development.md) for handlers.

## Passing Rail

Always pass `flow.Rail` down through function signatures (instead of `context.Context`) — it carries trace context for logging and lets downstream code call miso's tracing-enabled methods (HTTP client, dbquery, etc.):

```go
func ProcessOrder(rail flow.Rail, orderID string) error {
    rail.Infof("Processing order: %s", orderID)
    return nil
}
```

## Async Operations

For async operations (goroutines, background tasks), detach from the parent Rail's context so the operation is not cancelled when the parent finishes (e.g., HTTP handler returns), and give the operation a new span:

```go
// Detach from parent context — otherwise the async op inherits the parent's
// cancellation (e.g., HTTP handler rail context)
rail = rail.NewCtx()
// New span id for the new unit of work
rail = rail.NewSpanId()

go func() {
    rail.Infof("Running async operation")
    // ... rail is safe to use, not cancelled by the parent
}()
```

Note: by default (`server.handler.with-new-context: true`), the handler Rail already gets a fresh context, so async ops started directly with `inb.Rail()` are not cancelled on client disconnect. Use `NewCtx()` when the Rail comes from elsewhere or when that prop is disabled.

## Span & Trace Lifecycle

```go
// New span id, same trace id (marks a new unit of work, e.g., an async
// operation or an external API call)
childRail := rail.NewSpanId()

// New trace id + span id (for an independent operation)
newRail := rail.NewTrace()
```

Note: `NextSpan()` is deprecated since v0.4.15 — use `rail.NewCtx().NewSpanId()` if a new context is needed. `NextSpanId()` is an alias of `NewSpanId()`.

## Context Operations

```go
ctx := rail.Context()

// Rail with context.CancelFunc
rail2, cancel := rail.WithCancel()

// Rail with timeout and context.CancelFunc
rail2, cancel := rail.WithTimeout(5 * time.Second)
```

## User Info

Rail carries authenticated user info (populated by auth middleware from inbound headers, e.g. `x-userno`):

```go
user := rail.User() // flow.User (UserNo, Username, RoleNo)
```

## Logging

Log messages should start with a capital letter — consistent capitalization keeps logs scannable in the console and searchable in log aggregation tools.

```go
rail.Tracef("Trace message: %v", data)
rail.Debugf("Debug info: %v", data)
rail.Infof("Processing request: %v", req)
rail.Warnf("Warning: %v", msg)
rail.Errorf("Error occurred: %v", err)
rail.Fatalf("Fatal error: %v", err)
rail.Panicf("Panic: %v", msg)

// Non-formatted variants
rail.Debug(data)
rail.Info(data)
rail.Warn(data)
rail.Error(data)
rail.Warnln(data)
rail.Errorln(data)
```

Conditional logging — only logs when the error is not nil:

```go
rail.ErrorIf(err, "database operation failed")
rail.WarnIf(err, "cache miss occurred")
```

`rail.Errorf`/`rail.Warnf` automatically append the wrapped error's stack trace when the error is a `*errs.MisoErr`. See [error-handling.md](error-handling.md).

## Global Logging

Package-level logging functions backed by a zero-trace Rail — use them where no Rail is available (e.g., `init()`, main):

```go
flow.Infof("Server starting")
flow.Debugf("Debug message")
flow.Errorf("Error: %v", err)
flow.Fatalf("Fatal: %v", err)

// Also available as miso.Infof, miso.Errorf, etc.
```

## Trace Propagation

Miso automatically propagates trace context across components and services:

**Automatic:**
- Inbound HTTP requests: trace headers are extracted and loaded into Rail
- Outbound HTTP requests: use `miso.BuildTraceHeadersStr(rail)` to build trace headers for downstream calls
- RabbitMQ/Kafka messages: trace headers automatically attached to message headers and restored in consumer Rail
- New goroutines: use `rail.NewCtx()` to propagate trace context
- Database queries: trace context automatically propagated to the GORM SQL logger

**Default Propagation Keys:**

| Key | Meaning |
|-----|---------|
| `X-B3-TraceId` | Trace ID |
| `X-B3-SpanId` | Span ID |
| `x-username` | Username |
| `x-userno` | User number |
| `x-roleno` | Role number |

```go
// Build trace headers for downstream calls
headers := miso.BuildTraceHeadersStr(rail) // map[string]string

// Add custom propagation keys
flow.AddPropagationKeys("X-Custom-Header")
```

**Configuration:**

```yaml
# conf.yml
server:
  trace:
    inbound:
      propagate: true  # enable/disable trace propagation from inbound requests (default: true)
```

## Timing

```go
start := time.Now()
defer rail.TimeOp(start, "operation name, %v", arg)
```
