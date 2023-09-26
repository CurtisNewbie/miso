package miso

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	formEncoded     = "application/x-www-form-urlencoded"
	applicationJson = "application/json"
	textPlain       = "text/plain"
	contentType     = "Content-Type"

	httpProto  = "http://"
	httpsProto = "https://"
)

var (
	_serviceRegistry         ServiceRegistry = nil
	_initServiceRegistryOnce sync.Once
	defaultClient            *http.Client
)

func init() {
	defaultClient = &http.Client{Timeout: 5 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 500
	transport.MaxIdleConnsPerHost = 500
	transport.IdleConnTimeout = time.Minute * 5
	defaultClient.Transport = transport
}

// Helper type for handling HTTP responses
type TResponse struct {
	Rail       Rail
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

// Read response as []bytes.
//
// Response is always closed automatically.
func (tr *TResponse) Bytes() ([]byte, error) {
	if tr.Err != nil {
		return nil, tr.Err
	}

	defer tr.Close()
	return io.ReadAll(tr.Resp.Body)
}

// Read response as string.
//
// Response is always closed automatically.
func (tr *TResponse) Str() (string, error) {
	if tr.Err != nil {
		return "", tr.Err
	}

	defer tr.Close()
	b, e := io.ReadAll(tr.Resp.Body)
	if e != nil {
		return "", e
	}
	return string(b), nil
}

// Read response as JSON object.
//
// Response is always closed automatically.
func (tr *TResponse) Json(ptr any) error {
	if tr.Err != nil {
		return tr.Err
	}

	defer tr.Close()
	body, e := io.ReadAll(tr.Resp.Body)
	if e != nil {
		return e
	}

	if e = json.Unmarshal(body, ptr); e != nil {
		s := string(body)
		errMsg := fmt.Sprintf("Failed to unmarshal json from response, body: %v, %v", s, e)
		tr.Rail.Error(errMsg)
		return fmt.Errorf(errMsg)
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
		byt, err := tr.Bytes()
		if err == nil {
			body = string(byt)
		}
		return fmt.Errorf("unknown error, status code: %v, body: %v", tr.StatusCode, body)
	}
	return nil
}

// Helper type for sending HTTP requests
//
// Provides convenients methods to build requests, use http.Client and propagate tracing information
type TClient struct {
	Url        string              // request url (absolute or relative)
	Headers    map[string][]string // request headers
	Ctx        context.Context     // context provided by caller
	QueryParam map[string][]string // query parameters
	Rail       Rail                // rail

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
	url := t.Url

	if t.discoverService {
		sr := GetServiceRegistry()
		if sr == nil {
			return "", errors.New("service discovery enabled, but no service registry available")
		}

		resolved, err := sr.resolve(t.serviceName, t.Url)
		if err != nil {
			t.Rail.Errorf("Resolve service address failed, service: %v, %v", t.serviceName, err)
			return "", err
		}
		url = resolved
	}

	return concatQueryParam(url, t.QueryParam), nil
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

// Send POST request with JSON.
//
// Use simple types like struct instad of pointer for body.
func (t *TClient) PostJson(body any) *TResponse {
	ptr := body
	if reflect.TypeOf(body).Kind() != reflect.Pointer {
		ptr = &body
	}

	jsonBody, e := json.Marshal(ptr)
	if e != nil {
		return t.errorResponse(e)
	}
	t.SetContentType(applicationJson)
	return t.Post(bytes.NewReader(jsonBody))
}

func (t *TClient) errorResponse(e error) *TResponse {
	return &TResponse{Err: e, Ctx: t.Ctx, Rail: t.Rail}
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

	if t.Rail.IsDebugLogEnabled() {
		loggedBody := "***"
		if req.Body != nil && req.GetBody != nil {
			if v, ok := t.Headers[contentType]; ok && len(v) > 0 && contentTypeLoggable(v[0]) {
				if copy, err := req.GetBody(); err == nil && copy != nil {
					if buf, e := io.ReadAll(copy); e == nil {
						loggedBody = string(buf)
					}
				}
			}
		}
		t.Rail.Debugf("%v %v, Headers: %v, Body: %v", req.Method, req.URL, req.Header, loggedBody)
	}

	r, e := t.client.Do(req) // send HTTP requests

	var statusCode int
	var respHeaders http.Header
	if e == nil && r != nil {
		statusCode = r.StatusCode
		respHeaders = r.Header
	}

	return &TResponse{Resp: r, Err: e, Ctx: t.Ctx, Rail: t.Rail, StatusCode: statusCode, RespHeader: respHeaders}
}

func contentTypeLoggable(contentType string) bool {
	lct := strings.ToLower(contentType)
	return lct == applicationJson || lct == textPlain
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
func NewDefaultTClient(ec Rail, url string) *TClient {
	return NewTClient(ec, url, defaultClient)
}

// Create new defualt TClient with service discovery enabled, relUrl should be a relative url starting with '/'
func NewDynTClient(ec Rail, relUrl string, serviceName string) *TClient {
	return NewTClient(ec, relUrl, defaultClient).EnableServiceDiscovery(serviceName)
}

// Create new TClient
func NewTClient(rail Rail, url string, client *http.Client) *TClient {
	return &TClient{Url: url, Headers: map[string][]string{}, Ctx: rail.Ctx, client: client, Rail: rail, QueryParam: map[string][]string{}}
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
func SendGet(url string, headers map[string][]string) (*http.Response, error) {
	req, e := NewGetRequest(url)
	if e != nil {
		return nil, e
	}

	AddHeaders(req, headers)
	return defaultClient.Do(req)
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
	return SendPost(url, body)
}

// Send POST request
func SendPost(url string, body io.Reader) (*http.Response, error) {
	req, e := NewPostRequest(url, body)
	if e != nil {
		return nil, e
	}
	return defaultClient.Do(req)
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

	for _, key := range GetPropagationKeys() {
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
	return GetPropStr("client.host." + name)
}

// Service registry
type ServiceRegistry interface {

	// Resolve request url dynamically based on the services discovered
	resolve(service string, relativeUrl string) (string, error)
}

// Service registry based on Consul
type consulServiceRegistry struct {
}

func (r consulServiceRegistry) resolve(service string, relativeUrl string) (string, error) {
	return ConsulResolveRequestUrl(service, relativeUrl)
}

// Service registry backed by loaded configuration
type hardcodedServiceRegistry struct {
}

func (r hardcodedServiceRegistry) resolve(service string, relativeUrl string) (string, error) {
	if IsBlankStr(service) {
		return "", fmt.Errorf("service name is required")
	}

	host := resolveHostFromProp(service)
	if host != "" {
		return httpProto + host + relativeUrl, nil
	}

	return httpProto + service + relativeUrl, nil // use the
}

// Get service registry
//
// Service registry initialization is lazy, don't call this for global var
func GetServiceRegistry() ServiceRegistry {
	_initServiceRegistryOnce.Do(func() {
		rail := EmptyRail()
		if IsConsulClientInitialized() {
			_serviceRegistry = consulServiceRegistry{}
			rail.Debug("Detected Consul client, using consulServiceRegistry")
			return
		}

		// fallback to configuration based
		_serviceRegistry = hardcodedServiceRegistry{}
		rail.Debug("No dynamic service registry detected, fallback to hardcodedServiceRegistry")
	})

	return _serviceRegistry
}
