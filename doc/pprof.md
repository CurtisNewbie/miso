# pprof

If configured, miso exposes pprof http endpoints for debugging performance. Http endpoints for pprof are by default disabled. You can enable it as follows:

```yaml
server:
  pprof.enabled: true
```

The pprof is exposed at endpoint '/debug/pprof', it's not customizable at the moment.

Once the server is up and running, use pprof tool to connect to the exposed endpoint:

```sh
go tool pprof -http=:8081 http://localhost:8080/debug/pprof/heap
```