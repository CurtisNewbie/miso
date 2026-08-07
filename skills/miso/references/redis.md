# Redis API

Direct Redis API wrappers with Rail context tracing, consistent error handling, and typed JSON helpers. Import: `"github.com/curtisnewbie/miso/middleware/redis"`.

**Table of Contents:**
- Configuration
- Core Concepts
- Key-Value Operations
- Counter Operations
- List & Blocking List Operations
- Set Operations
- Hash Operations
- Batch & Scripting
- Caching (RCache)
- Distributed Locks (RLock)
- Pub/Sub (Topic)
- Best Practices

## Configuration

```yaml
redis:
  enabled: true
  address: localhost   # default
  port: 6379           # default
  database: 0          # default
  password: ""
```

When `redis.enabled` is `true`, importing the package auto-connects at bootstrap (L1). Otherwise call `InitRedisFromProp(rail)` manually.

## Core Concepts

### GetRedis() + Rail

All functions take `rail miso.Rail` as first param for trace propagation, and delegate to `GetRedis()` (a singleton `*redis.Client`).

```go
import "github.com/curtisnewbie/miso/middleware/redis"

func Handler(inb *miso.Inbound) {
    rail := inb.Rail()
    val, ok, err := redis.Get(rail, "mykey")
    // ...
}
```

### redis.Nil Handling

Functions that may encounter "key not found" return `(value, false, nil)` — NOT an error. Check `ok`:

```go
val, ok, err := redis.Get(rail, "key")
if err != nil { /* real error */ }
if !ok      { /* key not found */ }
```

Functions using this `(T, bool, error)` pattern:
`Get`, `GetJson`, `GetDel`, `LPop`, `RPop`, `LPopJson`, `RPopJson`, `BLPop`, `BRPop`, `BRPopAny`, `BLPopJson`, `BRPopJson`, `HGet`, `HGetJson`.

All other functions return errors directly (wrapped via `errs.Wrap(err)`).

### JSON Helpers

Use `*Json` variants for automatic serialization — avoid manual marshal/unmarshal:

```go
user, ok, err := redis.GetJson[User](rail, key)        // auto-deserialize
err := redis.SetJson(rail, key, userObj, 5*time.Minute) // auto-serialize
```

## Key-Value Operations

```go
// Set and Get
err := redis.Set(rail, "key", "value", 5*time.Minute)
val, ok, err := redis.Get(rail, "key")

// JSON
err = redis.SetJson(rail, "user:1", user, 5*time.Minute)
user, ok, err = redis.GetJson[User](rail, "user:1")

// Atomic GET+DELETE (returns value then deletes key)
val, ok, err = redis.GetDel(rail, "key")

// Bulk delete
n, err := redis.Del(rail, "k1", "k2", "k3")
```

| Function | Signature |
|----------|-----------|
| `Set` | `(rail, key string, val any, exp time.Duration) error` |
| `SetNX` | `(rail, key string, val any, exp time.Duration) (bool, error)` |
| `SetJson` | `(rail, key string, val any, exp time.Duration) error` |
| `SetNXJson` | `(rail, key string, val any, exp time.Duration) (bool, error)` |
| `Get` | `(rail, key string) (string, bool, error)` |
| `GetJson[T]` | `(rail, key string) (T, bool, error)` |
| `GetDel` | `(rail, key string) (string, bool, error)` |
| `Exists` | `(rail, key string) (bool, error)` |
| `Expire` | `(rail, key string, exp time.Duration) (bool, error)` |
| `TTL` | `(rail, key string) (time.Duration, error)` |
| `Del` | `(rail, keys ...string) (int64, error)` |
| `Scan` | `(rail, pat string, scanLimit int64, f func(key string) error) error` |

## Counter Operations

If key doesn't exist, initializes to 0 before applying the operation.

```go
n, err := redis.Incr(rail, "views")
n, err := redis.IncrBy(rail, "score", 10)
```

| Function | Signature |
|----------|-----------|
| `Incr` | `(rail, key string) (int64, error)` |
| `Decr` | `(rail, key string) (int64, error)` |
| `IncrBy` | `(rail, key string, v int64) (int64, error)` |
| `DecrBy` | `(rail, key string, v int64) (int64, error)` |

## List & Blocking List Operations

### Queue pattern

```go
// Producer
redis.RPushJson(rail, "queue:tasks", task)

// Consumer (blocking, single queue)
vals, ok, err := redis.BRPopJson[Task](rail, 5*time.Second, "queue:tasks")

// Consumer (blocking, multi-queue with priority)
// BRPopAny returns [key, value] so caller identifies the source queue
vals, ok, err := redis.BRPopAny(rail, 5*time.Second, "queue:high", "queue:low")
```

### Non-blocking operations

| Function | Signature |
|----------|-----------|
| `LPush` | `(rail, key string, v any) error` |
| `RPush` | `(rail, key string, v any) error` |
| `LPushJson` | `(rail, key string, v any) error` |
| `RPushJson` | `(rail, key string, v any) error` |
| `LPop` | `(rail, key string) (string, bool, error)` |
| `RPop` | `(rail, key string) (string, bool, error)` |
| `LPopJson[T]` | `(rail, key string) (T, bool, error)` |
| `RPopJson[T]` | `(rail, key string) (T, bool, error)` |
| `LLen` | `(rail, key string) (int64, error)` |
| `LRange` | `(rail, key string, start, stop int64) ([]string, error)` |
| `LRem` | `(rail, key string, count int64, value any) (int64, error)` |

