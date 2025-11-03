package miso

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/http/pprof"
	"net/url"
	"strings"
	"time"

	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"
)

var (
	errPathNotFound                 = errs.NewErrf("Path not found")
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
// In terms of connection reuse, the IdleConnTimeout is 1min, MaxIdleConns is 0, MaxIdleConnsPerHost is 100 and MaxConnsPerHost is 500.
type HttpProxy struct {
	client          *http.Client
	filters         []ProxyFilter
	resolveTarget   ProxyTargetResolver
	rootProxiedPath string
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
//
// See [NewDynProxyTargetResolver].
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
	p.rootProxiedPath = proxiedPath
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

	defer pc.Rail.Debugf("Proxy request processed")

	handler := func(pc *ProxyContext) {
		w, r := pc.Inb.Unwrap()
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

		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}

		rproxy := &httputil.ReverseProxy{}
		rproxy.Transport = h.client.Transport
		rproxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			pc.Rail.Warnf("Failed to proxy request, %v", err)
		}
		rproxy.Rewrite = func(pr *httputil.ProxyRequest) {
			targetUrl, _ := url.Parse(path)
			pr.Out.URL = targetUrl
			pc.Rail.Infof("Rewrite proxy-request to '%v'", targetUrl)

			// propagate all headers to proxied servers, except the headers for tracing
			UsePropagationKeys(func(key string) {

				// the inbound request may contain headers that are one of our propagation keys
				// this can be a security problem
				pr.Out.Header.Del(key)

				v := pc.Rail.ctx.Value(key)
				if v != nil {
					if key == XSpanId {
						pr.Out.Header.Set(key, NewSpanId())
						return
					}
					if sv := cast.ToString(v); sv != "" {
						pr.Out.Header.Set(key, sv)
					}
				}
			})

			if IsDebugLevel() {
				pc.Rail.Debugf("Proxy request headers: %v", pr.Out.Header)
			}
		}
		rproxy.ModifyResponse = func(r *http.Response) error {
			pc.Rail.Debugf("Proxy response headers: %v, status: %v", r.Header, r.StatusCode)
			return nil
		}

		rproxy.ServeHTTP(w, r)
	}
	pi := newProxyFilters(pc, h.filters, handler)
	pi.next()
}

func (h *HttpProxy) AddFilter(f ProxyFilter) {
	h.filters = append(h.filters, f)
}

func (h *HttpProxy) AddPathFilter(pathPatterns []string, f ProxyFilter) {
	h.AddFilter(func(pc *ProxyContext, next func()) {
		if _, ok := strutil.MatchPathAnyVal(pathPatterns, pc.ProxyPath); ok {
			f(pc, next)
			return
		}
		next()
	})
}

func (h *HttpProxy) isRootPath() bool {
	return h.rootProxiedPath == "/"
}

func (h *HttpProxy) AddAccessFilter(whitelistPatterns func() []string, checkAuth func(pc *ProxyContext) (statusCode int, ok bool), f ProxyFilter) {

	h.AddFilter(func(pc *ProxyContext, next func()) {
		w, r := pc.Inb.Unwrap()
		rail := pc.Rail
		proxyPath := pc.ProxyPath

		valid := false

		// whitelisted path patterns
		if matched, ok := strutil.MatchPathAnyVal(whitelistPatterns(), proxyPath); ok {
			rail.Infof("Matched whitelist path: %v", matched)
			valid = true
		}

		invalidStatusCode := http.StatusUnauthorized

		// check authentication/authorization
		if !valid {
			sc, ok := checkAuth(pc)
			if ok {
				valid = true
			} else if sc != 0 {
				invalidStatusCode = sc
			}
		}

		if !valid {
			var body string = "***"
			if r.Body != nil && ContentTypeLoggable(r.Header.Get("content-type")) {
				if buf, err := io.ReadAll(r.Body); err == nil {
					body = "\n" + string(buf)
				}
			}
			rail.Warnf("Request forbidden (resource access not authorized): %v %v, body: %v", r.Method, r.RequestURI, body)
			w.WriteHeader(invalidStatusCode)
			return
		}

		next()
	})

	Info("Registered Access Filter")
}

func (h *HttpProxy) AddReqTimeLogFilter(exclPath func(proxyPath string) bool) {
	h.AddFilter(func(pc *ProxyContext, next func()) {
		_, r := pc.Inb.Unwrap()

		if exclPath(pc.ProxyPath) {
			next()
			return
		}

		start := time.Now()
		pc.Rail.Infof("Receive '%v %v' [%v]", r.Method, r.RequestURI, r.RemoteAddr)
		next()
		pc.Rail.Infof("Processed '%v %v' [%v]", r.Method, r.RequestURI, time.Since(start))
	})
}

