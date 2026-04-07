# Configuration

Configuration management in miso using Viper with property constants and multiple sources.

## Complete Configuration Reference

For the complete list of all configuration properties, defaults, and descriptions, see **[doc/config.md](../../doc/config.md)**. This file is generated automatically from code and serves as the single source of truth for configuration.

## Property Constants

Define configuration properties as constants for type safety and documentation:

```go
package myapp

import "github.com/curtisnewbie/miso"

const (
    PropAppName      = "app.name"
    PropServerPort   = "server.port"
    PropDBEnabled    = "db.enabled"
    PropDBHost       = "db.host"
    PropDBPort       = "db.port"
    PropDBUser       = "db.user"
    PropDBPassword   = "db.password"
    PropDBName       = "db.name"
    PropRedisEnabled = "redis.enabled"
    PropRedisAddr    = "redis.addr"
)
```

## Accessing Configuration

```go
// String values
appName := miso.GetPropStr(PropAppName)
dbHost := miso.GetPropStr(PropDBHost)

// Integer values
port := miso.GetPropInt(PropServerPort)
dbPort := miso.GetPropInt(PropDBPort)

// Boolean values
enabled := miso.GetPropBool(PropRedisEnabled)
debugMode := miso.GetPropBool("debug.enabled")

// Float values
timeout := miso.GetPropFloat("request.timeout")

// Duration values
readTimeout := miso.GetPropDuration("server.read-timeout")
```

## Setting Configuration

```go
// Set value programmatically
miso.SetProp(PropAppName, "my-application")
miso.SetProp(PropServerPort, 8080)

// Set default value (only if not already set)
miso.SetDefProp(PropServerPort, 8080)
miso.SetDefProp(PropAppName, "default-app")
```

## Checking Configuration

```go
// Check if property exists
if miso.HasProp(PropServerPort) {
    port := miso.GetPropInt(PropServerPort)
}

// Check if property is empty
if miso.GetPropStr(PropAppName) == "" {
    // handle missing configuration
}
```

## Configuration Sources

Configuration is loaded from multiple sources in priority order:

1. **Default values** (`SetDefProp()`)
2. **Configuration file** (`conf.yml`)
3. **Environment variables** (`APP_NAME`, `SERVER_PORT`, etc.)
4. **Command-line arguments**
5. **Programmatic overrides** (`SetProp()`)

## Configuration File (conf.yml)

```yaml
app:
  name: my-application
  version: 1.0.0

server:
  port: 8080
  read-timeout: 30s
  write-timeout: 30s

db:
  host: localhost
  port: 3306
  user: root
  password: secret
  database: mydb

redis:
  enabled: true
  addr: localhost:6379
  password: ""

logging:
  level: info
  file: logs/app.log
```

## Environment Variables

Environment variables override configuration file values:

```bash
# Environment variables use uppercase and underscore
export APP_NAME=my-app
export SERVER_PORT=8080
export DB_HOST=localhost
export DB_USER=root
export REDIS_ENABLED=true
```

## Configuration in Bootstrap

Access configuration in bootstrap callbacks to conditionally initialize components:

```go
func redisBootstrap(rail flow.Rail) error {
    if !miso.GetPropBool(PropRedisEnabled) {
        rail.Infof("Redis disabled")
        return nil
    }

    addr := miso.GetPropStr(PropRedisAddr)
    db := miso.GetPropInt(PropRedisDB)

    // Initialize Redis client with config
    // ...
    rail.Infof("Redis connected: %s (db: %d)", addr, db)
    return nil
}
```

**Note:** Database connections are automatically managed by middleware packages (mysql, sqlite) via YAML configuration. See **[database.md](database.md)** for details.

## Conditional Configuration

Use configuration properties to conditionally enable/disable features:

```go
func init() {
    miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
        Name:      "Initialize Redis",
        Bootstrap: redisBootstrap,
        Condition: redisCondition,
        Order:     miso.BootstrapOrderL1,
    })
}

func redisCondition(rail flow.Rail) (bool, error) {
    return miso.GetPropBool(PropRedisEnabled), nil
}

func redisBootstrap(rail flow.Rail) error {
    addr := miso.GetPropStr(PropRedisAddr)
    // Redis initialization
    return nil
}
```

