package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/curtisnewbie/miso/consul"
	"github.com/curtisnewbie/miso/core"
)

const (
	formEncoded     = "application/x-www-form-urlencoded"
	applicationJson = "application/json"
	contentType     = "Content-Type"
)

// Helper type for handling HTTP responses
type TResponse struct {
	ExecCtx    core.Rail
	Ctx        context.Context
	Resp       *http.Response
	RespHeader http.Header
	StatusCode int
	Err        error
}

// Close Response
func (tr *TResponse) Close() error {
	return tr.Resp.Body.Close()
}

// Read response as []bytes, response is always closed automatically
func (tr *TResponse) ReadBytes() ([]byte, error) {
	defer tr.Close()
	return io.ReadAll(tr.Resp.Body)
}

// Read response as string, response is always closed automatically
func (tr *TResponse) ReadStr() (string, error) {
	defer tr.Close()
	b, e := io.ReadAll(tr.Resp.Body)
	if e != nil {
		return "", e
	}
	return string(b), nil
}

// Read response as JSON object, response is always closed automatically
func (tr *TResponse) ReadJson(ptr any) error {
	defer tr.Close()
	body, e := io.ReadAll(tr.Resp.Body)
	if e != nil {
		return e
	}

	if e = json.Unmarshal(body, ptr); e != nil {
		core.TraceLogger(tr.Ctx).Errorf("Failed to unmarchal '%s' as %v, %v", string(body), reflect.TypeOf(ptr), e)
		return e
	}
	return nil
}

// Is status code 2xx
func (tr *TResponse) Is2xx() bool {
	return tr.StatusCode >= 200 && tr.StatusCode < 300
}

// Check if it's 2xx, else return error
func (tr *TResponse) Require2xx() error {
	if !tr.Is2xx() {
		var body string
		byt, err := tr.ReadBytes()
		if err == nil {
			body = string(byt)
		}
		return fmt.Errorf("unknown error, status code: %v, body: %v", tr.StatusCode, body)
	}
	return nil
}

// Read response as GnResp[T] object, response is always closed automatically
func ReadGnResp[T any](tr *TResponse) (core.GnResp[T], error) {
	var gr core.GnResp[T]
	e := tr.ReadJson(&gr)
	return gr, e
}

// Helper type for sending HTTP requests
//
// Provides convenients methods to build requests, use http.Client and propagate tracing information
type TClient struct {
	Url        string              // request url (absolute or relative)
	Headers    map[string][]string // request headers
	ExecCtx    core.Rail         // execute context
	Ctx        context.Context     // context provided by caller
	QueryParam map[string][]string // query parameters

	client          *http.Client
	serviceName     string
	trace           bool
	discoverService bool
}

// Prepare request url.
//
// If service discovery is enabled, serviceName will be resolved using Consul.
//
// If consul is disabled, t.serviceName is used directly as the host name. This is especially useful in container environment.
func (t *TClient) prepReqUrl() (string, error) {
	if t.discoverService && consul.IsConsulClientInitialized() {
		url, err := consul.ResolveRequestUrl(t.serviceName, t.Url)
		if err != nil {
			return "", err
		}
		url = concatQueryParam(url, t.QueryParam)
		return url, nil
	}

	url := t.Url
	if t.serviceName != "" {
		host := resolveHostFromProp(t.serviceName)
		if host != "" {
			url = "http://" + host + t.Url
		} else {
			url = "http://" + t.serviceName + t.Url
		}
	}

	url = concatQueryParam(url, t.QueryParam)
	return url, nil
}

/*
Enable service discovery, the Url specified when creating the TClient must be relative, starting with '/'

For example:

	tr := NewDynTClient(ctx, "/open/api/gallery/brief/owned", "fantahsea").
		AddHeaders(map[string]string{
			"TestCase": "TestGet",
		}).
		EnableTracing().
		Get()

The resolved request url will be (imagine that the service 'fantahsea' is hosted on 'localhost:8081'):

	"http://localhost:8081/open/api/gallery/brief/owned?name=yongj.zhuang&name=zhuangyongj&age=103"
*/
func (t *TClient) EnableServiceDiscovery(serviceName string) *TClient {
	t.serviceName = serviceName
	t.discoverService = true
	return t
}

// Enable tracing by putting propagation key/value pairs on http headers
func (t *TClient) EnableTracing() *TClient {
	t.trace = true
	return t
}

// Set Content-Type
func (t *TClient) SetContentType(ct string) *TClient {
	t.SetHeaders(contentType, ct)
	return t
}

