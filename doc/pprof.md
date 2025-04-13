# pprof

If configured, miso exposes pprof http endpoints for debugging performance. The http endpoints for pprof are only enabled on non-prod mode. You can enable it in production mode as follows:

```yaml
server:
  pprof:
    enabled: true
```

The pprof is exposed at endpoint '/debug/pprof', it's not customizable at the moment.

You can enable authentication for the endpoints if necessary, only Bearer authentication is supported:

```yaml
server:
  pprof:
    enabled: true
    auth:
      enabled: true
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