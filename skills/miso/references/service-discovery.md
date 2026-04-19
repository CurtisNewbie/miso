# Service Discovery

Service discovery enables services to find and communicate with each other without hardcoding network addresses. Miso supports Nacos and Consul for service registration and discovery.

## Overview

When service discovery is enabled:
- Your service registers itself with the service registry on startup
- Services can discover each other using service names instead of IP addresses
- The HTTP client automatically resolves service names to actual endpoints
- Service instances are monitored for health changes

## Supported Registries

### Nacos

Nacos provides both service discovery and configuration center capabilities.

**Configuration:** See [Nacos Configuration](../../../doc/config.md#nacos-configuration) for all available properties.

### Consul

Consul provides service discovery, health checking, and KV store.

**Configuration:** See [Consul Configuration](../../../doc/config.md#consul-configuration) for all available properties.

### Usage

Both Nacos and Consul use the same API for service discovery. There are two ways to make requests with service discovery:

**Method 1: Using NewDynClient (recommended for dynamic service discovery)**

```go
// Create client with service discovery
// Second parameter is the relative URL path
// Third parameter is the service name to resolve
var res ApiResponse
err := miso.NewDynClient(rail, "/api/users", "target-service-name").
	Get().
	Json(&res)
```

**Method 2: Using "lb:" prefix with NewClient**

```go
// Use "lb:" prefix to indicate service discovery
// Format: lb:SERVICE_NAME/path
var res ApiResponse
err := miso.NewClient(rail, "lb:target-service-name/api/users").
	Get().
	Json(&res)

// Equivalent to using EnableServiceDiscovery:
// miso.NewClient(rail, "/api/users").
//     EnableServiceDiscovery("target-service-name").
//     Get()
```

The "lb:" prefix automatically enables service discovery and extracts the service name from the URL, making it more concise.

## Service Registration

Both Nacos and Consul automatically register your service when:

1. Service discovery is enabled (`enabled: true`)
2. Bootstrap phase completes (L4)
3. Application is ready

The registration includes:
- Service name (configured via `register_name`)
- Service address and port
- Health check endpoint
- Metadata (custom key-value pairs)

## Service Discovery Flow

```go
// 1. Service registration happens automatically during bootstrap
// No code needed if configuration is set correctly

// 2a. Discover and call another service using NewDynClient
// Second parameter is the relative URL path
// Third parameter is the service name to resolve
var res ApiResponse
err := miso.NewDynClient(rail, "/api/users", "other-service").
	Get().
	Json(&res)

// 2b. Alternatively, use "lb:" prefix with NewClient
// Format: lb:SERVICE_NAME/path
err = miso.NewClient(rail, "lb:other-service/api/users").
	Get().
	Json(&res)

// 3. The client resolves "other-service" to available endpoints
//    - Queries the service registry (Nacos/Consul)
//    - Filters healthy instances
//    - Selects an instance (load balancing)
//    - Makes the HTTP request

// 4. Instance changes are automatically tracked
//    - New instances are discovered
//    - Failed instances are removed
//    - Changes trigger server change listeners
```

## Health Checks

### Nacos
- Nacos performs active health checks on registered instances
- Unhealthy instances are excluded from service discovery
- Check configuration via Nacos dashboard

### Consul
- Consul performs periodic health checks on registered instances
- Health check is configurable via `server.health-check-url`
- Check interval and timeout are configurable
- Failed instances are deregistered after a timeout

## Manual Deregistration

Both registries support manual deregistration endpoints for maintenance scenarios:

**Nacos:** Configure via `nacos.discovery.enable-deregister-url` and `nacos.discovery.deregister-url`

**Consul:** Configure via `consul.enable-deregister-url` and `consul.deregister-url`

## Service Metadata

You can attach custom metadata to your service registration:

```yaml
# Nacos
nacos:
  discovery:
    metadata:
      version: "1.0.0"
      environment: "production"

# Consul
consul:
  metadata:
    version: "1.0.0"
    environment: "production"
```

Metadata is automatically included with:
- Registration time (`service.register_time`)
- Service ID (Consul)
- IP address and port

## Advanced Usage

### Watch Service Changes

```go
// Register a listener for service instance changes
miso.SubscribeServerChanges(rail, "service-name", func() {
    rail.Infof("Service instances changed")
    // Update caches, notify other components, etc.
})
```

### Fetch Service Instances Directly

```go
// Get server list and query for a specific service
serverList := miso.GetServerList()
servers := serverList.ListServers(rail, "service-name")

// Select a random server
server, err := miso.SelectAnyServer(rail, "service-name")
```

### Service Subscription

```go
// Subscribe to a service (automatically done when using HTTP client)
serverList := miso.GetServerList()
err := serverList.Subscribe(rail, "service-name")
```

## Best Practices

1. **Use descriptive service names** - Choose clear, consistent names across services
2. **Configure health checks** - Ensure health endpoints return accurate status
3. **Set appropriate timeouts** - Balance responsiveness with stability
4. **Monitor service health** - Use registry dashboards to track instance status
5. **Handle service unavailability** - Implement fallback logic for when services are down
6. **Use metadata for versioning** - Track service versions in metadata for A/B testing
7. **Test deregistration** - Verify graceful shutdown works correctly

## Reference Files

- `middleware/nacos/nacos.go` - Nacos implementation
- `miso/consul.go` - Consul implementation
- `miso/client.go` - HTTP client with service discovery
- `miso/proxy.go` - Service-based proxy resolution
- `doc/config.md` - Complete configuration reference