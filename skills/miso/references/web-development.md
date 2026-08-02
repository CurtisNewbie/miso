# Web Development

Building web servers and RESTful APIs with miso using Gin integration.

**Table of Contents:**
- Two Approaches
- Handler Types
- Auto Generation with misoapi
- Request/Response
- HTTP Methods
- Query Parameters
- miso.Inbound Methods
- Validation
- Metadata and Documentation
- Interceptors
- Manual Response
- Middleware
- HTTP Client
- Static Files
- Configuration

miso supports two ways to register HTTP endpoints:

1. **Manual Registration** - Use `miso/web.go` functions like `HttpGet()`, `HttpPost()`, `AutoHandler()`
2. **Auto Generation** - Use `misoapi` tool with `misoapi-*` comments to auto-generate endpoint registration code

Choose the approach that fits your workflow. Both produce the same runtime behavior.

## Handler Types

```go
// RawHandler - Full control over request/response
miso.HttpGet("/health", miso.RawHandler(func(inb *miso.Inbound) {
    // Use inb.Request(), inb.Writer(), inb.Header(), etc.
    inb.HandleResult(nil, nil)
}))

// ResHandler - Auto response wrapping, no request type
miso.HttpGet("/ping", miso.ResHandler[string](func(inb *miso.Inbound) (string, error) {
    return "pong", nil
}))

// AutoHandler - Auto request parsing and response wrapping
miso.HttpPost("/user", miso.AutoHandler[CreateUserReq, CreateUserRes](CreateUser))
```

## Auto Generation with misoapi

The `misoapi` tool scans your code for `misoapi-*` comments and generates endpoint registration code.

### Installation

```bash
go install github.com/curtisnewbie/miso/cmd/misoapi@latest
```

### Running misoapi

```bash
# Generate API code and API doc
misoapi
```

### Defining Endpoints

```go
// misoapi-http: GET /hello
// misoapi-desc: Simple hello endpoint
// misoapi-scope: PUBLIC
func Hello(inb *miso.Inbound) (string, error) {
    return "hello world", nil
}
```

### Request/Response Types

```go
// misoapi-http: POST /user
// misoapi-desc: Create a new user
// misoapi-query: page: current page index
// misoapi-header: Authorization: bearer authorization token
// misoapi-scope: PROTECTED
// misoapi-resource: user:create
func CreateUser(inb *miso.Inbound, req CreateUserReq) (CreateUserRes, error) {
    // Handler implementation
    return CreateUserRes{UserID: "123"}, nil
}

type CreateUserReq struct {
    Name  string `json:"name" valid:"notEmpty:Name is required"`
    Email string `json:"email" valid:"notEmpty:Email is required"`
}

type CreateUserRes struct {
    UserID string `json:"userId"`
}
```

### misoapi Tags

| Tag | Description |
|-----|-------------|
| `misoapi-http` | HTTP method and URL (required) |
| `misoapi-desc` | Endpoint description |
| `misoapi-scope` | Access scope (PUBLIC, PROTECTED) |
| `misoapi-resource` | Resource code for permission checking (supports `ref()` syntax to reference constants) |
| `misoapi-query` | Query parameter documentation |
| `misoapi-header` | Header parameter documentation |
| `misoapi-ngtable` | Generate Angular table code |
| `misoapi-raw` | Raw endpoint without auto JSON handling |
| `misoapi-json-resp-type` | Custom response type (for raw endpoints) |
| `misoapi-ignore` | Ignore this function |

### Where to Register Endpoints

By default, misoapi looks for `PrepareWebServer` in `./internal/web/web.go`:

```go
package web

import "github.com/curtisnewbie/miso/miso"

func PrepareWebServer(rail miso.Rail) error {
    // Endpoints are registered here via generated code
    rail.Infof("Web server prepared")
    return nil
}

func init() {
    miso.PreServerBootstrap(PrepareWebServer)
}
```

### Builtin Auto-Injected Parameters

misoapi automatically injects these parameters in generated code:

| Parameter Type | Auto-Injected Value |
|----------------|---------------------|
| `*miso.Inbound` | `inb` |
| `miso.Rail` | `inb.Rail()` |
| `*mysql.Query` | `mysql.NewQuery(dbquery.GetDB())` |
| `*gorm.DB` | `dbquery.GetDB()` |
| `flow.User` | `inb.Rail().User()` |

Example:

