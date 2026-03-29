# Web Development

Building web servers and RESTful APIs with miso using Gin integration.

## Two Approaches

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
# Generate API code
misoapi

# Generate and run the app (for API doc generation)
misoapi -run
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
    return errs.WrapErr(err, "HTTP request failed")
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
    return errs.WrapErr(err, "HTTP request failed")
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
    return errs.WrapErr(err, "HTTP request failed")
}
```

### Write Response to Writer

```go
err := miso.NewClient(rail, "https://api.example.com/file").
    Require2xx().
    Get().
    WriteTo(writer)

if err != nil {
    return errs.WrapErr(err, "HTTP request failed")
}
```

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