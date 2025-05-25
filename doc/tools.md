# Tools

## `misogen` - generate miso project

Install latest `misogen` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misogen@v0.1.25
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

# misogen, current miso version: v0.1.25
#
# Initialized module 'myapp'
# Installing dependency: github.com/curtisnewbie/miso/miso@v0.1.25
# Initializing conf.yml
# Initializing main.go
```

## `misoapi` - generate api endpoints

Install latest `misoapi` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misoapi@v0.1.25
```

```sh
$ misoapi -h

# misoapi - automatically generate web endpoint in go based on misoapi-* comments
#
#   Supported miso version: v0.1.25
#
# Usage of misoapi:
#   -debug
#         Enable debug log
#
# For example:
#
#   misoapi-http: GET /open/api/doc                                     // http method and url
#   misoapi-desc: open api endpoint to retrieve documents               // description
#   misoapi-query-doc: page: curent page index                          // query parameter
#   misoapi-header-doc: Authorization: bearer authorization token       // header parameter
#   misoapi-scope: PROTECTED                                            // access scope
#   misoapi-resource: document:read                                     // resource code
#   misoapi-ngtable                                                     // generate angular table code
#   misoapi-raw                                                         // raw endpoint without auto request/response json handling
#   misoapi-json-resp-type: MyResp                                      // json response type (struct), for raw api only
```

## `misocurl` - generate miso.TClient from curl

Install latest `misocurl` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misocurl@v0.1.25
```

```sh
$ misocurl -h

# misocurl - automatically miso.TClient code based on curl in clipboard
#
#   Supported miso version: v0.1.25
#
# Usage of misocurl:
#   -debug
#         Debug
```

## `misoconfig` - generate config doc

Install latest `misoconfig` tool:

```sh
go install github.com/curtisnewbie/miso/cmd/misoconfig@v0.1.25
```

```sh
$ misoconfig -h

# misoconfig - automatically generate configuration tables based on misoconfig-* comments
#
#   Supported miso version: v0.1.25
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