## Configuration Best Practices

1. **Define constants** for all configuration properties at package level
2. **Use meaningful names** with dot notation for nested properties
3. **Set defaults** for optional configuration values
4. **Validate configuration** in bootstrap phase
5. **Document properties** using comments or code tags

## Configuration with misoconfig CLI

miso framework provides `misoconfig` CLI command to automatically generate configuration documentation and handle default values.

### Using misoconfig Command

```bash
# Generate configuration table to markdown file
misoconfig -path doc/config.md

# Enable debug logging
misoconfig -debug
```

If misoconfig is not installed, install it using:
```bash
go install github.com/curtisnewbie/miso/cmd/misoconfig@latest
```

### Defining Configuration Properties with Tags

Use `misoconfig-*` comments in your code to document configuration properties and their defaults:

```go
// misoconfig-section: Database Configuration

// misoconfig-prop: Database server hostname
const PropDBHost = "db.host"

// misoconfig-prop: Database server port | 3306
const PropDBPort = "db.port"

// misoconfig-prop: Database user
const PropDBUser = "db.user"

// misoconfig-prop: Database password
const PropDBPassword = "db.password"

// misoconfig-default-start
miso.SetDefProp(PropDBPort, 3306)
miso.SetDefProp(PropDBHost, "localhost")
// misoconfig-default-end
```

### Supported Tags

- `// misoconfig-section: <name>` - Start a configuration section
- `// misoconfig-prop: <description> | <default-value>` - Document a configuration property
- `// misoconfig-alias: <old-name> | <version>` - Mark as alias for deprecated property
- `// misoconfig-doc-only` - Property shown only in documentation
- `// misoconfig-default-start` - Start default value block
- `// misoconfig-default-end` - End default value block

### Generated Documentation

The misoconfig command generates markdown tables between these markers in your documentation file:

```markdown
<!-- misoconfig-table-start -->
<!-- misoconfig-table-end -->
```

Example output:
```markdown
| Property | Description | Default |
|----------|-------------|---------|
| `db.host` | Database server hostname | localhost |
| `db.port` | Database server port | 3306 |
```

### Example: Custom Configuration Module

```go
package myapp

import "github.com/curtisnewbie/miso"

// misoconfig-section: My Application Configuration

// misoconfig-prop: API endpoint URL
const PropApiEndpoint = "api.endpoint"

// misoconfig-prop: Request timeout | 30s
const PropApiTimeout = "api.timeout"

// misoconfig-prop: Retry attempts | 3
const PropApiRetryAttempts = "api.retry.attempts"

// misoconfig-default-start
func init() {
    miso.SetDefProp(PropApiTimeout, 30*time.Second)
    miso.SetDefProp(PropApiRetryAttempts, 3)
}
// misoconfig-default-end
```

Generate documentation:
```bash
go run ./cmd/misoconfig -path doc/config.md
```

## Configuration Tags (for code generation)

```go
// misoconfig-section: Database Configuration

// misoconfig-prop: db.host - Database server hostname
const PropDBHost = "db.host"

// misoconfig-prop: db.port - Database server port (default: 3306)
const PropDBPort = "db.port"

// misoconfig-default-start
miso.SetDefProp(PropDBPort, 3306)
// misoconfig-default-end
```

## Accessing AppConfig

For advanced configuration management:

```go
import "github.com/curtisnewbie/miso"

// Get global AppConfig
app := miso.App()

// Access configuration methods
app.SetProp(PropAppName, "new-name")
val := app.GetProp(PropAppName)
```

## Type Conversion

Viper automatically handles type conversion:

```go
import "github.com/curtisnewbie/miso/errs"

// String to int
port := miso.GetPropInt("server.port")  // "8080" -> 8080

// String to bool
enabled := miso.GetPropBool("feature.enabled")  // "true" -> true

// String to duration
timeout := miso.GetPropDuration("server.timeout")  // "30s" -> 30s
```

Invalid type conversions return zero values without errors. Validate configuration explicitly:

```go
port := miso.GetPropInt(PropServerPort)
if port == 0 {
    return errs.NewErrf("Invalid server port: %d", port)
}
```