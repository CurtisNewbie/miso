# Declaring Endpoints

miso only supports HTTP style API endpoints, there are three types of API handlers:

1. **_raw_ api handler**

   - handler should unmarshal request data and marshal response data itself.

2. **intelligent api handler with response**

   - handler doesn't need request data or chooses to unmarshal request data itself, the response and error are returned by handler and automatically handled by miso.

3. **intelligent api handler with request and response**
   - both request and response data are unmarshalled and marshalled by miso, handler just needs to process the data and returns results.

## Using **raw** Api Handler

Choose the appropriate `miso.Raw***()` method to register api endpoints before server bootstraps. Use `*miso.Inbound`'s funcs to read data from request and write data to response.

E.g.,

```go
// POST /api/do-something
miso.RawPost("/api/do-something", func(inb *miso.Inbound) {
    // ...
    inb.Status(http.StatusOK)
})

// OPTIONS /api/do-something
miso.RawOptions("/api/do-something", func(inb *miso.Inbound) {
    // ...
    inb.Status(http.StatusOK)
})

// HEAD /api/do-something
miso.RawHead("/api/do-something", func(inb *miso.Inbound) {
    // ...
    inb.Status(http.StatusOK)
})

// PATCH /api/do-something
miso.RawPatch("/api/do-something", func(inb *miso.Inbound) {
    // ...
    inb.Status(http.StatusOK)
})

// PUT /api/do-something
miso.RawPut("/api/do-something", func(inb *miso.Inbound) {
    // ...
    inb.Status(http.StatusOK)
})

// CONNECT /api/do-something
miso.RawConnect("/api/do-something", func(inb *miso.Inbound) {
    // ...
    inb.Status(http.StatusOK)
})

// Trace /api/do-something
miso.RawTrace("/api/do-something", func(inb *miso.Inbound) {
    // ...
    inb.Status(http.StatusOK)
})

// DELETE /api/do-something
miso.RawDelete("/api/do-something", func(inb *miso.Inbound) {
    // ...
    inb.Status(http.StatusOK)
})
```

## Using intelligent api handler with response

Choose the appropriate `miso.***()` method to register api endpoints before server bootstraps. Response data and error are wrapped by `miso.Resp` and written to clients in json format.

E.g.,

```go
miso.Post("/api/do-something",
    func(inb *miso.Inbound) (any, error) {
        return doSomething(inb)
    })

miso.Get("/api/do-something",
    func(inb *miso.Inbound) (any, error) {
        return nil, doSomething(inb)
    })
```

You are free to customize the response data and error data using `miso.SetResultBodyBuilder(...)`.

## Using intelligent api handler with request and response

Same as the ones above, except that you are now using `miso.I***()` methods, which automatically map request parameters (headers, query parameters, json request body) for you.

E.g.,

```go
miso.IPost("/api/do-something",
    func(inb *miso.Inbound, req ApiReq) (ApiRes, error) {
        return doSomething(inb, req)
    })
```

To map different kinds of parameters to your request struct, add following tags:

- `form:"xxx"`: map query parameter or form-data to struct field
- `header:"xxx"`: map header parameter to struct field

Json is the default mapping strategy, but you can still add `json:"xxx"` tag to customize the json processing behaviours.

E.g.,

```go
type LoginReq struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	XForwardedFor string `header:"x-forwarded-for"`
	UserAgent     string `header:"user-agent"`
}
```

or something like

```go
type FileInfoReq struct {
	FileId       string `form:"fileId"`
	UploadFileId string `form:"uploadFileId"`
}
```

## Using misoapi to generate code!

You can also use `misoapi` to generate all these code for you! Add following comments on you func declaration, then run `misoapi` to generate.

E.g.,

```go
// Streaming file.
//
//   - misoapi-raw
//   - misoapi-http: GET /file/stream
//   - misoapi-scope: PUBLIC
//   - misoapi-query-doc: key: temporary file key
//   - misoapi-desc: Media streaming using temporary file key, the file_key's ttl is extended with each subsequent
//     request. This endpoint is expected to be accessible publicly without authorization, since a temporary
//     file_key is generated and used.
func ApiTempKeyStreamFile(inb *miso.Inbound) {
    // ...
}
```

After running `misoapi`, we then have

```go
miso.RawGet("/file/stream", ApiTempKeyStreamFile).
    Extra(miso.ExtraName, "ApiTempKeyStreamFile").
    Desc(`Media streaming using temporary file key, the file_key's ttl is extended with each subsequent request. This endpoint is expected to be accessible publicly without authorization, since a temporary file_key is generated and used.`).
    Public().
    DocQueryParam("key", "temporary file key")
```

For more on misoapi, have a look at [Tools](./tools.md).