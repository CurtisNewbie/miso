package miso

import (
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/curtisnewbie/miso/util"
	"github.com/gin-gonic/gin"
)

var (
	defaultProxyClient *http.Client = newProxyClient()
)

// Resolve proxy target path.
//
// Proxy path may be empty string if root path is requested, otherwise, it should guarantee to contain a prefix slash.
//
// returns ProxyHttpStatusError to respond a specific http status code.
type ProxyTargetResolver func(rail Rail, proxyPath string) (string, error)

// Http Reverse Proxy.
//
// HttpProxy by default use http.Client with 5s connect timeout and 30s response header timeout.
// In terms of connection reuse, the IdleConnTimeout is 5min, MaxIdleConns is 5, MaxIdleConnsPerHost is 100 and MaxConnsPerHost is 500.
type HttpProxy struct {
	client        *http.Client
	filters       []ProxyFilter
	resolveTarget ProxyTargetResolver
}

// Create HTTP proxy for specific path.
//
// If proxiedPath is '/', then the default health check endpoint handler,
// promethues endpoint handler, pprof endpoint handler, and apidoc endpoint handler are all disabled to avoid path conflicts.
//
// This func must be called before server bootstraps.
//
// e.g., to create proxy path for /proxy/** and redirect all requests to localhost:8081.
//
//	// proxy all requests to /proxy* to server localhost:8081
//	_ = miso.NewHttpProxy("/proxy", func(proxyPath string) (string, error) {
//		return "http://localhost:8081" + proxyPath, nil
//	})
func NewHttpProxy(proxiedPath string, targetResolver ProxyTargetResolver) *HttpProxy {
	if targetResolver == nil {
		panic("targetResolver cannot be nil")
	}
	proxiedPath = strings.TrimSpace(proxiedPath)
	if proxiedPath == "" {
		proxiedPath = "/"
	}

	if proxiedPath == "/" {
		DisableDefaultHealthCheckHandler() // disable the default health check endpoint to avoid conflicts
		DisablePrometheusBootstrap()       // bootstrap metrics and prometheus stuff manually
		DisablePProfEndpointRegister()     // handle pprof endpoints manually
		DisableApidocEndpointRegister()    // do not generate apidoc
	}

	p := &HttpProxy{
		client:  defaultProxyClient,
		filters: make([]ProxyFilter, 0),
	}
	p.resolveTarget = targetResolver
	if proxiedPath != "/" {
		HttpAny(proxiedPath, p.proxyRequestHandler)
	}
	wildcardPath := proxiedPath
	if !strings.HasSuffix(wildcardPath, "/") {
		wildcardPath += "/"
	}
	wildcardPath += "*proxyPath"
	HttpAny(wildcardPath, p.proxyRequestHandler)
	return p
}

