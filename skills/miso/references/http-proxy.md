# HTTP Reverse Proxy (HttpProxy)

HttpProxy is a reverse proxy built on Go's `httputil.ReverseProxy`, mounted on Gin routes, with a filter pipeline for access control, logging, metrics, and debugging. It is the building block for gateway-style services (see `demo/proxydemo`).

## Overview

- Proxy requests for a path prefix (e.g. `/proxy/**`) or the entire server (root path `/`) to a target resolved per request.
- Filters run before the request is proxied; they can allow, reject (with a custom status code), or handle the request locally.
- Access control supports static whitelist paths, static/bearer/basic auth credentials, and remote auth delegated to a downstream auth service — all dynamically loaded from configuration.
- Trace headers are propagated to the proxied server, with the inbound trace headers stripped and replaced (security, see [Trace Propagation](#trace-propagation)).
- Default client: 5s connect timeout, 30s response header timeout, IdleConnTimeout 1min, MaxIdleConnsPerHost 100, MaxConnsPerHost 500.

## Quick Start

```go
// proxy all requests to /proxy* to server localhost:8081
_ = miso.NewHttpProxy("/proxy", func(rail miso.Rail, proxyPath string) (string, error) {
	return "http://localhost:8081" + proxyPath, nil
})

miso.BootstrapServer(os.Args)
```

`NewHttpProxy(proxiedPath string, targetResolver ProxyTargetResolver)` must be called before the server bootstraps. The resolver receives the current `Rail` and the proxy path (empty string for the root path request, otherwise guaranteed to contain a leading slash) and returns the target URL to proxy to.

### Root Path `/`

If `proxiedPath` is `/`, the proxy takes over the whole server:

- Default health check handler, Prometheus bootstrap, and pprof endpoint registration are disabled to avoid path conflicts
- `server.trace.inbound.propagate` is defaulted to `false` — the proxy is the entry point, it must not trust inbound trace headers
- `AddHealthcheckFilter`, `AddMetricsFilter`, and `AddDebugFilter` are only active when the proxied path is `/`

### Target Resolution Errors

Return a `ProxyHttpStatusError` (e.g. `miso.GatewayError{StatusCode: 404}`) from the resolver to respond with a specific status code (a `Status()` of 0 falls back to `404`). Any other error responds `404`.

### Service Discovery Based Resolution

`NewDynProxyTargetResolver()` resolves targets from the service registry: the proxy path must have the form `/service-name/relative/path`, the service name is resolved via service discovery, and the rest is the backend relative path.

```go
_ = miso.NewHttpProxy("/", miso.NewDynProxyTargetResolver())
```

## ProxyContext

Each proxied request gets a `ProxyContext` passed through the filter chain:

```go
type ProxyContext struct {
	Rail      *Rail
	Inb       *Inbound
	ProxyPath string // Proxied path without query parameters
}
```

`ProxyContext` supports attribute storage for cross-filter communication:

```go
pc.SetAttr("user", user)
if v, ok := pc.GetAttr("user"); ok { ... }
pc.DelAttr("user")
```

## Filter Pipeline

Filters run in registration order; each filter calls `next()` to continue the chain. The last link in the chain performs the actual proxying.

```go
type ProxyFilter = func(pc *ProxyContext, next func())

func (h *HttpProxy) AddFilter(f ProxyFilter) {
	h.filters = append(h.filters, f)
}
```

To reject a request inside a filter, write the status code directly:

```go
h.AddFilter(func(pc *miso.ProxyContext, next func()) {
	if blocked(pc.ProxyPath) {
		pc.Inb.Writer().WriteHeader(http.StatusForbidden)
		return // don't call next()
	}
	next()
})
```

## Access Filters

### AddAccessFilter

`AddAccessFilter(whitelistPatterns func() []string, checkAuth func(pc *ProxyContext) (statusCode int, ok bool))`:

1. If the proxy path matches any whitelist pattern → allow.
2. Otherwise call `checkAuth`; if it returns `ok == true` → allow.
3. Otherwise respond with the returned status code (default `401`), and log a warning including the request body when the content type is loggable.

```go
h.AddAccessFilter(func() []string {
	return []string{"/health", "/open/api/**"}
}, func(pc *miso.ProxyContext) (int, bool) {
	if pc.Inb.Header("Authorization") == "Bearer xxx" {
		return 0, true
	}
	return http.StatusUnauthorized, false
})
```

### JoinCheckAuth

Combine multiple auth checks: the first check that returns `ok == true` wins; if all fail, the last status code is returned.

```go
checkAuth := h.JoinCheckAuth(checkStaticAuth, checkRemoteAuth)
```

### AddConfDynAccessFilter (config-driven dynamic access filter)

`AddConfDynAccessFilter(propRootKey string, refreshEvery time.Duration)` registers a full access filter — whitelist + auth routes — both loaded from configuration under a single prop root key and unmarshalled into `DynAccessFilterConfig`:

```go
type DynAccessFilterConfig struct {
	Whitelist  []string        // path patterns allowed without authentication
	AuthRoutes []DynAuthRoute  // auth routes, see below
}
```

Example configuration:

```yaml
proxy:
  access:
    filter:
      whitelist:
        - "/health"
        - "/open/api/**"
      auth-routes:
        - name: "myauth1"        # type: "bearer"
          type: "bearer"
          bearer: "mybearer1"
          trace:
            username: "trace-user"
            role: "myrole"
          path-patterns:
            - "/path1"
            - "/path2"
        - name: "myauth2"        # type: "basic"
          type: "basic"
          username: "myuser"
          password: "mypassword"
          trace:
            username: "trace-user"
            role: "myrole"
          path-patterns:
            - "/path4"
            - "/path5"
        - name: "myauth3"        # type: "remote"
          type: "remote"
          path-patterns:         # may be omitted; remote routes without path-patterns match all paths
            - "/path7"
          remote:
            path: "lb://auth-service/open/api/auth/check"  # lb:// resolves via service discovery
            body-map:
              authorization: "token"
              path: "url"
              method: "method"
            decision-field: "data.valid"   # dotted path into the response body (plain JSON responses work too, e.g. "valid")
            user:
              userno: "data.userno"
              username: "data.username"
              roleno: "data.roleno"
              role: "data.role"
```

```go
var h *miso.HttpProxy
h.AddConfDynAccessFilter("proxy.access.filter", 5*time.Second)
```

**Refresh semantics:** whitelist and auth routes are loaded together, cached, and refreshed in the background every `refreshEvery`, so frequent requests don't read configuration every time, and values are never outdated for longer than the refresh interval. If `refreshEvery <= 0`, both are loaded only once.

**Route validation at load time** (invalid routes are dropped):

- Bearer and basic routes must define at least one path pattern; remote routes may omit them (match all paths)
- `type` may be omitted: inferred as `bearer` when `bearer` is set, otherwise `basic`
- `basic` routes require both `username` and `password`; `bearer` routes require `bearer`; `remote` routes require `remote.path` and `remote.decision-field`; unknown types are dropped

**Evaluation order** (per request, handled by `WithDynAuthCheck`):

1. All bearer/basic routes are evaluated first, in config order: the request's `Authorization` header must match the route's credentials AND the proxy path must match one of its path patterns
2. On success, the route's `trace.username` / `trace.role` (if non-empty) are stored in the Rail and propagated downstream
3. Remote routes are evaluated after, in config order; the first matching route's decision is final
4. No match → `401`

### Remote Auth Routes

`type: "remote"` delegates authentication AND authorization to a downstream auth service in a single call:

- The auth request is a `POST` JSON to `remote.path`; the `lb://service-name/...` prefix (also `lb:service-name/...`) resolves the address via service discovery, a full URL calls it directly. Tracing is enabled and 2xx responses are required.
- The request body is built from `body-map`: `authorization` sends the raw `Authorization` header (no scheme parsing), `path` the proxy path, `method` the HTTP method. Empty value means the source is not sent.
- The response does **not** have to be GnResp-wrapped — any JSON object works. Dotted paths (`decision-field`, `user.*`) resolve against the full response body: for GnResp-wrapped responses (`{error, errorCode, msg, data}`) include the `data.` prefix (e.g. `data.valid`), plain responses use paths like `valid`.
- On allow, user info mapped via `user` (dotted paths) is stored in the trace with `flow.StoreUser` and propagated downstream. Note: `StoreUser` is applied unconditionally — when the response carries no user info, the trace user fields are overwritten with empty values and no user headers are emitted downstream.

Status code mapping for remote auth:

| Condition | Status |
|---|---|
| Auth request failed (transport/HTTP error) | 503 |
| Decision field missing in response (warning logged with the response body) | 502 |
| Decision `false`, user info present in response | 403 |
| Decision `false`, no user info | 401 |
| Missing `Authorization` header | 401 |
| Decision `true` | allow |

### AddWsAccessFilter

`AddWsAccessFilter(loadConfigs func() []WsAccessFilterConfig, checkAuth func(token string, pc *ProxyContext) (statusCode int, ok bool))` guards WebSocket upgrade requests with a token passed as a query parameter (e.g. a JWT, validated by `checkAuth`):

```go
type WsAccessFilterConfig struct {
	PathPatterns []string `mapstructure:"path-patterns"` // empty = match all paths
	TokenQueryKey string  `mapstructure:"token-query-key"` // e.g. "token" reads from ?token=xxx
}
```

- Only requests with `Connection: upgrade` + `Upgrade: websocket` headers are filtered; everything else passes through
- The first config whose path patterns match is used; empty `TokenQueryKey` or missing token → `401`
- `checkAuth(token, pc)` decides; failure responds with the returned status code (default `401`). The callback may store user info via `pc.SetAttr()`

## Other Built-in Filters

### AddIPBlacklistFilter

`AddIPBlacklistFilter(matchBlacklist func(ip string) bool)` — IP is extracted from the remote address (port stripped); matched requests get a fake `200` (no body).

### AddPathFilter

`AddPathFilter(pathPatterns []string, f ProxyFilter)` — runs `f` only when the proxy path matches any pattern, otherwise continues the chain.

### AddReqTimeLogFilter

`AddReqTimeLogFilter(exclPath func(proxyPath string) bool, unit ...ReqTimeLogUnit)` — logs `Receive 'METHOD URI [ip]'` before and `Processed ... [duration]` after, skipping excluded paths. With a unit (`ReqTimeLogUnit{Dur: time.Millisecond, Name: "ms"}`) the duration is printed in that unit.

### AddDebugFilter

`AddDebugFilter(mustAuthInProd bool) error` — registers handlers for `/debug/pprof/**` and `/debug/trace/**` (FlightRecorder recorder run/stop/snapshot). Only active for root path proxy and when `server.pprof.enabled` is true. Whenever `server.pprof.auth.bearer` is configured, requests must present the matching bearer token — regardless of environment. If `mustAuthInProd` is true and running in prod, `server.pprof.auth.bearer` must be configured, otherwise `AddDebugFilter` returns an error.

### AddHealthcheckFilter

`AddHealthcheckFilter()` — serves the health check at `server.health-check-url` (default `/health`), rate-limited to once per second; `200` when healthy, `503` otherwise. Only active for root path proxy. See [health-checks.md](health-checks.md).

### AddMetricsFilter

`AddMetricsFilter(hiso prometheus.Histogram, exclPath func(proxyPath string) bool)` — serves the Prometheus scrape endpoint at `metrics.route` (default `/metrics`) locally, and observes a histogram on proxied requests excluding matching paths. Only active for root path proxy.

### ChangeClient

`ChangeClient(c *http.Client)` replaces the proxy's HTTP client (panic if nil).

## Trace Propagation

Inbound requests may contain forged trace headers, so before proxying:

- All inbound headers matching the propagation keys are removed
- The `XSpanId` header is replaced with a fresh span id
- The current Rail's trace values are written into the outgoing request headers

This ensures downstream services see the proxy's own trace context, not attacker-supplied values.

## Best Practices

1. Call `NewHttpProxy` before `BootstrapServer` — registration happens at call time
2. Prefer `AddConfDynAccessFilter` with a refresh interval over hardcoded auth logic for gateway services — credentials and whitelists can change without redeploys
3. Keep remote auth routes with no `path-patterns` last — they match every path
4. When proxying the root path, re-register health check (`AddHealthcheckFilter`), metrics (`AddMetricsFilter`) and debug (`AddDebugFilter`) filters explicitly, since the default handlers are disabled
5. Use `ProxyHttpStatusError` from the target resolver to distinguish "not found" from other failures
6. Return `GatewayError`/custom status from resolvers instead of raw errors to control the client-visible status code

## Reference Files

- `miso/proxy.go` - HttpProxy implementation, filters, dyn auth routes
- `miso/proxy_dynremote_test.go` - Tests for `AddConfDynAccessFilter`, remote auth result mapping
- `demo/proxydemo/main.go` - Minimal proxy example
- `util/strutil/strutil.go` - `MatchPathAny` / `MatchPathAnyVal` path pattern matching