```go
// misoapi-http: GET /user
func GetUser(inb *miso.Inbound, req GetUserReq) (GetUserRes, error) {
    // inb is auto-injected
    rail := inb.Rail()  // or use miso.Rail directly
    // req is the custom request type
    return GetUserRes{}, nil
}

// misoapi-http: GET /user
func GetUserWithDb(req GetUserReq, db *gorm.DB) (GetUserRes, error) {
    // db is auto-injected as dbquery.GetDB()
    // use dbquery.NewQuery(db) for queries
    return GetUserRes{}, nil
}

// misoapi-http: GET /user
func GetUserWithQuery(req GetUserReq, qry *mysql.Query) (GetUserRes, error) {
    // qry is auto-injected as mysql.NewQuery(dbquery.GetDB())
    return GetUserRes{}, nil
}
```

**Note:** Use pointer types for `*miso.Inbound` and `*gorm.DB`, but value types for `miso.Rail` and `flow.User`.

## Request/Response

### Request Structure

```go
type CreateUserReq struct {
    Name  string `json:"name" valid:"notEmpty:Name is required"`
    Email string `json:"email" valid:"notEmpty:Email is required"`
    Age   int    `json:"age" valid:"positive"`
}
```

Fields are automatically mapped from:
- JSON body (POST/PUT) - using `json` tag
- Query parameters (GET) - using `form` tag
- Headers - using `header` tag

### Response Structure

```go
type CreateUserRes struct {
    UserID string `json:"userId"`
    Name   string `json:"name"`
    Email  string `json:"email"`
}
```

### JSON Tags

JSON tags are **required** (since v0.4.0):

```go
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name" desc:"User name"`
    Email string `json:"email" desc:"User email address"`
}
```

### Automatic Response Wrapping

Framework automatically wraps responses in JSON format:

```json
{
  "errorCode": "",
  "msg": "ok",
  "error": false,
  "data": {
    "userId": "123",
    "name": "John"
  }
}
```

Error responses:

```json
{
  "errorCode": "USER_NOT_FOUND",
  "msg": "User does not exist",
  "error": true,
  "data": null
}
```

## HTTP Methods

```go
miso.HttpGet("/api/users", GetUsers)
miso.HttpPost("/api/users", CreateUser)
miso.HttpPut("/api/users/:id", UpdateUser)
miso.HttpDelete("/api/users/:id", DeleteUser)
miso.HttpPatch("/api/users/:id", PatchUser)
miso.HttpHead("/api/users", HeadUsers)
miso.HttpOptions("/api/users", OptionsUsers)
miso.HttpTrace("/api/debug", TraceDebug)
miso.HttpConnect("/api/conn", ConnectHandler)
```

## Query Parameters

Query parameters are automatically mapped to request struct:

```go
func ListUsers(inb *miso.Inbound, req ListUsersReq) (ListUsersRes, error) {
    inb.Rail().Infof("Listing users: page=%d, size=%d", req.Page, req.Size)
    // Handler implementation
    return ListUsersRes{}, nil
}

type ListUsersReq struct {
    Page   int    `json:"page" form:"page" valid:"positive"`
    Size   int    `json:"size" form:"size" valid:"positive"`
    Filter string `json:"filter" form:"filter"`
}

func init() {
    miso.HttpGet("/users", miso.AutoHandler[ListUsersReq, ListUsersRes](ListUsers)).
        DocQueryReq(ListUsersReq{})
}
```

## miso.Inbound Methods

The `miso.Inbound` parameter provides access to request/response context:

| Method | Description |
|--------|-------------|
| `Rail()` | Get Rail for logging and tracing |
| `Engine()` | Get underlying Gin Context (use with caution) |
| `Unwrap()` | Get (http.ResponseWriter, *http.Request) |
| `Writer()` | Get http.ResponseWriter |
| `Request()` | Get *http.Request |
| `Status(status int)` | Set HTTP status code |
| `HandleResult(result any, err error)` | Handle result using framework result handler |
| `WriteJson(v any)` | Write JSON response |
| `WriteString(v string)` | Write plain text response |
| `WriteJsonStatus(v any, status int)` | Write JSON with status code |
| `Query(k string) string` | Get single query parameter |
| `Queries() url.Values` | Get all query parameters |
| `Header(k string) string` | Get single header value |
| `Headers() http.Header` | Get all headers |
| `SetHeader(k, v string)` | Set header |
| `AddHeader(k, v string)` | Add header |
| `MustBind(ptr any)` | Bind request to struct |
| `ReadRawBytes() ([]byte, error)` | Read raw request body |
| `WriteSSE(name string, message any)` | Write Server-Sent Event |
| `LogRequest()` | Log request details (headers/body) |
| `LogHeaders()` | Log request headers only |

```go
func CustomHandler(inb *miso.Inbound) {
    // Access request details
    inb.Rail().Infof("Processing request: %s", inb.Request().URL.Path)
    token := inb.Header("Authorization")
    userId := inb.Query("userId")

    // Manual response
    inb.Status(http.StatusCreated)
    inb.WriteJson(map[string]string{"id": "123"})

    // Or use framework result handler
    inb.HandleResult(data, nil)
}
```

## Validation

Use `valid` tags for request validation:

```go
type MyReq struct {
    Name     string `valid:"maxLen:10,notEmpty:Name is required"`
    Count    int    `valid:"positive:Count must be positive"`
    Type     string `valid:"member:PUBLIC|PROTECTED|PRIVATE"`
    Optional *Child `valid:"notNil,validated"`
}