func (h *HttpProxy) proxyRequestHandler(inb *Inbound) {
	_rail := inb.Rail()
	pc := newProxyContext(&_rail, inb)

	// proxy path
	proxyPath := inb.Engine().(*gin.Context).Param("proxyPath")
	pc.ProxyPath = proxyPath

	w, r := inb.Unwrap()
	pc.Rail.Debugf("Request: %v %v, headers: %v, proxyPath: %v", r.Method, r.URL.Path, r.Header, proxyPath)

	// resolve proxy target path, in most case, it's another backend server.
	path, err := h.resolveTarget(*pc.Rail, proxyPath)
	if err != nil {
		pc.Rail.Warnf("Resolve target failed, path: %v, %v", proxyPath, err)

		status := 0
		if hse, ok := err.(ProxyHttpStatusError); ok {
			status = hse.Status()
		}
		if status == 0 {
			status = 404
		}
		w.WriteHeader(status)
		return
	}
	defer pc.Rail.Debugf("Proxy request processed")

	handler := func(pc *ProxyContext) {

		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}
		cli := NewTClient(*pc.Rail, path).
			UseClient(h.client).
			EnableTracing()

		// propagate all headers to client, except the headers for tracing
		propagationKeys := util.NewSet[string]()
		propagationKeys.AddAll(GetPropagationKeys())

		for k, arr := range r.Header {
			// the inbound request may contain headers that are one of our propagation keys
			// this can be a security problem
			if propagationKeys.Has(k) {
				continue
			}
			for _, v := range arr {
				cli.AddHeader(k, v)
			}
		}

		var tr *TResponse
		switch r.Method {
		case http.MethodGet:
			tr = cli.Get()
		case http.MethodPut:
			tr = cli.Put(r.Body)
		case http.MethodPost:
			tr = cli.Post(r.Body)
		case http.MethodDelete:
			tr = cli.Delete()
		case http.MethodHead:
			tr = cli.Head()
		case http.MethodOptions:
			tr = cli.Options()
		default:
			w.WriteHeader(404) // not gonna happen
			return
		}

		if tr.Err != nil {
			pc.Rail.Warnf("Proxy request failed, original path: %v, actual path: %v, err: %v", r.URL.Path, path, tr.Err)
			if nerr, ok := tr.Err.(net.Error); ok && nerr.Timeout() {
				pc.Rail.Errorf("Proxy request failed, request timeout, original path: %v, actual path: %v, err: %v", r.URL.Path, path, tr.Err)
				w.WriteHeader(http.StatusGatewayTimeout)
				return
			}
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer tr.Close()

		pc.Rail.Debugf("Proxy response headers: %v, status: %v", tr.RespHeader, tr.StatusCode)

		// headers from proxied servers
		for k, v := range tr.RespHeader {
			for _, hv := range v {
				w.Header().Add(k, hv)
			}
		}

		if IsDebugLevel() {
			pc.Rail.Debug(w.Header())
		}

		// write data from proxied to client
		w.WriteHeader(tr.StatusCode)
		if tr.Resp.Body != nil {
			if _, err = io.Copy(w, tr.Resp.Body); err != nil {
				pc.Rail.Warnf("Failed to write proxy response, %v", err)
			}
		}
	}
	pi := newProxyFilters(pc, h.filters, handler)
	pi.next()
}

func (h *HttpProxy) AddFilter(f ProxyFilter) {
	h.filters = append(h.filters, f)
}

func (h *HttpProxy) ChangeClient(c *http.Client) {
	if c == nil {
		panic("*http.Client cannot be nil")
	}
	h.client = c
}

type ProxyContext struct {
	Rail      *Rail
	Inb       *Inbound
	ProxyPath string

	attr map[string]any // attributes, it's lazy, only initialized on write
}

func (pc *ProxyContext) SetAttr(key string, val any) {
	if pc.attr == nil {
		pc.attr = map[string]any{}
	}

	pc.attr[key] = val
}

func (pc *ProxyContext) DelAttr(key string) {
	if pc.attr == nil {
		pc.attr = map[string]any{}
	}

	delete(pc.attr, key)
}

func (pc *ProxyContext) GetAttr(key string) (any, bool) {
	if pc.attr == nil {
		return nil, false
	}

	v, ok := pc.attr[key]
	return v, ok
}

func newProxyContext(rail *Rail, inb *Inbound) *ProxyContext {
	return &ProxyContext{
		Rail: rail,
		attr: nil,
		Inb:  inb,
	}
}

type ProxyHttpStatusError interface {
	Status() int
}

type ProxyFilter = func(pc *ProxyContext, next func())

type proxyFilters struct {
	idx     int
	c       *ProxyContext
	filters []ProxyFilter
}

func (it *proxyFilters) next() {
	it.idx++
	if it.idx < len(it.filters) {
		it.filters[it.idx](it.c, it.next)
	}
}

func newProxyFilters(c *ProxyContext, pi []ProxyFilter, handler func(pc *ProxyContext)) *proxyFilters {
	copy := util.SliceCopy(pi)
	return &proxyFilters{
		idx:     -1,
		c:       c,
		filters: append(copy, func(pc *ProxyContext, next func()) { handler(c) }),
	}
}

func newProxyClient(opts ...func(*http.Transport)) *http.Client {
	c := &http.Client{}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 0
	transport.MaxIdleConnsPerHost = 100
	transport.MaxConnsPerHost = 500
	transport.IdleConnTimeout = time.Minute * 5
	transport.DialContext = (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 15 * time.Second,
	}).DialContext
	transport.ResponseHeaderTimeout = 30 * time.Second
	c.Transport = transport
	for _, op := range opts {
		op(transport)
	}
	return c
}
