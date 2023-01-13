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

	"github.com/curtisnewbie/gocommon/common"
	"github.com/sirupsen/logrus"
)

// Helper type for handling HTTP responses
type TResponse struct {
	Resp *http.Response
	Err  error
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
		logrus.Errorf("Failed to unmarchal '%s' as %v, %v", string(body), reflect.TypeOf(ptr), e)
		return e
	}
	return nil
}

// Helper type for sending HTTP requests
//
// Provides convenients methods to build requests, use http.Client and propagate tracing information
type TClient struct {
	// request url
	Url string
	// request headers
	Headers map[string][]string
	// context provided by caller
	ctx context.Context
	// enable tracing
	Trace bool
	// http client used
	client *http.Client
}

// Enable tracing by putting propagation key/value pairs on http headers
func (t *TClient) EnableTracing() *TClient {
	t.Trace = true
	return t
}

// Send GET request
func (t *TClient) Get(queryParams map[string][]string) *TResponse {
	url := concatQueryParam(t.Url, queryParams)
	req, e := NewGetRequest(url)
	if e != nil {
		return &TResponse{Resp: nil, Err: e}
	}
	return t.send(req)
}

// Send POST request with JSON
func (t *TClient) PostJson(body any) *TResponse {
	jsonBody, e := json.Marshal(body)
	if e != nil {
		return &TResponse{Resp: nil, Err: e}
	}
	t.AddHeaders(map[string]string{
		"content-type": "application/json",
	})
	return t.Post(bytes.NewReader(jsonBody))
}

// Send POST request
func (t *TClient) Post(body io.Reader) *TResponse {
	req, e := NewPostRequest(t.Url, body)
	if e != nil {
		return &TResponse{Resp: nil, Err: e}
	}
	return t.send(req)
}

// Send PUT request with JSON
func (t *TClient) PutJson(body any) *TResponse {
	jsonBody, e := json.Marshal(body)
	if e != nil {
		return &TResponse{Resp: nil, Err: e}
	}
	t.AddHeaders(map[string]string{
		"content-type": "application/json",
	})
	return t.Put(bytes.NewReader(jsonBody))
}

// Send PUT request
func (t *TClient) Put(body io.Reader) *TResponse {
	req, e := NewPutRequest(t.Url, body)
	if e != nil {
		return &TResponse{Resp: nil, Err: e}
	}
	return t.send(req)
}

// Send DELETE request
func (t *TClient) Delete(queryParams map[string][]string) *TResponse {
	url := concatQueryParam(t.Url, queryParams)
	req, e := NewDeleteRequest(url)
	if e != nil {
		return &TResponse{Resp: nil, Err: e}
	}
	return t.send(req)
}

// Send request
func (t *TClient) send(req *http.Request) *TResponse {
	if t.Trace {
		req = TraceRequest(t.ctx, req)
	}

	AddHeaders(req, t.Headers)

	logrus.Infof("TClient: %s '%s'", req.Method, req.URL)
	r, e := t.client.Do(req)
	return &TResponse{Resp: r, Err: e}
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
func NewDefaultTClient(ctx context.Context, url string) *TClient {
	return NewTClient(ctx, url, http.DefaultClient)
}

// Create new TClient
func NewTClient(ctx context.Context, url string, client *http.Client) *TClient {
	return &TClient{Url: url, Headers: map[string][]string{}, ctx: ctx, client: client}
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