type Child struct {
    Value string `valid:"notEmpty"`
}
```

Validation automatically runs before handler execution. Validation errors return a generic error with the first validation failure:

```json
{
  "errorCode": "XXXX",
  "msg": "name Name is required",
  "error": true,
  "data": null
}
```

## Metadata and Documentation

### Description

```go
miso.HttpPost("/user", CreateUserHandler).
    Desc("Create a new user with validation")
```

### Access Scope

```go
miso.HttpPost("/user", CreateUserHandler).
    Public()  // Public endpoint

miso.HttpDelete("/user/:id", DeleteUserHandler).
    Protected()  // Protected endpoint
```

### Resource Code

```go
miso.HttpPost("/user", CreateUserHandler).
    Resource("user:create")
```

### Parameter Documentation

```go
type ListUsersReq struct {
    Page   int    `json:"page" form:"page"`
    Size   int    `json:"size" form:"size"`
    Token  string `json:"token" header:"Authorization"`
}

func init() {
    miso.HttpGet("/users", ListUsersHandler).
        DocQueryReq(ListUsersReq{}).
        DocHeaderReq(ListUsersReq{}).
        DocQueryParam("filter", "Filter by name or email").
        DocHeader("X-Custom-Header", "Custom header value")
}
```

## Global Bearer Auth

Set `server.auth.bearer` to protect **every** endpoint (including pprof/trace debug routes) with a fixed bearer token:

```yaml
server:
  auth:
    bearer: "your-secret-token"
```

The token is checked by an auto-registered interceptor; requests without a matching `Authorization: Bearer <token>` header are rejected.

## Route Grouping and Introspection

```go
// Group routes under a URL prefix (mirrors the controller's context path)
miso.GroupRoute("/api/v1",
    miso.HttpGet("/users", ListUsersHandler),
    miso.HttpPost("/users", CreateUserHandler),
)

// Register the same handler for all HTTP methods
miso.HttpAny("/callback", WebhookHandler)

// Basic auth for the whole server
miso.EnableBasicAuth(func(username, password, url, method string) bool {
    return username == "admin" && password == "secret"
})

// Introspect registered routes + metadata
routes := miso.GetHttpRoutes() // []HttpRoute
```

## Interceptors

Add request interceptors for authentication, logging, etc.:

```go
func init() {
    miso.AddBearerAuthInterceptor(
        func(method, url string) bool {
            // Return true for endpoints that need auth
            return !strings.HasPrefix(url, "/public/")
        },
        func(token string) bool {
            // Validate bearer token
            return validateToken(token)
        },
    )
}
```

### Custom Interceptor

```go
miso.AddInterceptor(func(c *gin.Context, next func()) {
    // Pre-request logic
    rail := miso.BuildRail(c)
    rail.Infof("Intercepting request: %s %s", c.Request.Method, c.Request.URL)

    next()

    // Post-request logic
    status := c.Writer.Status()
    rail.Infof("Response status: %d", status)
})
```

### CORS

```go
func init() {
    miso.AddCorsAny()
}
```

## Manual Response

For raw handlers that need full control:

```go
func CustomHandler(inb *miso.Inbound) error {
    // Manual JSON response
    inb.WriteJson(map[string]string{"message": "hello"})

    // Manual text response
    inb.WriteString("hello")

    // Manual status code
    inb.Status(http.StatusCreated)
    inb.WriteJson(data)

    // Use framework's automatic wrapping
    inb.HandleResult(data, nil)  // Success
    inb.HandleResult(nil, err)   // Error
}
```

## Global Response Model

All endpoint responses share a uniform JSON body — `{errorCode, msg, error, data}` — produced by the framework's `WrapResp`. The response model types are available for client-side unmarshalling:

```go
type Resp struct {
    ErrorCode string      `json:"errorCode"`
    Msg       string      `json:"msg"`
    Error     bool        `json:"error"`
    Data      interface{} `json:"data"`
}

