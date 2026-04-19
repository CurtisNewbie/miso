# Caching

In-memory caching with TTL, LRU eviction, and distributed caching support.

## Overview

Miso provides multiple caching strategies:

1. **LocalCache** - Simple map-based cache without expiration
2. **LocalCacheV2** - Generic key/value cache without expiration
3. **TTLCache** - Time-based cache with automatic expiration and LRU eviction
4. **Redis Cache** - Distributed caching with Redis backend

## LocalCache

Simple string-keyed cache without expiration.

### Creating LocalCache

```go
import "github.com/curtisnewbie/miso"

func MyHandler(inb *miso.Inbound) {
    // Create cache
    cache := miso.NewLocalCache[User]()

    // Get value with supplier function
    user, err := cache.Get("user:123", func(key string) (User, error) {
        // Load from database
        return loadUserFromDB(key)
    })
    if err != nil {
        inb.HandleResult(nil, err)
        return
    }

    inb.WriteJson(user)
}
```

### Usage Pattern

```go
// Use as local variable for single-operation caching
// Benefit: Cache results that are reused multiple times within one operation
func GetUsersWithOrders(userIDs []string) ([]UserWithOrders, error) {
    // Create cache for this operation - avoids loading same user multiple times
    cache := miso.NewLocalCache[User]()

    var result []UserWithOrders
    for _, userID := range userIDs {
        // Cache prevents redundant DB calls if same user appears multiple times
        user, err := cache.Get(fmt.Sprintf("user:%s", userID), func(key string) (User, error) {
            return db.GetUser(userID)
        })
        if err != nil {
            return nil, err
        }

        orders, err := db.GetOrdersByUser(userID)
        if err != nil {
            return nil, err
        }

        result = append(result, UserWithOrders{User: user, Orders: orders})
    }

    // Cache is discarded after function ends - appropriate for short-lived use
    return result, nil
}
```

**Important:** `LocalCache` should not be a long-lived object. Create new instances as needed for individual operations.

## LocalCacheV2

Generic key/value cache without expiration.

### Creating LocalCacheV2

```go
import "github.com/curtisnewbie/miso"

func MyHandler(inb *miso.Inbound) {
    // Create cache with int keys
    cache := miso.NewLocalCacheV2[int, User]()

    // Get value
    if user, ok := cache.TryGet(123); ok {
        inb.WriteJson(user)
        return
    }

    // Load and cache
    user, err := loadUser(123)
    if err != nil {
        inb.HandleResult(nil, err)
        return
    }
    cache.Set(123, user)
    inb.WriteJson(user)
}
```

### Cache with Supplier

```go
cache := miso.NewLocalCacheV2[string, Config]()

config, err := cache.Get("app.config", func() (Config, error) {
    return loadConfigFromDB()
})
if err != nil {
    return err
}
```

### Available Methods

```go
type LocalCacheV2[K comparable, T any]

// Try get value without loading
func (lc LocalCacheV2[K, T]) TryGet(key K) (T, bool)

// Get value with supplier function
func (lc LocalCacheV2[K, T]) Get(key K, supplier func() (T, error)) (T, error)

// Set value directly
func (lc LocalCacheV2[K, T]) Set(key K, t T)

// Convert to map
func (lc LocalCacheV2[K, T]) ToMap() map[K]T
```

## TTLCache

Time-based cache with automatic expiration and LRU eviction.

### Creating TTLCache

```go
import "github.com/curtisnewbie/miso"

// Create TTL cache with 5-minute TTL and max 1000 items
cache := miso.NewTTLCache[User](5*time.Minute, 1000)
```

### Basic Usage

```go
import (
    "github.com/curtisnewbie/miso"
    "time"
)

func GetUser(id string) (*User, error) {
    cache := miso.NewTTLCache[User](5*time.Minute, 1000)

    // Get with supplier
    user, ok := cache.Get(fmt.Sprintf("user:%s", id), func() (*User, bool) {
        user, err := loadUserFromDB(id)
        if err != nil {
            return nil, false
        }
        return user, true
    })

    if !ok {
        return nil, fmt.Errorf("user not found")
    }
    return user, nil
}
```

