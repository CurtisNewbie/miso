# pprof, trace and gops

## pprof

If configured, miso exposes pprof http endpoints for debugging performance. The http endpoints for pprof are only enabled on non-prod mode. You can enable it in production mode as follows:

```yaml
server:
  pprof:
    enabled: true
```

The pprof is exposed at endpoint `/debug/pprof/**`, it's not customizable at the moment.

You can enable authentication for the endpoints if necessary, only Bearer authentication is supported:

```yaml
server:
  pprof:
    enabled: true
    auth:
      bearer: "your_secret_token"
```

Once the server is up and running, use pprof tool to connect to the exposed endpoint. E.g.,

for heap:

```sh
go tool pprof -http=: http://localhost:8080/debug/pprof/heap?seconds=30
```

for cpu profile:

```sh
go tool pprof -http=: http://localhost:8080/debug/pprof/profile?seconds=30
```

for goroutine blocking profile:

```sh
go tool pprof -http=: http://localhost:8080/debug/pprof/block?seconds=30
```

for goroutine's stacktrace:

```sh
curl http://localhost:8080/debug/pprof/goroutine?debug=2 -o stacktrace.txt
```

More about the go tool `pprof`:

- https://github.com/google/pprof/blob/main/doc/README.md.
- https://go.dev/blog/pprof

## trace

Same as pprof, if enabled, miso exposes http endpoints to capture trace info using FlightRecorder. These http endpoints are only enabled on non-prod mode. You can enable it in production mode as follows (same as pprof):

```yaml
server:
  pprof:
    enabled: true
```

Trace FlightRecorder can be controlled using following apis.

- `GET /debug/trace/recorder/run`

  - Description: Start FlightRecorder. Recorded result is written to trace.out when it's finished or stopped.
  - Query Parameter:
    - "duration": Duration of the flight recording. Required. Duration cannot exceed 30 min.
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/debug/trace/recorder/run?duration='
    ```

- `GET /debug/trace/recorder/stop`
  - Description: Stop existing FlightRecorder session.
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/debug/trace/recorder/stop'
    ```

Again, you can enable authentication for these apis using the same bearer token for pprof apis, e.g.,

```yaml
server:
  pprof:
    enabled: true
    auth:
      bearer: "your_secret_token"
```

For example:

```sh
# start flight recorder for 30s
curl -X GET 'http://localhost:8080/debug/trace/recorder/run?duration=30s' -v

# wait for 30s then open the trace.out file
go tool trace trace.out
```

## gops

When miso bootstraps, miso always creates a local `gops` agent (https://github.com/google/gops). You can install the gops tool to collect debugging info locally, e.g.,

```sh
$ gops
# 8446  8430  main * go1.24.7 /Users/photon/Library/Caches/go-build/a6/a6f5e5027c041588ea8398dabb0f0a4ce018890aff615c68986c6f5076f859fa-d/main

$ gops stack 8446
# ...
# stack snapshot ...
# ...
```