// Generic version
type GnResp[T any] struct {
    ErrorCode string `json:"errorCode"`
    Msg       string `json:"msg"`
    Error     bool   `json:"error"`
    Data      T      `json:"data"`
}
```

```go
var r miso.GnResp[User]
err := miso.NewClient(rail, "https://api.example.com/user").
    Get().
    Json(&r)
if err != nil {
    return err
}
user, err := r.Res() // (T, error), error from ErrorCode/Msg if Error
if err != nil {
    return errs.Wrapf(err, "failed to load user")
}

// Map error codes to typed errors
user, err = r.MappedRes(map[string]error{
    "USER_NOT_FOUND": errs.NewErrfCode("USER_NOT_FOUND", "User does not exist"),
})

// Build response bodies
okResp := miso.OkResp()
okData := miso.OkRespWData(data)
errResp := miso.ErrorResp("Something went wrong")
errCodeResp := miso.ErrorRespWCode("DB_ERROR", "Database error")
```

**Replacing the default wrapper:** use `miso.SetResultBodyBuilder(builder)` to change how the global response body is built (e.g., custom error/msg/ok shapes).

## Middleware

```go
import (
    "time"

    "github.com/curtisnewbie/miso/miso"
    "github.com/curtisnewbie/miso/flow"
)

func init() {
    miso.PreProcessGin(func(rail miso.Rail, engine *gin.Engine) {
        // Add custom middleware
        engine.Use(customMiddleware())
    })
}

func customMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        latency := time.Since(start)
        flow.Infof("Request completed in %v", latency)
    }
}
```

## HTTP Client

### Basic Usage

```go
import "github.com/curtisnewbie/miso/miso"

var resp Data
err := miso.NewClient(rail, "https://api.example.com/data").
    Require2xx().
    Get().
    Json(&resp)

if err != nil {
    rail.Errorf("Request failed: %v", err)
    return
}

rail.Infof("Response: %v", resp)
```

### POST JSON Request

```go
import "github.com/curtisnewbie/miso/errs"

var result Result
err := miso.NewClient(rail, "https://api.example.com/data").
    Require2xx().
    PostJson(payload).
    Json(&result)

if err != nil {
    return errs.Wrapf(err, "HTTP request failed")
}
```

### Request with Headers and Query Params

```go
import "github.com/curtisnewbie/miso/errs"

var data Data
err := miso.NewClient(rail, "https://api.example.com/data").
    AddHeader("Authorization", "Bearer token").
    AddQuery("page", "1").
    AddQuery("size", "10").
    Require2xx().
    Get().
    Json(&data)

if err != nil {
    return errs.Wrapf(err, "HTTP request failed")
}

// resp.StatusCode, resp.RespHeader also available
```

### Dynamic Client with Service Discovery

```go
import "github.com/curtisnewbie/miso/errs"

var resp Data
err := miso.NewDynClient(rail, "/api/data", "user-vault").
    Require2xx().
    Get().
    Json(&resp)

if err != nil {
    return errs.Wrapf(err, "HTTP request failed")
}
```

### Write Response to Writer

```go
err := miso.NewClient(rail, "https://api.example.com/file").
    Require2xx().
    Get().
    WriteTo(writer)

if err != nil {
    return errs.Wrapf(err, "HTTP request failed")
}
```

### Advanced Client Features

```go
// Chrome TLS-fingerprint impersonation (anti-bot detection)
resp := miso.NewClient(rail, "https://api.example.com/data").
    Impersonate().
    Require2xx().
    Get()

// Route through an HTTP proxy
miso.NewClient(rail, "https://api.example.com/data").WithProxy("http://proxy:8080")

// Per-request service discovery (like NewDynClient)
miso.NewClient(rail, "https://api.example.com/data").EnableServiceDiscovery("user-vault")

// Attach trace headers manually for downstream calls
miso.NewClient(rail, "https://api.example.com/data").EnableTracing()