// Send GET request
func (t *TClient) Get() *TResponse {
	u, e := t.prepReqUrl()
	if e != nil {
		return t.errorResponse(e)
	}

	req, e := NewGetRequest(u)
	if e != nil {
		return t.errorResponse(e)
	}
	return t.send(req)
}

// Send POST request with urlencoded form data
func (t *TClient) PostForm(data url.Values) *TResponse {
	t.SetContentType(formEncoded)
	return t.Post(strings.NewReader(data.Encode()))
}

// Send POST request with JSON
func (t *TClient) PostJson(body any) *TResponse {
	jsonBody, e := json.Marshal(body)
	if e != nil {
		return t.errorResponse(e)
	}
	t.SetContentType(applicationJson)
	return t.Post(bytes.NewReader(jsonBody))
}

func (t *TClient) errorResponse(e error) *TResponse {
	return &TResponse{Err: e, Ctx: t.Ctx, ExecCtx: t.ExecCtx}
}

// Send POST request
func (t *TClient) Post(body io.Reader) *TResponse {
	u, e := t.prepReqUrl()
	if e != nil {
		return t.errorResponse(e)
	}

	req, e := NewPostRequest(u, body)
	if e != nil {
		return t.errorResponse(e)
	}
	return t.send(req)
}

// Send PUT request with JSON
func (t *TClient) PutJson(body any) *TResponse {
	jsonBody, e := json.Marshal(body)
	if e != nil {
		return t.errorResponse(e)
	}
	t.SetContentType(applicationJson)
	return t.Put(bytes.NewReader(jsonBody))
}

// Send PUT request
func (t *TClient) Put(body io.Reader) *TResponse {
	u, e := t.prepReqUrl()
	if e != nil {
		return t.errorResponse(e)
	}

	req, e := NewPutRequest(u, body)
	if e != nil {
		return t.errorResponse(e)
	}
	return t.send(req)
}

// Send DELETE request
func (t *TClient) Delete() *TResponse {
	u, e := t.prepReqUrl()
	if e != nil {
		return t.errorResponse(e)
	}

	req, e := NewDeleteRequest(u)
	if e != nil {
		return t.errorResponse(e)
	}
	return t.send(req)
}

// Send HEAD request
func (t *TClient) Head() *TResponse {
	u, e := t.prepReqUrl()
	if e != nil {
		return t.errorResponse(e)
	}

	req, e := NewHeadRequest(u)
	if e != nil {
		return t.errorResponse(e)
	}
	return t.send(req)
}

// Send OPTIONS request
func (t *TClient) Options() *TResponse {
	u, e := t.prepReqUrl()
	if e != nil {
		return t.errorResponse(e)
	}

	req, e := NewOptionsRequest(u)
	if e != nil {
		return t.errorResponse(e)
	}
	return t.send(req)
}

// Send request
func (t *TClient) send(req *http.Request) *TResponse {
	if t.trace {
		req = TraceRequest(t.Ctx, req)
	}

	AddHeaders(req, t.Headers)

	t.ExecCtx.Debugf("%v %v, Headers: %v", req.Method, req.URL, req.Header)

	r, e := t.client.Do(req) // send HTTP requests

	var statusCode int
	var respHeaders http.Header
	if e == nil && r != nil {
		statusCode = r.StatusCode
		respHeaders = r.Header
	}

	return &TResponse{Resp: r, Err: e, Ctx: t.Ctx, ExecCtx: t.ExecCtx, StatusCode: statusCode, RespHeader: respHeaders}
}

// Append headers, subsequent method calls doesn't override previously appended headers
func (t *TClient) AddHeaders(headers map[string]string) *TClient {
	for k, v := range headers {
		if t.Headers[k] == nil {
			t.Headers[k] = []string{v}
		} else {
			t.Headers[k] = append(t.Headers[k], v)
		}
	}
	return t
}

// Append header, subsequent method calls doesn't override previously appended headers
func (t *TClient) AddHeader(k string, v string) *TClient {
	if t.Headers[k] == nil {
		t.Headers[k] = []string{v}
	} else {
		t.Headers[k] = append(t.Headers[k], v)
	}
	return t
}

// Overwrite header
func (t *TClient) SetHeaders(k string, v ...string) *TClient {
	t.Headers[k] = v
	return t
}

// Append Query Parameters, subsequent method calls doesn't override previously appended parameters
func (t *TClient) AddQueryParams(k string, v ...string) *TClient {
	for i := range v {
		t.addQueryParam(k, v[i])
	}
	return t
}

