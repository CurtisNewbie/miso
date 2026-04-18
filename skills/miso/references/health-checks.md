# Health Checks

Health check system for monitoring service component status and exposing health endpoints.

## Overview

Miso provides a health check framework that allows you to register custom health indicators for your service components. The health check endpoint is automatically called by service discovery tools (like Consul) to monitor service availability.

## Health Indicators

### Registering Health Indicators

```go
import "github.com/curtisnewbie/miso"

func init() {
    miso.AddHealthIndicator(miso.HealthIndicator{
        Name: "MySQL Database",
        CheckHealth: func(rail miso.Rail) bool {
            db := dbquery.GetDB()
            if err := db.DB().Ping(); err != nil {
                rail.Errorf("MySQL health check failed: %v", err)
                return false
            }
            return true
        },
    })

    miso.AddHealthIndicator(miso.HealthIndicator{
        Name: "Redis Connection",
        CheckHealth: func(rail miso.Rail) bool {
            redis := redis.GetRedis()
            if err := redis.Ping(rail.Context()).Err(); err != nil {
                rail.Errorf("Redis health check failed: %v", err)
                return false
            }
            return true
        },
    })

    miso.AddHealthIndicator(miso.HealthIndicator{
        Name: "External API",
        CheckHealth: func(rail miso.Rail) bool {
            // Check external service connectivity
            resp, err := http.Get("https://api.example.com/health")
            if err != nil {
                return false
            }
            defer resp.Body.Close()
            return resp.StatusCode == 200
        },
    })
}
```

### Running Health Checks

```go
import "github.com/curtisnewbie/miso"

// Run all health checks
rail := miso.EmptyRail()
healthStatus := miso.CheckHealth(rail)

for _, status := range healthStatus {
    if !status.Healthy {
        rail.Warnf("Component '%s' is unhealthy", status.Name)
    } else {
        rail.Infof("Component '%s' is healthy", status.Name)
    }
}

// Check if all health checks pass
if miso.IsHealthcheckPass(rail) {
    rail.Infof("All components are healthy")
} else {
    rail.Errorf("Some components are unhealthy")
}
```

## Default Health Check Endpoint

Miso automatically registers a default health check endpoint that:
1. Runs all registered health indicators
2. Returns 200 (OK) if all indicators are healthy
3. Returns 503 (Service Unavailable) if any indicator is unhealthy

### Health Check Response

**Success (200 OK):**
```
UP
```

**Failure (503 Service Unavailable):**
```
DOWN
```

### Customizing Health Check URL

Configure health check endpoint URL in `conf.yml`:

```yaml
# conf.yml
server:
  health-check-url: "/actuator/health"
```

### Disabling Default Health Check Handler

If you're using a proxy or custom handler, you can disable the default:

```go
func init() {
    miso.DisableDefaultHealthCheckHandler()
}
```

## Health Check with Proxy

When using `HttpProxy` with root path (`/`), health check is handled by a filter:

```go
import "github.com/curtisnewbie/miso"

proxy := miso.NewHttpProxy("/", targetResolver)
proxy.AddHealthcheckFilter()
```

This filter:
- Checks health at the configured URL
- Returns 200 if healthy, 503 if unhealthy
- Rate-limits to once per second

## Best Practices

### 1. Component-Specific Health Checks

Register health indicators for each critical component:

```go
// Database
miso.AddHealthIndicator(miso.HealthIndicator{
    Name: "MySQL",
    CheckHealth: checkMySQLHealth,
})

// Cache
miso.AddHealthIndicator(miso.HealthIndicator{
    Name: "Redis",
    CheckHealth: checkRedisHealth,
})

// Message Queue
miso.AddHealthIndicator(miso.HealthIndicator{
    Name: "RabbitMQ",
    CheckHealth: checkRabbitMQHealth,
})

// External Services
miso.AddHealthIndicator(miso.HealthIndicator{
    Name: "Payment Gateway",
    CheckHealth: checkPaymentGatewayHealth,
})
```

## Health Status Constants

```go
import "github.com/curtisnewbie/miso"

const (
    miso.ServiceStatusUp   = "UP"
    miso.ServiceStatusDown = "DOWN"
)
```

## Types

### HealthIndicator

```go
type HealthIndicator struct {
    Name        string               // Name of the health indicator
    CheckHealth func(rail Rail) bool // Function that checks health status
}
```

### HealthStatus

```go
type HealthStatus struct {
    Name    string // Component name
    Healthy bool   // Health status (true = healthy)
}
```

## Configuration

Health check behavior is controlled by configuration properties:

```yaml
# conf.yml
server:
  health-check-url: "/actuator/health"  # Default: "/"
```

See [config.md](https://github.com/CurtisNewbie/miso/blob/main/doc/config.md) for full configuration reference.