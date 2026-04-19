# Command-Line Tools

miso framework provides several CLI tools to streamline development workflow:

**Table of Contents:**
- Installation
- misogen
- misoapi
- misocurl
- misopatch
- misoconfig
- Common Patterns

- **misogen** - Generate new project scaffolding
- **misoapi** - Auto-generate web endpoint registration from comments
- **misocurl** - Generate HTTP client code from curl command
- **misopatch** - Apply version migration patches
- **misoconfig** - Generate configuration documentation tables

## Installation

Install tools via Go:

```bash
go install github.com/curtisnewbie/miso/cmd/misogen@latest
go install github.com/curtisnewbie/miso/cmd/misoapi@latest
go install github.com/curtisnewbie/miso/cmd/misocurl@latest
go install github.com/curtisnewbie/miso/cmd/misopatch@latest
go install github.com/curtisnewbie/miso/cmd/misoconfig@latest
```

## misogen

Generate new miso project scaffolding with proper directory structure and boilerplate code.

### Usage

```bash
# Generate project in existing module (requires go.mod)
misogen

# Generate new project with module name
misogen -name github.com/username/myapp

# Options
misogen -name github.com/username/myapp [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `-name string` | Module name for new project |
| `-static` | Generate code to embed and statically host frontend project |
| `-svc` | Generate code to integrate svc for automatic schema migration |
| `-disable-web` | Disable web server in generated project |
| `-cli` | Generate CLI style project instead of web server |

### Generated Structure

Standard web project:
```
myapp/
├── main.go
├── conf.yml
├── doc/
│   └── .gitkeep
└── internal/
    ├── server/
    │   ├── server.go
    │   └── version.go
    ├── config/
    │   └── prop.go
    ├── repo/
    │   └── .gitkeep
    ├── domain/
    │   └── .gitkeep
    └── web/
        └── web.go
```

With `-svc` flag adds:
```
└── internal/
    └── schema/
        ├── scripts/
        │   └── schema.sql
        └── migrate.go
```

With `-static` flag adds:
```
└── internal/
    └── static/
        ├── static.go
        └── static/
            └── miso.html
```

## misoapi

Automatically generate web endpoint registration code from `misoapi-*` comments.

### Usage

```bash
# Generate endpoint registration code
misoapi

# Generate and run app (to generate API doc)
misoapi -run

# Enable debug logging
misoapi -debug
```

### Supported Comments

Place these comments above your API handler functions:

```go
// misoapi-http: GET /api/users
// misoapi-desc: List all users
// misoapi-scope: PROTECTED
// misoapi-resource: user:read
// misoapi-query: page: current page index
// misoapi-header: Authorization: bearer authorization token
// misoapi-raw
// misoapi-json-resp-type: UserListResp
// misoapi-ignore
func GetUsers(inb *miso.Inbound) ([]User, error) {
    // implementation
}
```

### Comment Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `misoapi-http` | HTTP method and URL | `GET /api/users` |
| `misoapi-desc` | Endpoint description | `List all users` |
| `misoapi-scope` | Access scope | `PROTECTED`, `PUBLIC`, or custom |
| `misoapi-resource` | Resource code for authorization | `user:read` |
| `misoapi-query` | Query parameter documentation | `page: current page index` |
| `misoapi-header` | Header parameter documentation | `Authorization: bearer token` |
| `misoapi-raw` | Raw endpoint without auto JSON handling | (no value) |
| `misoapi-json-resp-type` | JSON response type for raw API | `UserListResp` |
| `misoapi-ignore` | Exclude from misoapi generation | (no value) |
| `misoapi-ngtable` | Generate Angular table code | (no value) |

### Handler Signatures

Supported handler patterns:

```go
// AutoHandler with request/response
func CreateUser(inb *miso.Inbound, req CreateUserReq) (CreateUserRes, error)

// ResHandler with response only
func GetUser(inb *miso.Inbound) (User, error)

// RawHandler with full control
func RawEndpoint(inb *miso.Inbound)

// Injected parameters (auto-injected by framework)
func HandlerWithDb(inb *miso.Inbound, db *dbquery.DB, rail miso.Rail) error
```

### Auto-injected Parameters

The following parameter types are auto-injected:

| Type | Value |
|------|-------|
| `*miso.Inbound` | `inb` |
| `miso.Rail` | `inb.Rail()` |
| `*mysql.Query` | `mysql.NewQuery(dbquery.GetDB())` |
| `*gorm.DB` | `dbquery.GetDB()` |
| `flow.User` | `inb.Rail().User()` |

### Registration

If `internal/web/web.go` exists with `PrepareWebServer(rail miso.Rail) error`, misoapi inserts registration calls there. Otherwise, it generates an `init()` function.

## misocurl

Generate HTTP client code from curl command in clipboard.

### Usage

```bash
# Copy curl command to clipboard, then run
misocurl

