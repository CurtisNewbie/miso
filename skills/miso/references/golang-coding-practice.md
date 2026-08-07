# Golang Coding Practice

General Go coding practices for miso services. Apply these to all new code.

**Table of Contents:**
- Defer Close Resources
- Never Panic
- Error Handling: Return or Log, Not Both
- Compile Regexp Once
- Functional Options
- Shutdown Awareness in Tasks
- Concurrency Protection with Locks

## Defer Close Resources

Almost always `defer` the close of a resource right where you acquire it — file, stream, HTTP response body, etc. This guarantees cleanup on every return path, including panics and early returns.

```go
f, err := os.Open(path)
if err != nil {
    return errs.Wrapf(err, "failed to open file")
}
defer f.Close()

// ... use f
```

`Close()` on writers can flush buffered data and report errors — check it when write integrity matters:

```go
defer func() {
    if err := f.Close(); err != nil {
        log.Printf("failed to close file: %v", err)
    }
}()
```

Avoid `defer` inside a loop — the closes accumulate until the function returns. Wrap the loop body in a closure instead:

```go
for _, path := range paths {
    err := func() error {
        f, err := os.Open(path)
        if err != nil {
            return err
        }
        defer f.Close()
        // ...
        return nil
    }()
    if err != nil {
        return err
    }
}
```

## Never Panic

Do not panic in business code. Panics crash the process (or are recovered far from the cause, losing context). Return errors instead — miso wraps and propagates them via `errs` and logs them with the Rail trace (see [error-handling.md](error-handling.md)).

```go
// Don't
if v == nil {
    panic("v is nil")
}

// Do
if v == nil {
    return errs.New("v is nil")
}
```

Legitimate panics are limited to programmer errors detected at startup, where failing fast is the correct behavior — e.g. `regexp.MustCompile` (see below) or misconfig that makes the service unusable. If you must recover, do it at a single outer boundary, never scattered through business logic.

## Error Handling: Return or Log, Not Both

When an error occurs, choose one of two paths — **return it to the caller** or **log it and continue** — never both.

**Return the error** (with or without added context) when the caller can act on it — retry, fall back, or propagate up the stack. Use `errs.Wrap`/`errs.Wrapf` to attach context without losing the original error chain (see [error-handling.md](error-handling.md)):

```go
if _, err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table(TableAccount).
    Where("id = ?", accountID).
    Scan(&account); err != nil {
    return errs.Wrapf(err, "failed to fetch account")
}
```

**Log and continue** when the error is recoverable and the current logic can proceed meaningfully without it — e.g. a non-critical side operation failed:

```go
if err := sendNotification(rail, accountID); err != nil {
    rail.Errorf("failed to send notification, continuing: %v", err)
}
// continue with the main flow
```

Doing both duplicates the error: it gets logged here and logged again wherever the wrapped error is handled, producing noisy, repeated log lines with no extra value. Prefer **returning** by default; reserve **log-and-continue** for genuinely non-critical failures where the caller cannot meaningfully react anyway.

Never **silently ignore** an error — bare `_ =` makes code "work" until the ignored failure matters, then fail in a way nobody can diagnose. If an error is truly irrelevant, say why — a warn log or explicit comment beats a silent discard:

```go
_ = f.Close() // Don't — silently swallowed

// Do — best-effort cleanup, failure doesn't affect the result
if err := f.Close(); err != nil {
    rail.Warnf("failed to close: %v", err)
}
```

This also applies to `defer`'d calls that return errors (e.g. `Close()`).

## Compile Regexp Once

`regexp.MustCompile` at package level compiles once at init — never compile inside a function that runs per request. Compiling is expensive and `MustCompile` panics on bad patterns, which is the right place to surface that error (at startup).

```go
var emailPattern = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)

func IsValidEmail(s string) bool {
    return emailPattern.MatchString(s)
}
```

Use `regexp.Compile` (returning an error instead of panicking) only when the pattern is dynamic.

## Functional Options

Construct complex types with functional options. Positional arguments with many fields are unreadable at call sites, and adding an option later breaks every caller — with options, defaults live in the constructor and each option is a self-documenting function.

```go
type Client struct {
    host    string
    timeout time.Duration
    retries int
}

type Option func(*Client)

func WithTimeout(t time.Duration) Option {
    return func(c *Client) { c.timeout = t }
}

func WithRetries(n int) Option {
    return func(c *Client) { c.retries = n }
}

func NewClient(host string, opts ...Option) *Client {
    c := &Client{host: host, timeout: 5 * time.Second, retries: 3}
    for _, opt := range opts {
        opt(c)
    }
    return c
}

// Usage
client := NewClient("example.com", WithTimeout(10*time.Second), WithRetries(5))
```

Good fit when a type has 3+ configurable fields or options that are usually omitted. For 1-2 simple fields, plain constructor arguments are fine.

## Shutdown Awareness in Tasks

Long-running tasks and loops must detect server shutdown and exit promptly, otherwise graceful shutdown hangs until the timeout. Use `miso.IsShuttingDown()` to check, and `miso.IsShuttingDownCh()` to block until shutdown begins (see [bootstrap.md](bootstrap.md)).

```go
func PollAndProcess(rail miso.Rail) error {
    for {
        // Stop accepting new work once shutdown starts
        if miso.IsShuttingDown() {
            rail.Infof("server shutting down, stopping task")
            return nil
        }

        records, err := fetchRecords(rail)
        if err != nil {
            return errs.Wrapf(err, "failed to fetch records")
        }
        if err := processRecords(rail, records); err != nil {
            return errs.Wrapf(err, "failed to process records")
        }
    }
}
```

For blocking waits, select on the shutdown channel instead of polling:

```go
func WaitAndProcess(rail miso.Rail) error {
    for {
        select {
        case <-miso.IsShuttingDownCh():
            return nil
        case rec := <-incoming:
            if err := handle(rail, rec); err != nil {
                return err
            }
        }
    }
}
```

Also relevant to cron tasks — check that a task worker knows when it should stop mid-execution, not just between iterations.

## Concurrency Protection with Locks

When multiple nodes or goroutines can modify the same record concurrently, use a lock to prevent lost updates and inconsistent state. Two options:

**Row lock** for read-modify-write cycles on a single DB row — lock the row for update inside a transaction so the read sees the latest committed value:

```go
import "gorm.io/gorm/clause"

var account Account
err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Clauses(clause.Locking{Strength: "UPDATE"}).
    Table(TableAccount).
    Where("id = ?", accountID).
    Scan(&account)
```

**Redis lock (RLock)** when the critical section spans multiple operations or services, or the shared resource isn't a DB row (see [redis.md](redis.md)). Prefer `TryLock` with a bounded `WithBackoff` timeout: `Lock()` blocks retrying for the full backoff window (default ~30s), stalling the caller even when the lock is held elsewhere, while `TryLock` returns `locked=false` so contention is handled explicitly:

```go
// TryLock — bounded wait, explicit contention handling
lock := redis.NewRLock(rail, "lock:account:123")
locked, err := lock.TryLock(redis.WithBackoff(3 * time.Second))
if err != nil {
    return err
}
if !locked {
    // lock busy — skip or return conflict, don't block the caller
    return nil
}
defer lock.Unlock()
```

Choose by scope: DB row lock for transactional single-row updates; Redis lock for cross-service coordination, multi-step jobs, or rate-limited critical sections.
