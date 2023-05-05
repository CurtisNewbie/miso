package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/consul"
)

// Helper type for handling HTTP responses
type TResponse struct {
	ExecCtx common.ExecContext
	Ctx     context.Context
	Resp    *http.Response
	Err     error
}

// Close Response
func (tr *TResponse) Close() error {
	return tr.Resp.Body.Close()
}

// Read response as []bytes
func (tr *TResponse) ReadBytes() ([]byte, error) {
	return io.ReadAll(tr.Resp.Body)
}

// Read response as string
func (tr *TResponse) ReadStr() (string, error) {
	b, e := tr.ReadBytes()
	if e != nil {
		return "", e
	}
	return string(b), nil
}

// Read response as JSON object
func (tr *TResponse) ReadJson(ptr any) error {
	body, e := tr.ReadBytes()
	if e != nil {
		return e
	}

	if e = json.Unmarshal(body, ptr); e != nil {
		common.TraceLogger(tr.Ctx).Errorf("Failed to unmarchal '%s' as %v, %v", string(body), reflect.TypeOf(ptr), e)
		return e
	}
	return nil
}

// Helper type for sending HTTP requests
//
// Provides convenients methods to build requests, use http.Client and propagate tracing information
type TClient struct {
	Url             string              // request url (absolute or relative)
	Headers         map[string][]string // request headers
	ExecCtx         common.ExecContext  // execute context
	Ctx             context.Context     // context provided by caller
	client          *http.Client        // http client used
	serviceName     string              // service name
	trace           bool                // enable tracing
	logRequest      bool                // whether requests are logged
	discoverService bool                // is service discovery enabled
}

// Prepare request url, if service discovery is enabled, serviceName will be resolved (currently supported by Consul)
func (t *TClient) prepReqUrl() (string, error) {
	if t.discoverService {
		return consul.ResolveRequestUrl(t.serviceName, t.Url)
	}
	return t.Url, nil
}

/*
	Enable service discovery, the Url specified when creating the TClient must be relative, starting with '/'

	For example:

		tr := NewDynTClient(ctx, "/open/api/gallery/brief/owned", "fantahsea").
			AddHeaders(map[string]string{
				"TestCase": "TestGet",
			}).
			EnableTracing().
			Get(map[string][]string{
				"name": {"yongj.zhuang", "zhuangyongj"},
				"age":  {"103"},
			})

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

// Enable request logging
func (t *TClient) EnableRequestLog() *TClient {
	t.logRequest = true
	return t
}

// Send GET request
func (t *TClient) Get(queryParams map[string][]string) *TResponse {
	u, e := t.prepReqUrl()
	if e != nil {
		return t.errorResponse(e)
	}

	u = concatQueryParam(u, queryParams)
	req, e := NewGetRequest(u)
	if e != nil {
		return t.errorResponse(e)
	}
	return t.send(req)
}

// Send POST request with JSON
func (t *TClient) PostJson(body any) *TResponse {
	jsonBody, e := json.Marshal(body)
	if e != nil {
		return t.errorResponse(e)
	}
	t.AddHeaders(map[string]string{
		"content-type": "application/json",
	})
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
	t.AddHeaders(map[string]string{
		"content-type": "application/json",
	})
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
func (t *TClient) Delete(queryParams map[string][]string) *TResponse {
	u, e := t.prepReqUrl()
	if e != nil {
		return t.errorResponse(e)
	}

	u = concatQueryParam(u, queryParams)
	req, e := NewDeleteRequest(u)
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

	var start time.Time
	if t.logRequest {
		start = time.Now()
	}

	r, e := t.client.Do(req) // send HTTP requests

	if t.logRequest {
		if req.Body != nil {
			t.ExecCtx.Log.Infof("%s '%s' (%s), Body: %v, Headers: %v", req.Method, req.URL, time.Since(start), req.Body, req.Header)
		} else {
			t.ExecCtx.Log.Infof("%s '%s' (%s), Headers: %v", req.Method, req.URL, time.Since(start), req.Header)
		}
	}
	return &TResponse{Resp: r, Err: e, Ctx: t.Ctx, ExecCtx: t.ExecCtx}
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

// Create new defualt TClient
func NewDefaultTClient(ec common.ExecContext, url string) *TClient {
	return NewTClient(ec, url, http.DefaultClient)
}

// Create new defualt TClient with service discovery enabled, relUrl should be a relative url starting with '/'
func NewDynTClient(ec common.ExecContext, relUrl string, serviceName string) *TClient {
	return NewTClient(ec, relUrl, http.DefaultClient).EnableServiceDiscovery(serviceName)
}

// Create new TClient
func NewTClient(ec common.ExecContext, url string, client *http.Client) *TClient {
	return &TClient{Url: url, Headers: map[string][]string{}, Ctx: ec.Ctx, client: client, ExecCtx: ec}
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

	for _, key := range common.GetPropagationKeys() {
		v := ctx.Value(key)
		if v != nil {
			if vstr, ok := v.(string); ok {
				req.Header.Set(key, vstr)
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
		for _, v := range vs {
			seg = append(seg, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return strings.Join(seg, "&")
}
