# Core Concepts

Detailed explanation of miso framework's core abstractions.

## Rail (Tracing Context)

Rail is the primary abstraction for distributed tracing and logging in miso. It wraps `context.Context` and adds trace/span information.

**Package:** `github.com/curtisnewbie/miso/flow`

### Creating Rail

```go
import "github.com/curtisnewbie/miso/flow"

// Empty rail (no trace)
rail := flow.EmptyRail()

// Rail from context (in HTTP handler)
rail := flow.NewRail(ctx)

// Child span
rail := rail.NextSpan()
```

### Logging Methods

```go
rail.Infof("Processing request: %v", req)
rail.Debugf("Debug info: %v", data)
rail.Warnf("Warning: %v", msg)
rail.Errorf("Error occurred: %v", err)
rail.Fatalf("Fatal error: %v", err)
```

### Trace Propagation

```go
// Get trace ID
tid := rail.TraceId()
sid := rail.SpanId()

// Context operations
ctx := rail.Ctx()
childRail := rail.NextSpan()

// Timing operations
start := time.Now()
rail.TimeOp(start, "Operation name")
```

### Trace Lifecycle

```go
// Create child span (same trace ID, new span ID)
childRail := rail.NextSpan()

// Create new trace (new trace ID, new span ID)
newRail := rail.NewTrace()
```

### Cross-Service Propagation

Miso automatically propagates trace information across components and services:

**Automatic Propagation:**
- Inbound HTTP requests: Trace headers are automatically extracted and loaded into Rail
- Outbound HTTP requests: Use `miso.BuildTraceHeadersStr(rail)` to get trace headers for downstream calls
- RabbitMQ messages: Trace headers are automatically propagated to message headers
- New goroutines: Use `rail.NewCtx()` to propagate trace context to background tasks
- Database queries: Trace context is automatically propagated to GORM logger

**Default Propagation Keys:**
- `X-B3-TraceId` - Trace ID
- `X-B3-SpanId` - Span ID
- `x-username` - Username
- `x-userno` - User number
- `x-roleno` - Role number

**Configuration:**

```yaml
# conf.yml
server:
  trace:
    inbound:
      propagate: true  # enable/disable trace propagation from inbound requests (default: true)
```

### Global Logging

```go
// Direct logging (uses zero-trace Rail)
flow.Infof("Server starting")
flow.Debugf("Debug message")
flow.Errorf("Error: %v", err)
```

### Rail in HTTP Handlers

```go
func MyHandler(inb *miso.Inbound, req MyReq) (MyRes, error) {
    rail.Infof("Processing request")
    // handler logic
    return res, nil
}
```

## Bootstrap (Component Lifecycle)

Bootstrap manages ordered initialization of framework components with dependency management.

**Package:** `github.com/curtisnewbie/miso`

### Order Levels

```go
const (
    BootstrapOrderL1      = -30  // Essential: DB, cache
    BootstrapOrderL2      = -20  // Pre-server: metrics
    BootstrapOrderL3      = -10  // Web server
    BootstrapOrderL4      = 10   // Post-server: service registration, jobs
    BootstrapOrderDefault = 0    // Other components
)
```

### Registering Components

```go
import "github.com/curtisnewbie/miso"

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

### Bootstrap Lifecycle

1. `BootstrapServer()` called
2. Configuration loaded from `conf.yml`
3. Logging configured
4. `PreServerBootstrap` callbacks run
5. Components bootstrapped in order (L1 → L2 → L3 → L4)
6. `PostServerBootstrap` callbacks run
7. `OnAppReady` callbacks triggered
8. Server ready signal
9. Wait for shutdown signal (SIGTERM/SIGINT)
10. Shutdown hooks run in reverse order

### Shutdown Hooks

```go
import "github.com/curtisnewbie/miso/flow"

miso.AddShutdownHook(func() {
    flow.Infof("Cleaning up...")
})

miso.AddOrderedShutdownHook(1, func() {
    flow.Infof("Closing DB connections...")
})
```

## Inbound (Request Context)

Inbound encapsulates HTTP request/response with automatic error handling and tracing.

**Package:** `github.com/curtisnewbie/miso`

### API Handler Signature

```go
func MyHandler(inb *miso.Inbound, req MyReq) (MyRes, error) {
    rail := inb.Rail()
    // handler logic
    return MyRes{Data: "result"}, nil
}
```

### Registering Handlers

```go
import "github.com/curtisnewbie/miso"

miso.HttpPost("/api/my-endpoint", miso.AutoHandler[MyReq, MyRes](MyHandler))
```

### Inbound Fields

```go
type Inbound struct {
    // Embedded Rail for tracing and logging
    flow.Rail
    // Internal fields (not directly exposed)
    // engine  *gin.Context
    // w       http.ResponseWriter
    // r       *http.Request
}
```

### Accessing Request Data

```go
// Query parameters
query := inb.Query("param")

// Headers
header := inb.Header("Authorization")

// User context (if auth middleware enabled)
user := inb.User()

// Get request/response
req := inb.Request()
writer := inb.Writer()

// Set headers
inb.SetHeader("X-Custom-Header", "value")
inb.AddHeader("X-Multi-Header", "value1")
```

### Response Handling

```go
// Success response
return MyRes{Data: "result"}, nil

// Error response
return MyRes{}, errs.NewErrf("Operation failed")
```

Framework automatically wraps responses for JSON serialization with standard response format.

## MisoErr (Error Type)

MisoErr provides structured error handling with error codes, messages, internal messages, and stack traces.

**Package:** `github.com/curtisnewbie/miso/errs`

### Creating Errors

```go
import "github.com/curtisnewbie/miso/errs"

// Simple error
err := errs.NewErrf("operation failed")

// Error with code
err := errs.NewErrfCode("USER_NOT_FOUND", "User does not exist")

// Error with internal message
err := errs.NewErrf("operation failed").
    WithInternalMsg("detailed debug information")

// Error with code and internal message
err := errs.NewErrfCode("DB_ERROR", "Database operation failed").
    WithInternalMsg("query failed: SELECT * FROM users")
```

### Wrapping Errors

```go
// Wrap existing error
return errs.WrapErr(err, "failed to process request")

// Wrap with formatted message
return errs.WrapErrf(err, "failed to load user: %s", userID)

// Wrap with additional context
err := errs.WrapErr(err, "database query failed").
    WithInternalMsg("query: SELECT * FROM users WHERE id = ?", userID)
```

### Error Methods

```go
// Message returned to client
msg := err.Msg()

// Internal debug message (server-only)
internalMsg := err.InternalMsg()

// Error code string
code := err.Code()

// Stack trace
stackTrace := err.StackTrace()
```

### Checking Errors

```go
// Check for none/not found error
if errs.IsNoneErr(err) {
    // handle not found
}

// Check for multiple specific errors
if errs.IsAny(err, ErrUserNotFound, ErrUserDeleted) {
    // handle specific errors
}
```

### Error Handling with Rail

```go
// Conditional logging (only logs if error is not nil)
rail.ErrorIf(err, "database operation failed")
rail.WarnIf(err, "cache miss occurred")

// Logging with error
rail.Errorf("Operation failed: %v", err)
```

### Error Response Format

```json
{
  "code": "USER_NOT_FOUND",
  "msg": "User does not exist",
  "data": null
}
```

Framework automatically converts `MisoErr` to JSON responses with `code`, `msg`, and `data` fields.