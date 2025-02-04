# pprof

If configured, miso exposes pprof http endpoints for debugging performance. The http endpoints for pprof are only enabled on non-prod mode. You can enable it in production mode as follows:

```yaml
server:
  pprof.enabled: true
```

The pprof is exposed at endpoint '/debug/pprof', it's not customizable at the moment.

Once the server is up and running, use pprof tool to connect to the exposed endpoint:

```sh
go tool pprof -http=: http://localhost:8080/debug/pprof/heap
```

More about the go tool `pprof`: https://github.com/google/pprof/blob/main/doc/README.md.