# Enable debug output
misocurl -debug
```

### Features

- Reads curl command from system clipboard
- Generates `miso.TClient` code with proper error handling
- Automatically generates Go structs from JSON payloads
- Supports GET, POST, PUT, DELETE methods
- Handles headers, form data, and JSON payloads
- Also works with JSON only (generates Go struct)

### Example

Input (clipboard):
```bash
curl -X POST https://api.example.com/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token" \
  -d '{"name":"John","email":"john@example.com"}'
```

Output:
```go
type Req struct {
    Email string `json:"email"`
    Name  string `json:"name"`
}

rail := miso.EmptyRail()
s, err := miso.NewClient(rail, "https://api.example.com/users").
    AddHeader("Content-Type", "application/json").
    AddHeader("Authorization", "Bearer token").
    PostJson(Req{}).
    Str()
if err != nil {
    panic(err)
}
rail.Infof("Response: %v", s)
```

## misopatch

Apply version migration patches using gopatch.

### Usage

```bash
# Apply patches up to current version
misopatch

# Apply all patches regardless of version
misopatch -all

# Enable debug logging
misopatch -debug
```

### Features

- Auto-detects current miso version from `go.mod`
- Applies patches from `patch/` directory in version order
- Auto-installs gopatch if not found
- Applies all patches with `-all` flag
- Patches are named by version (e.g., `v0.4.0.patch`)

### Manual Patch Application

```bash
# Apply specific patch
gopatch -p /path/to/miso/patch/v0.4.0.patch ./...
```

## misoconfig

Generate configuration documentation tables from source code comments.

### Usage

```bash
# Generate config table (default: ./doc/config.md)
misoconfig

# Specify custom path
misoconfig -path ./docs/configuration.md

# Enable debug logging
misoconfig -debug
```

### Supported Comments

```go
// misoconfig-section: Web Server Configuration
const (
    // misoconfig-prop: enable http server | true
    PropServerEnabled = "server.enabled"

    // misoconfig-prop: deprecated property
    // misoconfig-alias: old-prop | v0.4.0
    PropNewName = "new-prop"

    // misoconfig-prop: production-only property
    // misoconfig-doc-only
    PropProdOnly = "prod.only"
)

// misoconfig-default-start
// misoconfig-default-end
```

### Comment Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `misoconfig-section` | Section name for grouping configs | `Web Server Configuration` |
| `misoconfig-prop` | Property description | `desc \| default value` |
| `misoconfig-alias` | Deprecated property alias | `old-name \| version` |
| `misoconfig-doc-only` | Only show in documentation | (no value) |

### Output

Generates markdown tables in format:

| Property | Description | Default Value |
|----------|-------------|---------------|
| `server.enabled` | enable http server | `true` |

Also generates default value initialization code between `// misoconfig-default-start` and `// misoconfig-default-end` markers.

### Embedding

To embed config table in existing markdown:

```markdown
# My Documentation

## Configuration

<!-- misoconfig-table-start -->
<!-- misoconfig-table-end -->
```

## Common Patterns

### New Project Workflow

```bash
# 1. Generate project
mkdir myapp && cd myapp
go mod init github.com/username/myapp
misogen

# 2. Define API with comments
# In internal/user/api.go:
// misoapi-http: POST /user
// misoapi-desc: Create user
func CreateUser(inb *miso.Inbound, req CreateUserReq) (CreateUserRes, error)

# 3. Generate endpoint registration
misoapi

# 4. Run and generate API doc
misoapi -run

# 5. Test with generated HTTP client
# Copy curl from API doc, then:
misocurl
```

### Version Migration Workflow

```bash
# Update miso version
go get github.com/curtisnewbie/miso@latest

# Apply patches
misopatch

# Run go mod tidy
go mod tidy
```

### Configuration Documentation Workflow

```bash
# Define props with comments
# In internal/config/prop.go

// misoconfig-section: General
const (
    // misoconfig-prop: app name | myapp
    PropAppName = "app.name"
)

// misoconfig-default-start
// misoconfig-default-end

# Generate documentation
misoconfig
```