### Blocking operations

| Function | Signature |
|----------|-----------|
| `BLPop` | `(rail, timeout time.Duration, key string) ([]string, bool, error)` |
| `BRPop` | `(rail, timeout time.Duration, key string) ([]string, bool, error)` |
| `BRPopAny` | `(rail, timeout time.Duration, keys ...string) ([]string, bool, error)` |
| `BLPopJson[T]` | `(rail, timeout time.Duration, key string) ([]T, bool, error)` |
| `BRPopJson[T]` | `(rail, timeout time.Duration, key string) ([]T, bool, error)` |

## Set Operations

```go
redis.SAdd(rail, "online:users", "user1", "user2")
isMember, _ := redis.SIsMember(rail, "online:users", "user1")
```

| Function | Signature |
|----------|-----------|
| `SAdd` | `(rail, key string, members ...interface{}) (int64, error)` |
| `SMembers` | `(rail, key string) ([]string, error)` |
| `SIsMember` | `(rail, key, member string) (bool, error)` |
| `SRem` | `(rail, key string, members ...interface{}) (int64, error)` |
| `SCard` | `(rail, key string) (int64, error)` |

## Hash Operations

```go
// Basic
redis.HSet(rail, "user:1", "name", "john", "age", "30")
name, ok, _ := redis.HGet(rail, "user:1", "name")
all, _ := redis.HGetAll(rail, "user:1")

// Typed JSON
redis.HSetJson(rail, "user:1", "profile", Profile{Bio: "hi", Level: 5})
profile, ok, _ := redis.HGetJson[Profile](rail, "user:1", "profile")

// Atomic counter
n, _ := redis.HIncrBy(rail, "stats:daily", "pageViews", 1)
```

| Function | Signature |
|----------|-----------|
| `HSet` | `(rail, key string, values ...interface{}) (int64, error)` |
| `HGet` | `(rail, key, field string) (string, bool, error)` |
| `HGetAll` | `(rail, key string) (map[string]string, error)` |
| `HGetJson[T]` | `(rail, key, field string) (T, bool, error)` |
| `HSetJson` | `(rail, key, field string, val interface{}) error` |
| `HDel` | `(rail, key string, fields ...string) (int64, error)` |
| `HExists` | `(rail, key, field string) (bool, error)` |
| `HKeys` | `(rail, key string) ([]string, error)` |
| `HVals` | `(rail, key string) ([]string, error)` |
| `HLen` | `(rail, key string) (int64, error)` |
| `HIncrBy` | `(rail, key, field string, incr int64) (int64, error)` |

## Batch & Scripting

```go
// Batch get/set — reduce round-trips
vals, err := redis.MGet(rail, "k1", "k2", "k3")
err = redis.MSet(rail, "k1", "v1", "k2", "v2")

// Lua scripting
result, err := redis.Eval(rail, `return redis.call("GET", KEYS[1])`, []string{"mykey"})
```

| Function | Signature |
|----------|-----------|
| `MGet` | `(rail, keys ...string) ([]interface{}, error)` |
| `MSet` | `(rail, values ...interface{}) error` |
| `Eval` | `(rail, script string, keys []string, args ...interface{}) (any, error)` |
| `Script` | `(script string) *redis.Script` |

## Caching (RCache)

Typed Redis cache with supplier pattern. Preferred over raw API for standard caching:

```go
var userCache = redis.NewRCache[User]("user", redis.RCacheConfig{
    Exp: 5 * time.Minute,
})

user, err := userCache.GetValElse(rail, id, func() (*User, error) {
    return loadFromDB(id)
})
```

Also: `RCacheV2` for complex key types (struct with `String()`), `GroupCache` for partitioned caches. See [caching.md](caching.md) for full patterns.

## Distributed Locks (RLock)

Prefer `TryLock` with a bounded `WithBackoff` timeout — `Lock()` blocks retrying for the full backoff window (default ~30s), while `TryLock` returns `locked=false` on contention so the caller can handle it explicitly:

```go
// TryLock — bounded wait, explicit contention handling
lock := redis.NewRLock(rail, "lock:account:123")
locked, err := lock.TryLock(redis.WithBackoff(3 * time.Second))
if err != nil {
    return err
}
if !locked {
    return nil // lock busy — skip or handle contention
}
defer lock.Unlock()
```

## Pub/Sub (Topic)

```go
topic := redis.NewTopic[Event]("user:events")

// Subscribe
cancel, _ := topic.SubscribeSync(func(rail miso.Rail, evt Event) error {
    rail.Infof("Event: %+v", evt)
    return nil
})
defer cancel()

// Publish
topic.Publish(rail, Event{Type: "user_created"})
```

## Best Practices

1. **Always check the `ok` boolean** on get/read operations — missing key is not an error.
2. **Use JSON helpers** for struct values (`SetJson`/`GetJson`, `HSetJson`/`HGetJson`).
3. **Prefer RCache** for standard cache patterns (supplier loading, TTL refresh, batch delete).
4. **Use raw API** when you need specific data structures (hash, set, sorted set) or fine-grained control.
5. **Prefer BRPopAny** for work queues with priority levels — returns `[key, value]` to identify source queue.
6. **Use `Del(rail, keys...)`** for batch key deletion in one round-trip.
