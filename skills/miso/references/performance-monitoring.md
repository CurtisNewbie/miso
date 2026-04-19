# Performance Monitoring

Performance profiling and debugging tools including pprof, FlightRecorder, and gops.

## Overview

Miso provides built-in performance monitoring tools for debugging production issues, analyzing performance bottlenecks, and profiling Go applications. These tools include pprof for CPU/memory profiling, FlightRecorder for execution traces, and gops for runtime inspection.

## pprof (CPU/Memory Profiling)

### Enabling pprof

pprof endpoints are automatically enabled in non-production mode. Enable in production:

```yaml
# conf.yml
server:
  pprof:
    enabled: true
```

### pprof Authentication

Protect pprof endpoints with Bearer token authentication:

```yaml
server:
  pprof:
    enabled: true
    auth:
      bearer: "your_secret_token"
```

### pprof Endpoints

pprof is exposed at `/debug/pprof/**`:

| Endpoint | Description |
|----------|-------------|
| `/debug/pprof/` | pprof index page |
| `/debug/pprof/heap` | Heap profile |
| `/debug/pprof/profile` | CPU profile |
| `/debug/pprof/goroutine` | Goroutine profile |
| `/debug/pprof/block` | Blocking profile |
| `/debug/pprof/mutex` | Mutex profile |
| `/debug/pprof/trace` | Execution trace |

### Using pprof Tool

#### Heap Profile

```bash
# Capture heap profile for 30 seconds
go tool pprof -http=: http://localhost:8080/debug/pprof/heap?seconds=30
```

#### CPU Profile

```bash
# Capture CPU profile for 30 seconds
go tool pprof -http=: http://localhost:8080/debug/pprof/profile?seconds=30
```

#### Goroutine Profile

```bash
# Capture goroutine blocking profile
go tool pprof -http=: http://localhost:8080/debug/pprof/block?seconds=30

# Dump goroutine stack traces
curl http://localhost:8080/debug/pprof/goroutine?debug=2 -o stacktrace.txt
```

### Common pprof Commands

```bash
# Interactive pprof session
go tool pprof http://localhost:8080/debug/pprof/heap

# List top 10 functions
(pprof) top10

# View graph
(pprof) web

# View source
(pprof) list <function_name>

# Save profile
(pprof) save profile.pb.gz
```

### Authentication with pprof

```bash
# With Bearer token
curl -H "Authorization: Bearer your_token" http://localhost:8080/debug/pprof/heap
```

## FlightRecorder (Execution Tracing)

### Enabling FlightRecorder

FlightRecorder is automatically enabled in non-production mode:

```yaml
# conf.yml
server:
  pprof:
    enabled: true
```

### FlightRecorder Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/debug/trace/recorder/run` | GET | Start flight recorder |
| `/debug/trace/recorder/stop` | GET | Stop flight recorder |
| `/debug/trace/recorder/snapshot` | GET | Take trace snapshot |

### Starting FlightRecorder

```bash
# Start flight recorder for 30 seconds
curl -X GET 'http://localhost:8080/debug/trace/recorder/run?duration=30s' -v
```

**Response:**
```
Flight recorder started, duration: 30s, will write to: trace.out
```

### Stopping FlightRecorder

```bash
# Stop flight recorder early
curl -X GET 'http://localhost:8080/debug/trace/recorder/stop' -v
```

**Response:**
```
Flight recorder stopped, wrote trace to: trace.out
```

### Analyzing Trace

```bash
# Open trace in browser
go tool trace trace.out

# View trace statistics
go tool trace -http=:6060 trace.out
```

### Taking Snapshot

```bash
# Take immediate snapshot
curl -X GET 'http://localhost:8080/debug/trace/recorder/snapshot' -v
```

### FlightRecorder Parameters

- `duration` (required): Recording duration, maximum 30 minutes
  - Format: `30s`, `5m`, `1h`
  - Example: `?duration=30s`

### Authentication

FlightRecorder uses same authentication as pprof:

```bash
curl -H "Authorization: Bearer your_token" \
  -X GET 'http://localhost:8080/debug/trace/recorder/run?duration=30s'
```

## gops (Runtime Inspection)

### Using gops

gops agent is automatically started when miso bootstraps:

```bash
# List all Go processes
$ gops

8446  8430  main * go1.24.7 /path/to/myapp
```

### Inspecting Process

```bash
# View process details
$ gops 8446

parent PID: 8430
threads: 12
memory usage: 123.45 MB
cpu usage: 2.3%
username: user
...

# Dump stack traces
$ gops stack 8446

goroutine 1 [running]:
main.main()
    /path/to/main.go:42 +0x123
...

# View memory stats
$ gops memstats 8446

alloc: 12345678 bytes
total alloc: 98765432 bytes
sys: 23456789 bytes
...

# View GC stats
$ gops gc 8446

# View version info
$ gops version 8446

# Send SIGINT (graceful shutdown)
$ gops stop 8446
```

### Installing gops

```bash
go install github.com/google/gops@latest
```

## Configuration

### pprof Configuration

```yaml
# conf.yml
server:
  pprof:
    enabled: true          # Enable pprof endpoints
    auth:
      bearer: "token"      # Bearer token for authentication
```

### Default Behavior

- **Non-production mode**: pprof and FlightRecorder enabled by default (no auth)
- **Production mode**: Disabled by default (must explicitly enable with auth)

## Performance Monitoring with Proxy

When using `HttpProxy`, pprof and FlightRecorder endpoints need a filter:

```go
import "github.com/curtisnewbie/miso"

proxy := miso.NewHttpProxy("/", targetResolver)

// Add debug filter with authentication
err := proxy.AddDebugFilter(true)
if err != nil {
    panic(err)
}
```

The debug filter:
- Handles `/debug/pprof/**` requests
- Handles `/debug/trace/**` requests
- Enforces Bearer authentication (if configured)

## Best Practices

### 1. Enable pprof in Staging

```yaml
# staging/conf.yml
server:
  pprof:
    enabled: true
```

### 2. Protect pprof in Production

```yaml
# production/conf.yml
server:
  pprof:
    enabled: true
    auth:
      bearer: "${PPROF_BEARER_TOKEN}"  # Load from environment
```

### 3. Regular Profiling

```bash
# Weekly heap profile
0 0 * * 0 curl http://localhost:8080/debug/pprof/heap?seconds=60 -o heap_$(date +%Y%m%d).pb.gz

# Weekly CPU profile
0 0 * * 1 curl http://localhost:8080/debug/pprof/profile?seconds=60 -o cpu_$(date +%Y%m%d).pb.gz
```

### 4. Monitor Goroutine Leaks

```bash
# Check goroutine count
curl http://localhost:8080/debug/pprof/goroutine?debug=1

# Set up alert if goroutine count exceeds threshold
```

### 5. Capture Traces for Issues

```bash
# Start trace when issue is reported
curl -X GET 'http://localhost:8080/debug/trace/recorder/run?duration=60s'

# Reproduce the issue

# Stop trace and analyze
curl -X GET 'http://localhost:8080/debug/trace/recorder/stop'
go tool trace trace.out
```

## Memory Statistics

Miso provides built-in memory statistics collection:

```go
import "github.com/curtisnewbie/miso"

// Collect CPU stats
cpuStats, err := miso.CollectCpuStats()
if err != nil {
    rail.Warnf("Failed to collect CPU stats: %v", err)
}

rail.Infof("CPU Usage: %.2f%%, Goroutines: %d",
    cpuStats.CpuUsage, cpuStats.Goroutines)
```

## Performance Monitoring Checklist

- [ ] Enable pprof in staging environment
- [ ] Protect pprof with authentication in production
- [ ] Set up regular heap profiling (weekly)
- [ ] Set up regular CPU profiling (weekly)
- [ ] Monitor goroutine count for leaks
- [ ] Use FlightRecorder for investigating performance issues
- [ ] Use gops for runtime inspection
- [ ] Set up alerts for memory/CPU usage
- [ ] Capture traces before investigating complex issues

## Resources

- [Go pprof documentation](https://github.com/google/pprof/blob/main/doc/README.md)
- [Go blog: pprof](https://go.dev/blog/pprof)
- [gops documentation](https://github.com/google/gops)
- [Go trace tool](https://pkg.go.dev/cmd/internal/trace)