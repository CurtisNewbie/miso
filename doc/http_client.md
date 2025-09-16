# HTTP Client

miso provides builtin HTTP Client for sending requests to external services and other microservices within the cluster.

Use `miso.NewClient` func to create a new Client.

E.g., the following example creates a Client to send `POST application/json` request to url `https://somewebsite/open/api/echo`, and then automatically unmarshals response JSON to `res MyResult`.

```go
type MyRequest struct {
    SomeId string
    SomeNum float64
}

var res MyResult
var err error = miso.NewClient(rail, "https://somewebsite/open/api/echo").
    PostJson(MyRequest{SomeId: "id_123", SomeNum: 123.45}).
    Json(&res)
```

The created Client is mainly a thin wrapper of `net/http.Client`, you should not reuse `miso.Client`. By default, all `miso.Client` share the same `net/http.Client` for performance. (See `miso.MisoDefaultClient`).

You can change the underlying `http.Client` as follows using `Client.UseClient`:

```go
var myClient *http.Client

var res TriggerResult
var err error = miso.NewClient(rail, "https://somewebsite/open/api/echo").
    UseClient(myClient).
    PostJson(MyRequest{SomeId: "id_123", SomeNum: 123.45}).
    Json(&res)
```

Just make sure to reuse the same \*http.Client as much as possible.

If service discovery is enabled by enabling `Consul` or `Nacos` Module, the created Client will automatically route requests to one of the available instance by the given service name.

E.g., the following example creates a Client to send `POST application/json` request to one of the instance of `workflow-engine` with the reletive path url `/open/api/engine`, and then automatically unmarshals response JSON to `res TriggerResult`.

```go
var res TriggerResult
var err error = miso.NewDynClient(rail, "/open/api/engine", "workflow-engine" /* service name */).
    PostJson(TriggerWorkFlow{WorkFlowId: "123"}).
    Json(&res)
```

By default, requests are routed randomly to one of the instance. However, load-balancing strategy is not yet customizable for now.

Essentially, `miso.NewDynClient()` is just a helper func for the following code:

```go

// Create new defualt Client with EnableServiceDiscovery(), EnableTracing(), and Require2xx() turned on.
//
// The provided relUrl should be a relative url starting with '/'.
func NewDynClient(rail Rail, relUrl string, serviceName string) *Client {
	return NewClient(rail, relUrl).
		EnableServiceDiscovery(serviceName).
		EnableTracing().
		Require2xx()
}
```

You can always use `miso.NewClient` and then cutomize the `Client` objects using the chaining methods, e.g.,

```go
func (t *Client) AddAuthBearer(v string) *Client
func (t *Client) AddAuthHeader(v string) *Client
func (t *Client) AddHeader(k string, v string) *Client
func (t *Client) AddHeaders(headers map[string]string) *Client
func (t *Client) AddQueryParams(k string, v ...string) *Client
func (t *Client) EnableServiceDiscovery(serviceName string) *Client
func (t *Client) EnableTracing() *Client
func (t *Client) Http() *Client
func (t *Client) Https() *Client
func (t *Client) LogBody() *Client
func (t *Client) Require2xx() *Client
func (t *Client) SetContentType(ct string) *Client
func (t *Client) SetHeaders(k string, v ...string) *Client
func (t *Client) UseClient(client *http.Client) *Client
```

After your customization, use following methods to trigger the HTTP request:

```go
func (t *Client) Connect() *TResponse
func (t *Client) Delete() *TResponse
func (t *Client) Get() *TResponse
func (t *Client) Head() *TResponse
func (t *Client) Options() *TResponse
func (t *Client) Patch() *TResponse
func (t *Client) Post(body io.Reader) *TResponse
func (t *Client) PostBytes(body []byte) *TResponse
func (t *Client) PostForm(data url.Values) *TResponse
func (t *Client) PostFormData(data map[string]io.Reader) *TResponse
func (t *Client) PostJson(body any) *TResponse
func (t *Client) Put(body io.Reader) *TResponse
func (t *Client) PutBytes(body []byte) *TResponse
func (t *Client) PutForm(data url.Values) *TResponse
func (t *Client) PutFormData(data map[string]io.Reader) *TResponse
func (t *Client) PutJson(body any) *TResponse
func (t *Client) Trace() *TResponse
```

The returned `TResponse` is also a thin wrapper of the underlying HTTP response. Use it's methods to check response status and read response data.

```go
func (tr *TResponse) Close() error
func (tr *TResponse) Bytes() ([]byte, error)
func (tr *TResponse) Is2xx() bool
func (tr *TResponse) Json(ptr any) error
func (tr *TResponse) Require2xx() error
func (tr *TResponse) Sse(parse func(e sse.Event) (stop bool, err error), options ...func(c *SseReadConfig)) error
func (tr *TResponse) Str() (string, error)
func (tr *TResponse) WriteTo(writer io.Writer) (int64, error)
func (tr *TResponse) WriteToFile(path string) (int64, error)
```

Notice that `TResponse` also provides a `Close()` method, if you use one of the method of `TResponse` to read it's data, `Close()` is automatically called for you.
However, if none of these method is called, you should call `Close()` yourself.

E.g.,

```go
var res TriggerResult
var err error = miso.NewDynClient(rail, "/open/api/engine", "workflow-engine").
    PostJson(TriggerWorkFlow{WorkFlowId: "123"}).
    Json(&res) // <----------------------------------- Close() is called here
```

More specifically, inside `Json()` method:

```go
// Read response as JSON object.
//
// Response is always closed automatically.
//
// If response body is somehow empty, *miso.NoneErr is returned.
func (tr *TResponse) Json(ptr any) error {
	defer tr.Close() // <------------------------------ Close() is always called
	if tr.Err != nil {
		return tr.Err
	}
	if tr.Resp.Body == nil {
		return NoneErr
	}

	body, e := io.ReadAll(tr.Resp.Body)
	if e != nil {
		return WrapErr(e)
	}
	tr.logRespBody(body)

	if e = json.ParseJson(body, ptr); e != nil {
		s := util.UnsafeByt2Str(body)
		return errs.WrapErrf(e, "failed to unmarshal json from response, body: %v", s)
	}

	if v, ok := ptr.(TResponseJsonCheckErr); ok && v != nil {
		if err := v.CheckErr(); err != nil {
			return WrapErr(err)
		}
	}

	return nil
}
```

`miso.Client` also supports tracing if `EnableTracing()` is called. See [trace.md](./trace.md) for more about tracing.