### TTLCache Methods

```go
type TTLCache[T any]

// Try get value
func (tc TTLCache[T]) TryGet(key string) (T, bool)

// Get with supplier function
func (tc TTLCache[T]) Get(key string, elseGet func() (T, bool)) (T, bool)

// Put value
func (tc TTLCache[T]) Put(key string, t T)

// Delete value
func (tc TTLCache[T]) Del(key string)

// Check if key exists
func (tc TTLCache[T]) Exists(key string) bool

// Put only if absent
func (tc TTLCache[T]) PutIfAbsent(key string, t T) bool

// Get cache size
func (tc TTLCache[T]) Size() int

// Get all keys
func (tc TTLCache[T]) Keys() []string

// Register eviction callback
func (tc TTLCache[T]) OnEvicted(f func(key string, t T))
```

### Eviction Callback

```go
cache := miso.NewTTLCache[User](5*time.Minute, 1000)

// Register callback when item is evicted
cache.OnEvicted(func(key string, user User) {
    rail.Infof("User evicted from cache: %s", key)
})
```

### TTL and Size Management

- **TTL**: Each entry has a timestamp checked on access
- **Max Size**: When exceeded, least recently put item is evicted
- **Lazy Cleanup**: Eviction only happens on key lookup (no background goroutine)

### Advanced Usage

```go
import "github.com/curtisnewbie/miso"
import "time"

type CacheService struct {
    userCache    miso.TTLCache[User]
    configCache  miso.TTLCache[Config]
}

func NewCacheService() *CacheService {
    cs := &CacheService{}

    // User cache: 10 min TTL, max 10000 items
    cs.userCache = miso.NewTTLCache[User](10*time.Minute, 10000)

    // Config cache: 1 hour TTL, max 100 items
    cs.configCache = miso.NewTTLCache[Config](1*time.Hour, 100)

    // Add eviction logging
    cs.userCache.OnEvicted(func(key string, user User) {
        miso.Infof("User cache evicted: %s", key)
    })

    return cs
}

func (cs *CacheService) GetUser(id string) (*User, error) {
    user, ok := cs.userCache.Get(id, func() (*User, bool) {
        user, err := loadUserFromDB(id)
        if err != nil {
            miso.Errorf("Failed to load user: %v", err)
            return nil, false
        }
        return user, true
    })

    if !ok {
        return nil, fmt.Errorf("user not found")
    }
    return user, nil
}
```

## Redis Cache

Distributed caching with Redis backend using the `middleware/redis` package.

### Redis Helper Functions

```go
import (
    "github.com/curtisnewbie/miso/middleware/redis"
    "time"
)

// Create RCache instance
var userCache = redis.NewRCache[User]("user", redis.RCacheConfig{
    Exp:    5 * time.Minute,
    NoSync: false, // use distributed lock for synchronization
})

// Get from cache with automatic loading
func GetUser(id string) (*User, error) {
    key := fmt.Sprintf("%s", id)

    // GetValElse loads from cache, runs supplier on miss
    return userCache.GetValElse(rail, key, func() (*User, error) {
        return loadUserFromDB(id)
    })
}

// Manual get and put
func GetAndCacheUser(id string) (*User, error) {
    key := fmt.Sprintf("%s", id)

    // Check if exists
    exists, _ := userCache.Exists(rail, key)
    if exists {
        return userCache.GetVal(rail, key)
    }

    // Load from DB and cache
    user, err := loadUserFromDB(id)
    if err != nil {
        return nil, err
    }

    userCache.Put(rail, key, user)
    return user, nil
}
```

### Using RCacheV2 for Complex Keys

```go
import (
    "fmt"
    "github.com/curtisnewbie/miso/middleware/redis"
    "time"
)

type CacheKey struct {
    UserID   string
    DataType string
}

func (k CacheKey) String() string {
    return fmt.Sprintf("%s:%s", k.UserID, k.DataType)
}

// Create RCacheV2 with struct keys
var dataCache = redis.NewRCacheV2[CacheKey, Data]("data", redis.RCacheConfig{
    Exp: 10 * time.Minute,
})

func GetUserData(userID string, dataType string) (*Data, error) {
    key := CacheKey{UserID: userID, DataType: dataType}

    // Get with supplier - auto-caches on miss
    return dataCache.GetValElse(rail, key, func() (*Data, error) {
        return loadDataFromDB(userID, dataType)
    })
}
```

