package miso

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/curtisnewbie/miso/encoding/json"
	"github.com/curtisnewbie/miso/util"
	"github.com/spf13/cast"
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
	httpProtoRegex = regexp.MustCompile(`(?i)https?://`)

	MisoDefaultClient *http.Client
)

func init() {
	MisoDefaultClient = &http.Client{Timeout: 15 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 500
	transport.MaxIdleConnsPerHost = 50
	transport.IdleConnTimeout = time.Minute * 5
	MisoDefaultClient.Transport = transport
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

// Write the response data to the given writer.
//
// Response is always closed automatically.
//
// If response body is somehow empty, *miso.NoneErr is returned.
func (tr *TResponse) WriteTo(writer io.Writer) (int64, error) {
	if tr.Err != nil {
		return 0, tr.Err
	}
	if tr.Resp.Body == nil {
		return 0, NoneErr
	}

	defer tr.Close()
	n, err := io.Copy(writer, tr.Resp.Body)
	if err != nil {
		return 0, UnknownErr(err)
	}
	return n, nil
}

// Read response as []bytes.
//
// Response is always closed automatically.
//
// If response body is somehow empty, *miso.NoneErr is returned.
func (tr *TResponse) Bytes() ([]byte, error) {
	if tr.Err != nil {
		return nil, tr.Err
	}
	if tr.Resp.Body == nil {
		return nil, NoneErr
	}

	defer tr.Close()
	return io.ReadAll(tr.Resp.Body)
}

// Read response as string.
//
// Response is always closed automatically.
//
// If response body is somehow empty, *miso.NoneErr is returned.
func (tr *TResponse) Str() (string, error) {
	if tr.Err != nil {
		return "", tr.Err
	}
	if tr.Resp.Body == nil {
		return "", NoneErr
	}

	defer tr.Close()
	b, e := io.ReadAll(tr.Resp.Body)
	if e != nil {
		return "", UnknownErr(e)
	}
	return util.UnsafeByt2Str(b), nil
}

// Read response as JSON object.
//
// Response is always closed automatically.
//
// If response body is somehow empty, *miso.NoneErr is returned.
func (tr *TResponse) Json(ptr any) error {
	if tr.Err != nil {
		return tr.Err
	}
	if tr.Resp.Body == nil {
		return NoneErr
	}

	defer tr.Close()
	body, e := io.ReadAll(tr.Resp.Body)
	if e != nil {
		return UnknownErr(e)
	}

	if e = json.ParseJson(body, ptr); e != nil {
		s := util.UnsafeByt2Str(body)
		return UnknownErrf(e, "failed to unmarshal json from response, body: %v", s)
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
			body = util.UnsafeByt2Str(byt)
		}
		return ErrUnknownError.WithInternalMsg("unknown error, status code: %v, body: %v", tr.StatusCode, body)
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
	require2xx      bool
}

// Change the underlying *http.Client
func (t *TClient) UseClient(client *http.Client) *TClient {
	t.client = client
	return t
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
			return "", UnknownErrMsgf("service discovery enabled, but no service registry available")
		}

		resolved, err := sr.ResolveUrl(t.Rail, t.serviceName, t.Url)
		if err != nil {
			return "", UnknownErrf(err, "Resolve service address failed, service: %v", t.serviceName)
		}
		url = resolved
	}

	if !httpProtoRegex.MatchString(url) { // missing a protocol
		url = httpProto + url
	}
	return concatQueryParam(url, t.QueryParam), nil
}

// Requires response to have 2xx status code, if not, the *TResponse will contain error built for this specific reason.
func (t *TClient) Require2xx() *TClient {
	t.require2xx = true
	return t
}

// Enable service discovery
func (t *TClient) EnableServiceDiscovery(serviceName string) *TClient {
	t.serviceName = serviceName
	t.discoverService = true
	return t
}

// Enable tracing by putting propagation key/value pairs on http headers.
func (t *TClient) EnableTracing() *TClient {
	t.trace = true
	return t
}