// Add Filter for /debug/pprof/** and /debug/trace/** paths.
//
// Only active when the proxied path is '/'.
func (h *HttpProxy) AddDebugFilter(mustAuthInProd bool) error {
	if !h.isRootPath() {
		return nil
	}
	if !GetPropBool(PropServerPprofEnabled) {
		return nil
	}

	bearer := GetPropStr(PropServerPprofAuthBearer)
	if mustAuthInProd && IsProdMode() {
		if bearer == "" {
			return errs.NewErrf("Configuration '%v' for pprof authentication is missing, but pprof authentication is enabled", PropServerPprofAuthBearer)
		}
	}

	pat := []string{"/debug/pprof/**", "/debug/trace/**"}
	h.AddPathFilter(pat, func(pc *ProxyContext, _ func()) {
		w, r := pc.Inb.Unwrap()
		p := pc.ProxyPath

		if bearer != "" {
			token, ok := ParseBearer(r.Header.Get("Authorization"))
			if !ok || token != bearer {
				pc.Rail.Warnf("Bearer authorization failed, missing bearer token or token mismatch, %v %v", r.Method, r.RequestURI)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		if strings.HasPrefix(p, "/debug/pprof") {
			switch p {
			case "/debug/pprof/cmdline":
				pprof.Cmdline(w, r)
			case "/debug/pprof/profile":
				pprof.Profile(w, r)
			case "/debug/pprof/symbol":
				pprof.Symbol(w, r)
			case "/debug/pprof/trace":
				pprof.Trace(w, r)
			default:
				if name, found := strings.CutPrefix(p, "/debug/pprof/"); found && name != "" {
					pprof.Handler(name).ServeHTTP(w, r)
					return
				}
				pprof.Index(w, r)
			}
		} else if strings.HasPrefix(p, "/debug/trace") {
			switch p {
			case "/debug/trace/recorder/run":
				HandleFlightRecorderRun(pc.Inb)
			case "/debug/trace/recorder/stop":
				HandleFlightRecorderStop(pc.Inb)
			}
		}
	})

	Infof("Registered Debug Filter for %v", pat)
	return nil
}

// Add Filter for healthcheck.
//
// Only active when the proxied path is '/'.
func (h *HttpProxy) AddHealthcheckFilter() {
	if !h.isRootPath() {
		return
	}
	hcUrl := GetPropStr(PropHealthCheckUrl)
	if hcUrl == "" {
		return
	}
	h.AddPathFilter([]string{hcUrl}, func(pc *ProxyContext, next func()) {
		// check if it's a healthcheck endpoint (for consul), we don't really return anything, so it's fine to expose it
		w, _ := pc.Inb.Unwrap()
		if IsHealthcheckPass(*pc.Rail) {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	Infof("Registered Healthcheck Filter for %v", hcUrl)
}

// Add Filter for metrics and prometheus.
//
// Only active when the proxied path is '/'.
func (h *HttpProxy) AddMetricsFilter(hiso prometheus.Histogram, exclPath func(proxyPath string) bool) {
	if !h.isRootPath() {
		return
	}

	metricsEndpoint := GetPropStr(PropMetricsRoute)
	if metricsEndpoint == "" {
		return
	}

	h.AddFilter(func(pc *ProxyContext, next func()) {

		if pc.ProxyPath == metricsEndpoint {
			w, r := pc.Inb.Unwrap()
			PrometheusHandler().ServeHTTP(w, r)
			return
		}

		if exclPath(pc.ProxyPath) {
			next()
			return
		}

		timer := NewHistTimer(hiso)
		defer timer.ObserveDuration()
		next()
	})
	Infof("Registered Metrics Filter for %v", metricsEndpoint)
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
	ProxyPath string // Proxied path without query parameters.

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
	copy := slutil.SliceCopy(pi)
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
	transport.IdleConnTimeout = time.Minute * 1
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

// Resolve proxy target based on service discovery.
func NewDynProxyTargetResolver() ProxyTargetResolver {
	return func(rail Rail, proxyPath string) (string, error) {
		// parse the request path, extract service name, and the relative url for the backend server
		var sp ServicePath
		var err error
		if sp, err = parseServicePath(proxyPath); err != nil {
			rail.Warnf("Invalid request, %v", err)
			return "", GatewayError{StatusCode: 404}
		}
		rail.Debugf("Parsed service path: %#v", sp)
		target, err := GetServiceRegistry().ResolveUrl(rail, sp.ServiceName, sp.Path)
		if err != nil {
			rail.Warnf("ServiceRegistry ResolveUrl failed, %v", err)
			return "", GatewayError{StatusCode: 404}
		}
		return target, nil
	}
}

type ServicePath struct {
	ServiceName string
	Path        string
}

func parseServicePath(url string) (ServicePath, error) {
	rurl := []rune(url)[1:] // remove leading '/'

	// root path, invalid request
	if len(rurl) < 1 {
		return ServicePath{}, errPathNotFound.New()
	}

	start := 0
	for i := range rurl {
		if rurl[i] == '/' && i > 0 {
			start = i
			break
		}
	}

	if start < 1 {
		return ServicePath{}, errPathNotFound.New()
	}

	return ServicePath{
		ServiceName: string(rurl[0:start]),
		Path:        string(rurl[start:]),
	}, nil
}

type GatewayError struct {
	StatusCode int
}

func (g GatewayError) Status() int {
	return g.StatusCode
}

func (g GatewayError) Error() string {
	return fmt.Sprintf("gateway error, %v", g.StatusCode)
}