### RCache Operations

```go
import "github.com/curtisnewbie/miso/middleware/redis"

// Get value (returns error if not found)
user, err := userCache.GetVal(rail, "user:123")

// Get with supplier (loads on miss)
user, err := userCache.GetValElse(rail, "user:123", func() (*User, error) {
    return loadUserFromDB("123")
})

// Get with full control (returns value, found, error)
user, found, err := userCache.Get(rail, "user:123")

// Get with supplier callback (returns value, found, error)
user, found, err := userCache.GetElse(rail, "user:123", func() (*User, bool, error) {
    user, err := loadUserFromDB("123")
    if err != nil {
        return nil, false, err
    }
    return user, true, nil
})

// Put value into cache
err := userCache.Put(rail, "user:123", user)

// Delete from cache
err := userCache.Del(rail, "user:123")

// Check if key exists
exists, err := userCache.Exists(rail, "user:123")

// Refresh TTL for existing key
err := userCache.RefreshTTL(rail, "user:123")

// Delete all keys in this cache
err := userCache.DelAll(rail)
```

## Cache Comparison

| Cache Type | Expiration | Max Size | Distributed | Thread-Safe | Use Case |
|------------|------------|----------|-------------|-------------|----------|
| `LocalCache` | No | No | No | Yes | Short-lived, simple caching |
| `LocalCacheV2` | No | No | No | Yes | Generic key/value cache |
| `TTLCache` | Yes | Yes | No | Yes | In-memory with TTL |
| `Redis Cache` | Yes | Yes | Yes | Yes | Distributed caching |

## Best Practices

### 1. Choose Right Cache Type

```go
// Simple, short-lived cache
func GetConfig() (*Config, error) {
    cache := miso.NewLocalCache[Config]()
    return cache.Get("config", loadConfig)
}

// In-memory with TTL
func GetUser(id string) (*User, error) {
    cache := miso.NewTTLCache[User](5*time.Minute, 1000)
    return cache.Get(id, loadUser)
}

// Distributed cache
var sessionCache = redis.NewRCache[Session]("session", redis.RCacheConfig{
    Exp: 30 * time.Minute,
})

func GetSession(token string) (*Session, error) {
    return sessionCache.GetValElse(rail, token, loadSession)
}
```

### 2. Handle Cache Misses Gracefully

```go
user, ok := cache.Get(id, func() (*User, bool) {
    user, err := loadUserFromDB(id)
    if err != nil {
        rail.Errorf("Failed to load user: %v", err)
        return nil, false  // Don't cache failures
    }
    return user, true
})

if !ok {
    return nil, fmt.Errorf("user not found")
}
```

### 3. Use Appropriate TTL

```go
// User data - moderate TTL (5-10 min)
userCache := miso.NewTTLCache[User](5*time.Minute, 10000)

// Config data - longer TTL (1 hour)
configCache := miso.NewTTLCache[Config](1*time.Hour, 100)

// Session data - short TTL (30 min)
sessionCache := miso.NewTTLCache[Session](30*time.Minute, 5000)
```

### 4. Cache Invalidation

```go
// Delete cache when data changes
func UpdateUser(user *User) error {
    if err := db.UpdateUser(user); err != nil {
        return err
    }

    // Invalidate cache
    key := fmt.Sprintf("user:%s", user.ID)
    cache.Del(key)
    userCache.Del(rail, key)

    return nil
}
```

## Configuration

No specific configuration required for `LocalCache` and `TTLCache`.

For Redis cache, configure Redis middleware:

```yaml
# conf.yml
redis:
  enabled: true
  host: localhost
  port: 6379
  db: 0
  password: ""
```

See [Redis middleware](https://github.com/CurtisNewbie/miso/blob/main/doc/rabbitmq.md) for full configuration.