// Set Content-Type
func (t *TClient) SetContentType(ct string) *TClient {
	t.SetHeaders(contentType, ct)
	return t
}

// Append 'http://' protocol.
//
// If service discovery is enabled, or the url contains http protocol already, this will be skipped.
func (t *TClient) Http() *TClient {
	if t.discoverService || httpProtoRegex.MatchString(t.Url) {
		return t
	}

	t.Url = httpProto + t.Url
	return t
}

// Append 'https://' protocol.
//
// If service discovery is enabled, or the url contains http protocol already, this will be skipped.
func (t *TClient) Https() *TClient {
	if t.discoverService || httpProtoRegex.MatchString(t.Url) {
		return t
	}

	t.Url = httpsProto + t.Url
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
	jsonBody, e := json.WriteJson(body)
	if e != nil {
		return t.errorResponse(e)
	}
	t.SetContentType(applicationJson)
	return t.Post(bytes.NewReader(jsonBody))
}

func (t *TClient) errorResponse(e error) *TResponse {
	return &TResponse{Err: e, Ctx: t.Ctx, Rail: t.Rail}
}

// Send POST request with reader to request body.
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

// Send POST request with bytes.
func (t *TClient) PostBytes(body []byte) *TResponse {
	return t.Post(bytes.NewReader(body))
}

// Send PUT request with JSON
func (t *TClient) PutJson(body any) *TResponse {
	jsonBody, e := json.WriteJson(body)
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

// Send PUT request with bytes.
func (t *TClient) PutBytes(body []byte) *TResponse {
	return t.Put(bytes.NewReader(body))
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

	if IsDebugLevel() {
		loggedBody := "***"
		if req.Body != nil && req.GetBody != nil {
			if v, ok := t.Headers[contentType]; ok && len(v) > 0 && contentTypeLoggable(v[0]) {
				if copy, err := req.GetBody(); err == nil && copy != nil {
					if buf, e := io.ReadAll(copy); e == nil {
						loggedBody = util.UnsafeByt2Str(buf)
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

	tr := &TResponse{Resp: r, Err: e, Ctx: t.Ctx, Rail: t.Rail, StatusCode: statusCode, RespHeader: respHeaders}

	// check http status code
	if tr.Err == nil && t.require2xx {
		tr.Err = tr.Require2xx()
	}

	return tr
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

// Create new defualt TClient with EnableServiceDiscovery(), EnableTracing(), and Require2xx() turned on.
//
// The provided relUrl should be a relative url starting with '/'.
func NewDynTClient(ec Rail, relUrl string, serviceName string) *TClient {
	return NewTClient(ec, relUrl).EnableServiceDiscovery(serviceName).EnableTracing().Require2xx()
}

// Create new TClient.
func NewTClient(rail Rail, url string) *TClient {
	return &TClient{
		Url: url, Headers: map[string][]string{}, Ctx: rail.Context(), client: MisoDefaultClient,
		Rail: rail, QueryParam: map[string][]string{},
	}
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
	return MisoDefaultClient.Do(req)
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
	body := bytes.NewBuffer(util.UnsafeStr2Byt(json))
	return SendPost(url, body)
}

// Send POST request
func SendPost(url string, body io.Reader) (*http.Response, error) {
	req, e := NewPostRequest(url, body)
	if e != nil {
		return nil, e
	}
	return MisoDefaultClient.Do(req)
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

	UsePropagationKeys(func(key string) {
		v := ctx.Value(key)
		if v != nil {
			if sv := cast.ToString(v); sv != "" {
				req.Header.Set(key, sv)
			}
		}
	})

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

// Disable TLS certificate check.
func ClientSkipTlsSecureCheck() {
	MisoDefaultClient.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify = true
}

// Set default http client timeout
func SetDefaultTimeout(ttl time.Duration) {
	MisoDefaultClient.Timeout = ttl
}
