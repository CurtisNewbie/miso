# Tools

## `misogen` - generate miso project

Install latest `misogen` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misogen@v0.4.12
```

Use `misogen` to generate new projects, e.g.,

```sh
$ misogen -h

# Usage of misogen:
#   -cli
#         Generate CLI style project
#   -disable-web
#         Disable web server
#   -name string
#         Module name
#   -static
#         Generate code to embed and statically host frontend project
#   -svc
#         Generate code to integrate svc for automatic schema migration

$ mkdir myapp && cd myapp && misogen -name "myapp"

# misogen, current miso version: v0.3.6
#
# Initialized module 'myapp'
# Installing dependency: github.com/curtisnewbie/miso/miso@v0.3.6
# Initializing conf.yml
# Initializing main.go
```

## `misoapi` - generate api endpoints

Install latest `misoapi` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misoapi@v0.4.12
```

```sh
$ misoapi -h

# misoapi - automatically generate web endpoint in go based on misoapi-* comments
#
#   Supported miso version: v0.4.10-beta.3
#
#
# Usage of misoapi:
#   -debug
#         Enable debug log
#   -run
#         Run app after api generated (to generate api doc)
#
#
# For example:
#
#   misoapi-http: GET /open/api/doc                                     // http method and url
#   misoapi-desc: open api endpoint to retrieve documents               // description
#   misoapi-query: page: curent page index                              // query parameter
#   misoapi-header: Authorization: bearer authorization token           // header parameter
#   misoapi-scope: PROTECTED                                            // access scope
#   misoapi-resource: document:read                                     // resource code
#   misoapi-ngtable                                                     // generate angular table code
#   misoapi-raw                                                         // raw endpoint without auto request/response json handling
#   misoapi-json-resp-type: MyResp                                      // json response type (struct), for raw api only
#   misoapi-ignore                                                      // ignored by misoapi
#
# Important:
#
#   By default, misoapi looks for `func PrepareWebServer(rail miso.Rail) error` in file './internal/web/web.go'.
#   If file is not found, APIs are registered in init() func, however it's not recommended as it's implicit.
#   If the file is found, APIs are registered explicitly in PrepareWebServer(..) func, and you should
#   makesure the PrepareWebServer(..) is called in miso.PreServerBootstrap(..)
```

## `misocurl` - generate miso.TClient from curl

Install latest `misocurl` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misocurl@v0.4.12
```

```sh
$ misocurl -h

# misocurl - automatically miso.TClient code based on curl in clipboard
#
#   Supported miso version: v0.3.6
#
# Usage of misocurl:
#   -debug
#         Debug
```

## `misoconfig` - generate config doc

Install latest `misoconfig` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misoconfig@v0.4.12
```

```sh
$ misoconfig -h

# misoconfig - automatically generate configuration tables based on misoconfig-* comments
#
#   Supported miso version: v0.3.6
#
# Usage of misoconfig:
#   -debug
#         Enable debug log
#   -path string
#         Path to the generated markdown config table file
#
# For example:
#
# In prop.go:
#
#   // misoconfig-section: Web Server Configuration
#   const (
#
#           // misoconfig-prop: enable http server | true
#           PropServerEnabled = "server.enabled"
#
#           // misoconfig-prop: my prop
#           // misoconfig-alias: old-prop
#           PropDeprecated = "new-prop"
#
#           // misoconfig-prop: my special prop
#           // misoconfig-doc-only
#           PropDocOnly = "prod-only-shown-in-doc"
#
#           // misoconfig-default-start
#           // misoconfig-default-end
#   )
#
# In ./doc/config.md:
#
#   <!-- misoconfig-table-start -->
#   <!-- misoconfig-table-end -->
```

## `misopatch` - apply gopatch for code migration

Install latest `misopatch` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misopatch@v0.4.12
```

```sh
$ misopatch -h

# misopatch - automatically apply gopatch on current working directory
#
#   miso build version: v0.3.6
#
#
# Usage of misopatch:
#  -all
#        apply all patches
#   -debug
#         enable debug log
```
