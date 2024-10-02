package miso

import (
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/util"
	"github.com/gin-gonic/gin"
)

var (
	proxyClient *http.Client
)

func init() {
	proxyClient = &http.Client{Timeout: 0}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 1500
	transport.MaxIdleConnsPerHost = 1000
	transport.IdleConnTimeout = time.Minute * 10 // make sure that we can maximize the re-use of connnections
	proxyClient.Transport = transport
}

// Resolve proxy target path.
//
// Proxy path may be empty string if root path is requested, otherwise, it should guarantee to contain a prefix slash.
//
// returns ProxyHttpStatusError to respond a specific http status code.
type ProxyTargetResolver func(rail Rail, proxyPath string) (string, error)

type HttpProxy struct {
	filterLock    *sync.RWMutex
	filters       []ProxyFilter
	resolveTarget ProxyTargetResolver
}

// Create HTTP proxy for specific path.
//
// e.g., to create proxy path for /proxy/** and redirect all requests to localhost:8081.
//
//	_ = miso.NewHttpProxy("/proxy", func(proxyPath string) (string, error) {
//		return "http://localhost:8081" + proxyPath, nil
//	})
func NewHttpProxy(proxiedPath string, targetResolver ProxyTargetResolver) *HttpProxy {
	if targetResolver == nil {
		panic("targetResolver cannot be nil")
	}
	p := &HttpProxy{
		filters:    make([]ProxyFilter, 0),
		filterLock: &sync.RWMutex{},
	}
	p.resolveTarget = targetResolver
	if proxiedPath != "/" {
		RawAny(proxiedPath, p.proxyRequestHandler)
	}
	wildcardPath := proxiedPath
	if !strings.HasSuffix(wildcardPath, "/") {
		wildcardPath += "/"
	}
	wildcardPath += "*proxyPath"
	RawAny(wildcardPath, p.proxyRequestHandler)
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

	h.filterLock.RLock()
	defer h.filterLock.RUnlock()

	// filter request
	for _, f := range h.filters {
		if f.PreRequest == nil {
			continue
		}
		fr, err := f.PreRequest(pc)
		if err != nil || !fr.Next {
			pc.Rail.Debugf("request filtered, err: %v, ok: %v", err, fr)
			if err != nil {
				inb.HandleResult(WrapResp(*pc.Rail, nil, err, r.RequestURI), nil)
				return
			}
			return // discontinue, the filter should write the response itself, e.g., returning a 403 status code
		}
	}

	defer func() {
		for _, f := range h.filters {
			if f.PostRequest == nil {
				continue
			}
			f.PostRequest(pc)
		}
	}()

	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	cli := NewTClient(*pc.Rail, path).
		UseClient(proxyClient).
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

	pc.Rail.Debugf("Proxy request processed")
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

type ProxyFilter struct {
	PreRequest  func(pc *ProxyContext) (FilterResult, error)
	PostRequest func(pc *ProxyContext)
	Order       int // ascending order
}

type FilterResult struct {
	// should we continue processing the request.
	//
	// if Next=false, the filter should write proper response itself.
	Next bool
}

func (h *HttpProxy) AddFilter(f ProxyFilter) {
	h.filterLock.Lock()
	defer h.filterLock.Unlock()
	h.filters = append(h.filters, f)
	sort.Slice(h.filters, func(i, j int) bool { return h.filters[i].Order < h.filters[j].Order })
}

type ProxyHttpStatusError interface {
	Status() int
}
