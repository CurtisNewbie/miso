package miso

import (
	"io"
	"net/http"
	"sort"
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

type HttpProxy struct {
	filterLock    *sync.RWMutex
	filters       []ProxyFilter
	resolveTarget func(relPath string) (string, error)
}

// Create HTTP proxy for specific path.
//
// e.g., to create proxy path for /proxy/** and redirect all requests to localhost:8081.
//
//	_ = miso.NewHttpProxy("/proxy", func(relPath string) (string, error) {
//		return "http://localhost:8081" + relPath, nil
//	})
func NewHttpProxy(proxiedPath string, targetResolver func(relPath string) (string, error)) *HttpProxy {
	if targetResolver == nil {
		panic("targetResolver cannot be nil")
	}
	p := &HttpProxy{
		filters:    make([]ProxyFilter, 0),
		filterLock: &sync.RWMutex{},
	}
	p.resolveTarget = targetResolver
	RawAny(proxiedPath+"/*proxyPath", p.proxyRequestHandler)
	return p
}

func (h *HttpProxy) proxyRequestHandler(inb *Inbound) {
	rail := inb.Rail()
	w, r := inb.Unwrap()
	rail.Debugf("Request: %v %v, headers: %v", r.Method, r.URL.Path, r.Header)

	// parse the request path, extract service name, and the relative url for the backend server
	path, err := h.resolveTarget(r.URL.Path)
	if err != nil {
		rail.Warnf("Resolve target failed, path: %v, %v", r.URL.Path, err)
		w.WriteHeader(404)
		return
	}

	pc := newProxyContext(rail, inb)
	proxyPath := inb.Engine().(*gin.Context).Param("proxyPath")
	pc.SetAttr("PROXY_PATH", proxyPath)

	h.filterLock.RLock()
	defer h.filterLock.RUnlock()

	for i := range h.filters {
		fr, err := h.filters[i].FilterFunc(pc)
		if err != nil || !fr.Next {
			rail.Debugf("request filtered, err: %v, ok: %v", err, fr)
			if err != nil {
				inb.HandleResult(WrapResp(rail, nil, err, r.RequestURI), nil)
				return
			}

			return // discontinue, the filter should write the response itself, e.g., returning a 403 status code
		}
		pc = fr.ProxyContext // replace the ProxyContext, trace may be set
	}

	// continue propagating the trace
	rail = pc.Rail

	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	cli := NewTClient(rail, path).
		UseClient(proxyClient).
		EnableTracing()

	propagationKeys := util.NewSet[string]()
	propagationKeys.AddAll(GetPropagationKeys())

	// propagate all headers to client
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
		w.WriteHeader(404)
		return
	}

	if tr.Err != nil {
		rail.Warnf("proxy request failed, original path: %v, actual path: %v, err: %v", r.URL.Path, path, tr.Err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer tr.Close()

	rail.Debugf("post proxy request, proxied response headers: %v, status: %v", tr.RespHeader, tr.StatusCode)

	// headers from proxied servers
	for k, v := range tr.RespHeader {
		for _, hv := range v {
			w.Header().Add(k, hv)
		}
	}
	if IsDebugLevel() {
		rail.Debug(w.Header())
	}

	w.WriteHeader(tr.StatusCode)

	// write data from proxied to client
	if tr.Resp.Body != nil {
		if _, err = io.Copy(w, tr.Resp.Body); err != nil {
			rail.Warnf("Failed to write proxy response, %v", err)
		}
	}

	rail.Debugf("proxy request handled")
}

type ProxyContext struct {
	Rail Rail
	Inb  *Inbound

	attr map[string]any // attributes, it's lazy, only initialized on write
}

func (pc *ProxyContext) SetAttr(key string, val any) {
	if pc.attr == nil {
		pc.attr = map[string]any{}
	}

	pc.attr[key] = val
}

func (pc *ProxyContext) GetAttr(key string) (any, bool) {
	if pc.attr == nil {
		return nil, false
	}

	v, ok := pc.attr[key]
	return v, ok
}

func newProxyContext(rail Rail, inb *Inbound) ProxyContext {
	return ProxyContext{
		Rail: rail,
		attr: nil,
		Inb:  inb,
	}
}

type ProxyFilter struct {
	FilterFunc func(proxyContext ProxyContext) (FilterResult, error) // filtering function
	Order      int                                                   // ascending order
}

type FilterResult struct {
	ProxyContext ProxyContext
	Next         bool
}

func NewFilterResult(pc ProxyContext, next bool) FilterResult {
	return FilterResult{ProxyContext: pc, Next: next}
}

func (h *HttpProxy) AddFilter(f ProxyFilter) {
	h.filterLock.Lock()
	defer h.filterLock.Unlock()
	h.filters = append(h.filters, f)
	sort.Slice(h.filters, func(i, j int) bool { return h.filters[i].Order < h.filters[j].Order })
}