// Log request/response bodies
miso.NewClient(rail, "https://api.example.com/data").LogBody()

// Bearer auth
miso.NewClient(rail, "https://api.example.com/data").AddAuthBearer(token)

// Multipart upload
err := miso.NewClient(rail, "https://api.example.com/upload").
    Require2xx().
    PostFormData(map[string]io.Reader{
        "file": miso.NewReaderFile(f, "report.pdf"),
        "note": strings.NewReader("hello"),
    }).
    Json(&resp)

// Response inspection without JSON
tr := miso.NewClient(rail, "https://api.example.com/data").Require2xx().Get()
if tr.Ok() != nil { /* not 2xx */ }
if tr.Is2xx() { /* 2xx */ }
data, err := tr.JsonStr(&resp) // returns raw JSON string too
n, err := tr.WriteToFile("download.bin") // save response body to file
```

## HTTP Reverse Proxy

Proxy requests to backend servers, with filter chains for auth/rate limiting/metrics.

```go
// Static target: proxy /proxy/** to localhost:8081
miso.NewHttpProxy("/proxy", func(rail miso.Rail, proxyPath string) (string, error) {
    return "http://localhost:8081" + proxyPath, nil
})
```

**Gateway pattern with service discovery** — route `/{serviceName}/path` to a registered service:

```go
miso.NewHttpProxy("/", miso.NewDynProxyTargetResolver())
```

**Filters** (run in registration order):

```go
p := miso.NewHttpProxy("/proxy", resolver)

// Path-scoped filter
p.AddPathFilter([]string{"/admin/**"}, func(pc *miso.ProxyContext, next func()) {
    // ... auth check, then next()
})

// Whitelist + auth check for the whole proxy
p.AddAccessFilter(func() []string { return whitelistPatterns },
    func(pc *miso.ProxyContext) (statusCode int, ok bool) { return 0, checkAuth(pc) })

// IP blacklist (returns fake 200 for blacklisted IPs)
p.AddIPBlacklistFilter(func(ip string) bool { return isBlacklisted(ip) })

// Request-time logging filter (exclude specific paths)
p.AddReqTimeLogFilter(func(proxyPath string) bool { return proxyPath == "/health" })

// Prometheus histogram metrics
p.AddMetricsFilter(histogram, nil)

// WebSocket upgrade auth via ?token=xxx query param
p.AddWsAccessFilter(func() []miso.WsAccessFilterConfig {
    return []miso.WsAccessFilterConfig{{
        PathPatterns:  []string{"/ws/**"},
        TokenQueryKey: "token",
    }}
}, func(token string, pc *miso.ProxyContext) (statusCode int, ok bool) {
    return 0, validateWsToken(token)
})

// Share state between filters via the proxy context
pc.SetAttr("user", u)
```

**Notes:**
- Must be called before server bootstrap
- If `proxiedPath` is `/`, default health/prometheus/pprof/apidoc handlers are disabled and inbound trace propagation is turned off (the proxy is the entry point)
- `ProxyTargetResolver` may return `miso.ProxyHttpStatusError`/`GatewayError` to respond with a specific status code
- Default proxy client: 5s connect timeout, 30s response-header timeout, 100 max idle conns per host, 500 max conns per host

## Static Files

miso provides built-in support for serving static files, including embedded files.

### Embedded Static Files

```go
//go:embed static
var staticFs embed.FS

func init() {
    // Serve static files from embedded fs at /static/*filepath
    // Note: index.html must be renamed to index.htm
    miso.PrepareWebStaticFs(staticFs, "static")
}
```

Build frontend with correct base path:
```bash
# Angular
ng build --baseHref=/static/

# React
npm run build -- --homepage=/static/
```

### File System Static Files

```go
func init() {
    miso.HttpGet("/static/*filepath", miso.RawHandler(func(inb *miso.Inbound) {
        c := inb.Engine().(*gin.Context)
        c.File("./static/" + c.Param("filepath"))
    }))

    miso.HttpGet("/favicon.ico", miso.RawHandler(func(inb *miso.Inbound) {
        c := inb.Engine().(*gin.Context)
        c.File("./resources/favicon.ico")
    }))
}
```

## Configuration

Web server is configured via YAML:

```yaml
server:
  enabled: true
  host: 127.0.0.1
  port: 8080
```

For complete configuration options, see [config.md](https://github.com/CurtisNewbie/miso/blob/main/doc/config.md).