# Tracing

Tracing in golang must be hardcoded. Trace info are passed around in infrastructure and application code. This is mainly achieved using `context.Context`. In miso, we don't pass around `context.Context` explicitly, instead we pass around `miso.Rail`, a `context.Context` value is stored in `miso.Rail` though.

It's very common to write code like the following when using miso:

```go
func CreateGallery(rail miso.Rail, cmd CreateGalleryCmd, tx *gorm.DB) (*Gallery, error) {
	rail.Infof("Creating gallery, cmd: %#v", cmd)

    // ...
}
```

Whenever you write code to send http request or MQ messages, you propagate the `miso.Rail` object to miso's api. For example, in the following code, we want to call `fstore` service's `/file/info` endpoint. We propagate our `miso.Rail` to `miso.NewDynClient`, the trace info is extracted internally by miso, and sent to the `fstore` service in forms of HTTP headers. It might be different when we are using other transports, e..g, sending MQ messages, but the idea remain the same.

```go
func FetchFstoreFileInfo(rail miso.Rail, fileId string, uploadFileId string) (FstoreFile, error) {
	var r miso.GnResp[FstoreFile]
	err := miso.NewDynClient(rail, "/file/info", "fstore").
		Require2xx().
		AddQueryParams("fileId", fileId).
		AddQueryParams("uploadFileId", uploadFileId).
		Get().
		Json(&r)
	if err != nil {
		return FstoreFile{}, fmt.Errorf("failed to fetch mini-fstore fileInfo, %v", err)
	}
	return r.Res()
}
```

In `fstore` service, we use miso's api to declare HTTP endpoint. When new request arrive, miso internally parses the HTTP headers and build the `miso.Rail` object automatically:

```go
// declare endpoint
miso.HttpGet("/file/info", miso.AutoHandler(ApiGetFileInfo)).Desc("Fetch file info")

// the handler
func ApiGetFileInfo(inb *miso.Inbound, req FileInfoReq) (api.FstoreFile, error) {
	rail := inb.Rail()
	rail.Info("Got the trace!")
    // ...
}
```

Since `miso.Rail` internally wraps the `context.Context` value, you are free to unwrap it if necessary.

```go
var rail miso.Rail

// unwrap the internal context
context := rail.Context()
```

`miso.Rail` also implements most of the commonly used logging methods (powered by logrus). You can call these methods directly using Rail. By default, the trace_id and span_id of current `miso.Rail` are also logged.

E.g.,

```go
rail.Info("Got the trace!")
rail.Infof("I am %v", name)
rail.Warnf("Something goes wrong!, %v", err)
```

With the default log formatter, the log looks like the following, the trace_id is `'lwmyiuuqywqgtxas'` and the span_id is `'xygskuilruvwtsay'`.

```log
2024-03-04 23:43:14.544 INFO  [lwmyiuuqywqgtxas,xygskuilruvwtsay]  miso.SchedulerBootstrap       : Cron Scheduler started
```

By default, the trace_id and span_id are represented by key `'X-B3-TraceId'` and `'X-B3-SpanId'`. Not only the trace_id and span_id are propagated. The trace info that you want to include can also be customized. You can specify propagation keys through APIs or configuration properties.

E.g.,

```go
miso.AddPropagationKeys("x-my-key")
```

or specify the keys in your conf.yaml:

```yml
tracing.propagation.keys:
    - "x-my-key"
```

Sometimes, you don't want to accept any trace info from the inbound request, e.g., when current app is a gateway. You can disable it using following property:

```yml
server:
    trace.inbound.propagate: false
```

But you should still be careful if you are doing proxy stuff, inbound requests may contain headers that are same as your propagation keys. This may trick the proxied service
as if these headers values are actually part of the trace. You may filter them explictly as the following:

```go
// propagate all headers to client
for k, arr := range r.Header {
    // the inbound request may contain headers that are one of our propagation keys
    // this can be a security problem
    if propagationKeys.Has(k) {
        continue
    }
    for _, v := range arr {
        client.AddHeader(k, v)
    }
}
```
