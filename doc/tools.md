# Tools

## `misogen` - generate miso project

Install latest `misogen` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misogen@v0.1.19
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

$ mkdir myapp && cd myapp && misogen -name "myapp" -svc

# misogen, current miso version: v0.1.19
#
# Initialized module 'myapp'
# Installing dependency: github.com/curtisnewbie/miso/miso@v0.1.19
# Initializing conf.yml
# Initializing internal/schema/scripts/schema.sql
# Initializing internal/schema/migrate.go
# Initializing main.go
```

## `misoapi` - generate api endpoints

Install latest `misoapi` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misoapi@v0.1.19
```

```sh
$ misoapi -h

# misoapi - automatically generate web endpoint in go based on misoapi-* comments
#
#   Supported miso version: v0.1.19
#
# Usage of misoapi:
#   -debug
#         Enable debug log
#
# For example:
#
#   misoapi-http: GET /open/api/doc
#   misoapi-desc: open api endpoint to retrieve documents
#   misoapi-query-doc: page: curent page index
#   misoapi-header-doc: Authorization: bearer authorization token
#   misoapi-scope: PROTECTED
#   misoapi-resource: document:read
#   misoapi-ngtable
```

## `misocurl` - generate miso.TClient from curl

Install latest `misocurl` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misocurl@v0.1.19
```

```sh
$ misocurl -h

# misocurl - automatically miso.TClient code based on curl in clipboard
#
#   Supported miso version: v0.1.19-beta.1
#
# Usage of misocurl:
#   -debug
#         Debug
```

## `misoconfig` - generate config doc

Install latest `misoconfig` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misoconfig@v0.1.19
```

```sh
$ misoconfig -h

# misoconfig - automatically generate configuration tables based on misoconfig-* comments
#
#   Supported miso version: v0.1.19
#
# Usage of misoconfig:
#   -debug
#         Enable debug log
#   -path string
#         Path to the generated markdown config table file
#
# For example:
#
#
# // misoconfig-section: Web Server Configuration
# const (
#
#         // misoconfig-prop: enable http server | true
#         PropServerEnabled = "server.enabled"
#
# )
```