// Append Query Parameters, subsequent method calls doesn't override previously appended parameters
func (t *TClient) addQueryParam(k string, v string) *TClient {
	if t.QueryParam[k] == nil {
		t.QueryParam[k] = []string{v}
	} else {
		t.QueryParam[k] = append(t.QueryParam[k], v)
	}
	return t
}

// Create new defualt TClient
func NewDefaultTClient(ec core.Rail, url string) *TClient {
	return NewTClient(ec, url, http.DefaultClient)
}

// Create new defualt TClient with service discovery enabled, relUrl should be a relative url starting with '/'
func NewDynTClient(ec core.Rail, relUrl string, serviceName string) *TClient {
	return NewTClient(ec, relUrl, http.DefaultClient).EnableServiceDiscovery(serviceName)
}

// Create new TClient
func NewTClient(ec core.Rail, url string, client *http.Client) *TClient {
	return &TClient{Url: url, Headers: map[string][]string{}, Ctx: ec.Ctx, client: client, ExecCtx: ec, QueryParam: map[string][]string{}}
}

// Concatenate url and query parameters
func concatQueryParam(url string, queryParams map[string][]string) string {
	qp := JoinQueryParam(queryParams)
	if len(qp) > 0 && !strings.HasSuffix(url, "?") {
		url = url + "?" + qp
	}
	return url
}

// Send GET request
func Get(url string, headers map[string][]string) (*http.Response, error) {
	req, e := NewGetRequest(url)
	if e != nil {
		return nil, e
	}

	AddHeaders(req, headers)
	return http.DefaultClient.Do(req)
}

// Add http headers
func AddHeaders(req *http.Request, headers map[string][]string) {
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
}

// Send POST request
func PostJson(url string, json string) (*http.Response, error) {
	body := bytes.NewBuffer([]byte(json))
	return Post(url, body)
}

// Send POST request
func Post(url string, body io.Reader) (*http.Response, error) {
	req, e := NewPostRequest(url, body)
	if e != nil {
		return nil, e
	}
	return http.DefaultClient.Do(req)
}

// Create HEAD request
func NewHeadRequest(url string) (*http.Request, error) {
	req, e := http.NewRequest(http.MethodHead, url, nil)
	if e != nil {
		return nil, e
	}
	return req, e
}

// Create OPTIONS request
func NewOptionsRequest(url string) (*http.Request, error) {
	req, e := http.NewRequest(http.MethodOptions, url, nil)
	if e != nil {
		return nil, e
	}
	return req, e
}

// Create DELETE request
func NewDeleteRequest(url string) (*http.Request, error) {
	req, e := http.NewRequest(http.MethodDelete, url, nil)
	if e != nil {
		return nil, e
	}
	return req, e
}

// Create PUT request
func NewPutRequest(url string, body io.Reader) (*http.Request, error) {
	req, e := http.NewRequest(http.MethodPut, url, body)
	if e != nil {
		return nil, e
	}
	return req, e
}

// Create POST request
func NewPostRequest(url string, body io.Reader) (*http.Request, error) {
	req, e := http.NewRequest(http.MethodPost, url, body)
	if e != nil {
		return nil, e
	}
	return req, e
}

// Create GET request
func NewGetRequest(url string) (*http.Request, error) {
	req, e := http.NewRequest(http.MethodGet, url, nil)
	if e != nil {
		return nil, e
	}
	return req, e
}

// Wraper request with tracing key/value pairs on http headers
func TraceRequest(ctx context.Context, req *http.Request) *http.Request {
	if ctx == nil {
		return req
	}

	for _, key := range core.GetPropagationKeys() {
		v := ctx.Value(key)
		if v != nil {
			if vstr, ok := v.(string); ok {
				req.Header.Set(key, vstr)
			} else {
				req.Header.Set(key, fmt.Sprintf("%v", v))
			}
		}
	}
	return req
}

// Join query parameters
func JoinQueryParam(queryParams map[string][]string) string {
	if queryParams == nil {
		return ""
	}

	seg := []string{}
	for k, vs := range queryParams {
		for i := range vs {
			seg = append(seg, fmt.Sprintf("%s=%s", k, url.QueryEscape(vs[i])))
		}
	}
	return strings.Join(seg, "&")
}

func resolveHostFromProp(name string) string {
	if name == "" {
		return ""
	}
	return core.GetPropStr("client.host." + name)
}
