package miso

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/http/pprof"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/flow"
	"github.com/curtisnewbie/miso/util/async"
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

		SetDefProp(PropServerPropagateInboundTrace, false) // disable trace propagation, we are the entry point
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

// isSuspiciousProxyPath reports whether the proxy path contains dot-segment traversal
// (e.g. "..", "..;") or backslashes.
//
// gin already percent-decodes the path, so "%2e%2e" arrives as "..". Backend servers
// (Tomcat, nginx, Spring, etc.) normalize such paths differently than the gateway's
// pattern matcher (e.g. "/open/api/../admin" may be served as "/admin" by the backend),
// so forwarding them verbatim could bypass the access filters. They are rejected here
// at the gateway, which must be the single normalization point.
func isSuspiciousProxyPath(p string) bool {
	for _, seg := range strings.Split(p, "/") {
		if strings.HasPrefix(seg, "..") || strings.Contains(seg, "\\") {
			return true
		}
	}
	return false
}

func (h *HttpProxy) proxyRequestHandler(inb *Inbound) {
	_rail := inb.Rail()

	pc := newProxyContext(&_rail, inb)

	// proxy path
	proxyPath := inb.Engine().(*gin.Context).Param("proxyPath")
	pc.ProxyPath = proxyPath

	// reject suspicious paths (dot-segment traversal, backslashes) before they reach
	// the access filters or get forwarded, see [isSuspiciousProxyPath]
	if isSuspiciousProxyPath(proxyPath) {
		pc.Rail.Warnf("Rejecting request with suspicious proxy path: %v", proxyPath)
		pc.Inb.Status(http.StatusBadRequest)
		return
	}

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

				v := pc.Rail.Value(key)
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
			if IsDebugLevel() {
				pc.Rail.Debugf("Proxy response headers: %v, status: %v", r.Header, r.StatusCode)
			} else {
				pc.Rail.Infof("Proxy response status_code: %v", r.StatusCode)
			}
			return nil
		}

		if err := async.CapturePanicErr(func() {
			rproxy.ServeHTTP(w, r)
		}); err != nil {
			pc.Rail.Warnf("Proxy request failed, %v", err)
		}
	}
	pi := newProxyFilters(pc, h.filters, handler)
	pi.next()
}

func (h *HttpProxy) AddFilter(f ProxyFilter) {
	h.filters = append(h.filters, f)
}

func (h *HttpProxy) AddIPBlacklistFilter(matchBlacklist func(ip string) bool) {
	h.AddFilter(func(pc *ProxyContext, next func()) {
		ip := pc.Inb.r.RemoteAddr
		if ipseg := strings.SplitN(ip, ":", 2); len(ipseg) > 1 {
			ip = ipseg[0]
		}

		if matchBlacklist(ip) {
			pc.Rail.Warnf("Matched IP blacklist (%v), returning fake 200, request rejected", ip)
			pc.Inb.Status(http.StatusOK) // fake 200 :D
			return
		}
		next()
	})
}

func (h *HttpProxy) AddPathFilter(pathPatterns []string, f ProxyFilter) {
	h.AddFilter(func(pc *ProxyContext, next func()) {
		if ok := strutil.MatchPathAny(pathPatterns, pc.ProxyPath); ok {
			f(pc, next)
			return
		}
		next()
	})
}

func (h *HttpProxy) isRootPath() bool {
	return h.rootProxiedPath == "/"
}

func (h *HttpProxy) JoinCheckAuth(checkAuth ...func(pc *ProxyContext) (statusCode int, ok bool)) func(pc *ProxyContext) (statusCode int, ok bool) {
	return func(pc *ProxyContext) (statusCode int, ok bool) {
		var lastCode int
		for _, check := range checkAuth {
			if c, ok := check(pc); ok {
				return 0, true
			} else {
				lastCode = c
			}
		}
		return lastCode, false
	}
}

// Add Access Filter.
//
// See [HttpProxy.WithDynAuthCheck].
func (h *HttpProxy) AddAccessFilter(whitelistPatterns func() []string, checkAuth func(pc *ProxyContext) (statusCode int, ok bool)) {

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
			rail.Warnf("Request forbidden (resource access not authorized): %v %v (%v), body: %v", r.Method, r.RequestURI, r.RemoteAddr, body)
			w.WriteHeader(invalidStatusCode)
			return
		}

		next()
	})

	Info("Registered Access Filter")
}

// DynAccessFilterConfig is the configuration loaded by [HttpProxy.AddConfDynAccessFilter] from a
// single prop root key.
type DynAccessFilterConfig struct {
	// Whitelist path patterns, requests matching these paths are allowed without authentication.
	Whitelist []string
	// AuthRoutes loaded dynamically, see [HttpProxy.LoadDynAuthRouteFromProp] for the route format.
	AuthRoutes []DynAuthRoute
}

// AddConfDynAccessFilter adds an access filter with whitelist path patterns and dynamically loaded
// auth routes, both loaded from the configuration under a single prop root key, unmarshalled into
// [DynAccessFilterConfig].
//
// E.g., for prop root key "proxy.access.filter":
//
//	proxy.access.filter:
//	  whitelist:
//	    - "/health"
//	    - "/open/api/**"
//	  auth-routes:
//	    - name: "myauth1"
//	      type: "bearer"
//	      bearer: "mybearer1"
//	      trace:
//	        username: "trace-user"
//	        role: "myrole"
//	      path-patterns:
//	        - "/path1"
//	        - "/path2"
//	        - "/path3"
//	    - name: "myauth2"
//	      type: "basic"
//	      username: "myuser"
//	      password: "mypassword"
//	      trace:
//	        username: "trace-user"
//	        role: "myrole"
//	      path-patterns:
//	        - "/path4"
//	        - "/path5"
//	        - "/path6"
//	    - name: "myauth3"
//	      type: "remote"
//	      path-patterns: # may be omitted, remote routes without path-patterns match all paths
//	        - "/path7"
//	      remote:
//	        path: "lb://auth-service/open/api/auth/check" # lb:// prefix resolves the address via service discovery
//	        body-map:
//	          authorization: "token"
//	          path: "url"
//	          method: "method"
//	        decision-field: "data.valid" # dotted path into the response body, doesn't have to be GnResp-wrapped
//	        user:
//	          userno: "data.userno"
//	          username: "data.username"
//	          roleno: "data.roleno"
//	          role: "data.role"
//
// The whitelist and the dyn auth routes are loaded together, cached, and refreshed in the background
// every refreshEvery, so frequent requests don't have to read the configuration every time, and the
// values are never outdated for longer than the refresh interval.
//
// If refreshEvery <= 0, both are loaded only once.
//
// E.g.,
//
//	var h *miso.HttpProxy
//	h.AddConfDynAccessFilter("access.filter", 5*time.Second)
func (h *HttpProxy) AddConfDynAccessFilter(propRootKey string, refreshEvery time.Duration) {
	c := NewRefreshedCache(refreshEvery, func() DynAccessFilterConfig {
		cfg := UnmarshalFromPropKeyAs[DynAccessFilterConfig](propRootKey)
		cfg.AuthRoutes = filterDynAuthRoutes(cfg.AuthRoutes)
		return cfg
	})
	h.AddAccessFilter(func() []string {
		return c.Get().Whitelist
	}, h.WithDynAuthCheck(func() []DynAuthRoute {
		return c.Get().AuthRoutes
	}))
}

// filterDynAuthRoutes validates and filters DynAuthRoutes, dropping invalid routes.
func filterDynAuthRoutes(p []DynAuthRoute) []DynAuthRoute {
	return slutil.CopyFilterUpdate(p, func(d DynAuthRoute) (_d DynAuthRoute, incl bool) {
		// remote routes may omit path-patterns to match all paths, other routes require them
		if len(d.PathPatterns) < 1 && !strings.EqualFold(d.Type, DynAuthTypeRemote) {
			return d, false
		}

		if d.Type == "" {
			if d.Bearer != "" {
				d.Type = DynAuthTypeBearer
			} else {
				d.Type = DynAuthTypeBasic
			}
		}

		switch strings.ToLower(d.Type) {
		case DynAuthTypeBasic:
			if d.Username == "" || d.Password == "" {
				return d, false
			}
		case DynAuthTypeBearer:
			if d.Bearer == "" {
				return d, false
			}
		case DynAuthTypeRemote:
			if d.Remote.Path == "" || d.Remote.DecisionField == "" {
				return d, false
			}
		default:
			return d, false
		}
		return d, true
	})
}

// WsAccessFilterConfig configures a WebSocket access filter entry.
type WsAccessFilterConfig struct {
	// PathPatterns to match for WebSocket upgrade requests. If empty, match all path patterns.
	PathPatterns []string `mapstructure:"path-patterns"`
	// TokenQueryKey is the query parameter key name that carries the JWT token. If empty, reject all matched requests.
	// e.g., "token" reads from ?token=xxx.
	TokenQueryKey string `mapstructure:"token-query-key"`
}

// AddWsAccessFilter adds a filter for WebSocket upgrade requests.
//
// For requests that are WebSocket upgrades and whose proxy path matches
// a configured PathPatterns entry, this filter:
// 1. Reads the query parameter named by TokenQueryKey
// 2. Calls checkAuth(token, pc) to validate
// 3. If auth fails, responds with the returned statusCode (defaults to 401)
// 4. If auth passes, proceeds to the next filter
//
// Non-WebSocket requests and WS requests with no matching config pass through.
//
// The checkAuth callback should validate the token and may set user info
// in the proxy context via pc.SetAttr().
func (h *HttpProxy) AddWsAccessFilter(
	loadConfigs func() []WsAccessFilterConfig,
	checkAuth func(token string, pc *ProxyContext) (statusCode int, ok bool),
) {
	h.AddFilter(func(pc *ProxyContext, next func()) {
		_, r := pc.Inb.Unwrap()

		// only interested in WebSocket upgrade requests
		if !strings.EqualFold(r.Header.Get("Connection"), "upgrade") ||
			!strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			next()
			return
		}

		proxyPath := pc.ProxyPath

		for _, cfg := range loadConfigs() {
			// empty PathPatterns matches all paths
			if len(cfg.PathPatterns) > 0 && !strutil.MatchPathAny(cfg.PathPatterns, proxyPath) {
				continue
			}

			pc.Rail.Infof("WS access filter matched path: %v", proxyPath)

			if cfg.TokenQueryKey == "" {
				pc.Rail.Warnf("WS access filter: TokenQueryKey is empty in config")
				http.Error(pc.Inb.Writer(), http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			token := r.URL.Query().Get(cfg.TokenQueryKey)
			if token == "" {
				pc.Rail.Warnf("WS access filter: token not found in query param '%v'", cfg.TokenQueryKey)
				http.Error(pc.Inb.Writer(), http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			sc, authOk := checkAuth(token, pc)
			if !authOk {
				if sc == 0 {
					sc = http.StatusUnauthorized
				}
				pc.Rail.Warnf("WS access filter auth failed, status: %v", sc)
				http.Error(pc.Inb.Writer(), http.StatusText(sc), sc)
				return
			}

			pc.Rail.Infof("WS access filter auth passed")
			next()
			return
		}

		// no matching config, pass through
		next()
	})

	Info("Registered WS Access Filter")
}

const (
	// DynAuthTypeBearer, bearer token authentication.
	DynAuthTypeBearer = "bearer"

	// DynAuthTypeBasic, basic authentication.
	DynAuthTypeBasic = "basic"

	// DynAuthTypeRemote, authentication/authorization delegated to a downstream auth service.
	DynAuthTypeRemote = "remote"
)

type DynAuthRoute struct {
	// Name of the auth route, used for logging.
	Name string

	// Type of authentication, 'bearer', 'basic' or 'remote'.
	Type string

	// Bearer token to match against when Type is 'bearer'.
	Bearer string

	// Username used to build the Basic auth credentials when Type is 'basic'.
	Username string

	// Password used to build the Basic auth credentials when Type is 'basic'.
	Password string

	// PathPatterns that this auth route applies to.
	//
	// Bearer and basic routes must define at least one path pattern, routes without any are filtered
	// out when the configuration is loaded. Remote routes may omit path-patterns, in which case they
	// match all paths.
	PathPatterns []string

	// Trace info propagated downstream via trace when the request is authenticated.
	//
	// Only used by bearer and basic routes. When the route matches and the field is non-empty, the
	// value is stored in the trace and propagated to downstream services. Remote routes propagate
	// user info from the auth response instead, see [DynRemoteUserMapping].
	Trace DynAuthTrace

	// Remote configures remote authentication/authorization, only used when Type is 'remote'.
	Remote DynRemoteAuthConfig

	basic string `mapstructure:"-"`
}

// DynRemoteAuthConfig configures a remote authentication/authorization route that
// delegates authentication AND authorization to a downstream auth service in a single call.
type DynRemoteAuthConfig struct {

	// Path of the endpoint on the downstream auth service.
	// Use the 'lb://service-name/...' syntax to resolve the address via service discovery
	// (the 'lb:service-name/...' form is also supported), or a full URL
	// (e.g., 'http://host:port/...') to call it directly.
	Path string

	// BodyMap maps source values to field names of the auth request body.
	BodyMap DynRemoteBodyMap

	// DecisionField is the dotted path (e.g. "data.valid") into the response body that holds
	// the allow/deny boolean. Paths are resolved against the full response body, so for
	// GnResp-wrapped responses ({error, errorCode, msg, data}) they must include the "data."
	// prefix, while plain responses use paths like "valid".
	DecisionField string

	// User mapping from response body dotted paths to the user stored in trace via flow.StoreUser when the request is allowed.
	User DynRemoteUserMapping
}

// DynRemoteBodyMap maps source values to field names of the auth request body.
//
// The value of each field is the field name in the auth request body, e.g.
// 'authorization: "token"' sends the raw Authorization header as the 'token'
// field of the auth request body. Empty value means the source is not sent.
type DynRemoteBodyMap struct {
	// Authorization - raw Authorization header, without any scheme-specific parsing.
	Authorization string

	// Path - proxy path, without query string.
	Path string

	// Method - HTTP method.
	Method string
}

type DynRemoteUserMapping struct {
	UserNo   string
	Username string
	RoleNo   string
	Role     string
}

// DynAuthTrace configures the trace info propagated downstream via trace when the request is authenticated.
type DynAuthTrace struct {
	Username string
	Role     string
}

func (d *DynAuthRoute) BuildBasic() string {
	if d.basic == "" {
		base := d.Username + ":" + d.Password
		d.basic = "Basic " + base64.StdEncoding.EncodeToString(strutil.UnsafeStr2Byt(base))
	}
	return d.basic
}

func (d *DynAuthRoute) CheckAuth(v *dynAuthReq) bool {
	switch strings.ToLower(d.Type) {
	case "basic":
		if d.Username == "" || d.Password == "" {
			// never valid without credentials, e.g. "Basic " + base64(":") must not authenticate
			return false
		}
		return v.CheckBasic(d.BuildBasic())
	default:
		return v.CheckBearer(d.Bearer)
	}
}

type dynAuthReq struct {
	auth         string
	basicParsed  bool
	bearer       string
	isBearer     bool
	bearerParsed bool
}

func (d *dynAuthReq) Bearer() (string, bool) {
	if d.bearerParsed {
		return d.bearer, d.isBearer
	}
	d.bearerParsed = true
	d.bearer, d.isBearer = ParseBearer(d.auth)
	return d.bearer, d.isBearer
}

func (d *dynAuthReq) CheckBasic(auth string) bool {
	// constant-time comparison to avoid timing side-channels on credentials
	return subtle.ConstantTimeCompare([]byte(d.auth), []byte(auth)) == 1
}

func (d *dynAuthReq) CheckBearer(bearer string) bool {
	v, ok := d.Bearer()
	if !ok {
		return false
	}
	// constant-time comparison to avoid timing side-channels on bearer tokens
	return subtle.ConstantTimeCompare([]byte(bearer), []byte(v)) == 1
}

// Check Authorization With dynamically loaded DynAuthRoute.
//
// When a request is authenticated, the configured user info (e.g., Role) is stored in the
// trace and propagated to downstream services.
//
// E.g.,
//
//	var h *miso.HttpProxy
//	h.AddConfDynAccessFilter("access.filter", 5*time.Second)
//
// See [HttpProxy.AddAccessFilter].
// See [HttpProxy.AddConfDynAccessFilter].
func (h *HttpProxy) WithDynAuthCheck(load func() []DynAuthRoute) func(pc *ProxyContext) (statusCode int, ok bool) {
	return func(pc *ProxyContext) (statusCode int, isBearer bool) {
		authHeader := pc.Inb.Header("Authorization")

		// all auth routes require an Authorization header
		if strutil.IsBlankStr(authHeader) {
			return 0, false
		}

		cand := &dynAuthReq{auth: authHeader}
		bars := load()
		for _, bar := range bars {
			if strings.EqualFold(bar.Type, DynAuthTypeRemote) { // handle basic/bearer first
				continue
			}
			if !bar.CheckAuth(cand) {
				continue
			}
			matched, ok := strutil.MatchPathAnyVal(bar.PathPatterns, pc.ProxyPath)
			if ok {
				pc.Inb.Infof("Matched '%v' Bearer Authrization Path Pattern: '%v'", bar.Name, matched)

				if bar.Trace.Username != "" {
					*pc.Rail = pc.Rail.WithCtxVal(flow.XUsername, bar.Trace.Username)
				}
				if bar.Trace.Role != "" {
					*pc.Rail = pc.Rail.WithCtxVal(flow.XRole, bar.Trace.Role)
				}

				return 0, true
			}
		}

		// remote auth routes are evaluated after all non-remote routes, in config order
		for _, bar := range bars {
			if !strings.EqualFold(bar.Type, DynAuthTypeRemote) {
				continue
			}
			matched := true
			var matchedPath string
			if len(bar.PathPatterns) > 0 {
				matched = false
				if mp, ok := strutil.MatchPathAnyVal(bar.PathPatterns, pc.ProxyPath); ok {
					matchedPath = mp
					matched = true
				}
			}

			if matched {
				if matchedPath != "" {
					pc.Inb.Infof("Matched '%v' Remote Auth Path Pattern: '%v'", bar.Name, matchedPath)
				} else {
					pc.Inb.Infof("Matched '%v' Remote Auth", bar.Name)
				}
				return h.checkRemoteAuth(pc, bar)
			}
		}
		return 0, false
	}
}

// checkRemoteAuth delegates authentication AND authorization to a downstream auth service
// for the given remote auth route.
//
// The downstream auth service is called via service discovery, and its response body
// (doesn't have to be GnResp-wrapped, any JSON object works) is used to determine the decision:
//
//   - request error -> 503
//   - decision field missing -> 502
//   - decision false -> 403 if a user is present in the response, otherwise 401
//   - decision true -> user info (if any) stored in trace via flow.StoreUser, request allowed
func (h *HttpProxy) checkRemoteAuth(pc *ProxyContext, bar DynAuthRoute) (int, bool) {
	rail := pc.Rail
	r := pc.Inb.Request()

	// extract vocabulary values, the raw Authorization header is sent as-is,
	// any scheme-specific parsing is up to the downstream auth service
	authHeader := pc.Inb.Header("Authorization")
	if strutil.IsBlankStr(authHeader) {
		rail.Warnf("Remote auth: missing Authorization header")
		return http.StatusUnauthorized, false
	}

	// build auth request body from body-map
	body := map[string]any{}
	if dst := bar.Remote.BodyMap.Authorization; dst != "" {
		body[dst] = authHeader
	}
	if dst := bar.Remote.BodyMap.Path; dst != "" {
		body[dst] = pc.ProxyPath
	}
	if dst := bar.Remote.BodyMap.Method; dst != "" {
		body[dst] = r.Method
	}

	// call downstream auth service, 'lb:service-name/...' paths are resolved via service discovery
	var res map[string]any
	err := NewClient(*rail, bar.Remote.Path).
		EnableTracing().
		Require2xx().
		PostJson(body).
		Json(&res)
	if err != nil {
		rail.Warnf("Remote auth request failed, %v", err)
		return http.StatusServiceUnavailable, false
	}

	// decision
	sc, ok, user := mapRemoteAuthResult(*rail, res, bar.Remote)
	if !ok {
		return sc, false
	}

	// allowed: store user in trace for downstream propagation (skip when the
	// auth response carries no user info, so that upstream trace identity is preserved)
	*pc.Rail = flow.StoreUser(*pc.Rail, user)

	return 0, true
}

// mapRemoteAuthResult maps the downstream auth response to a decision.
//
// Dotted paths (decision-field, user.*) are resolved against the full response body,
// so for GnResp-wrapped responses ({error, errorCode, msg, data}) they must include the
// "data." prefix (e.g. "data.valid"), while plain responses use paths like "valid".
//
// When denied (ok=false), the returned status code is the response code:
//   - decision field missing -> 502 (warning logged with the response body)
//   - decision false -> 403 if user info is present in the response, otherwise 401
//
// When allowed (ok=true), the user info (if any) is returned so it can be
// propagated downstream via trace.
func mapRemoteAuthResult(rail Rail, body map[string]any, cfg DynRemoteAuthConfig) (statusCode int, ok bool, user flow.User) {
	dv, found := readDottedPath(body, cfg.DecisionField)
	if !found {
		rail.Warnf("Remote auth: decision field '%v' not found in response, body: %v", cfg.DecisionField, body)
		return http.StatusBadGateway, false, flow.User{}
	}
	user = flow.User{
		Username: readDottedPathStr(body, cfg.User.Username),
		UserNo:   readDottedPathStr(body, cfg.User.UserNo),
		RoleNo:   readDottedPathStr(body, cfg.User.RoleNo),
		Role:     readDottedPathStr(body, cfg.User.Role),
	}
	if !cast.ToBool(dv) {
		// denied: 403 if an authenticated user is present in the response, otherwise 401
		if user.Username != "" || user.UserNo != "" || user.RoleNo != "" || user.Role != "" {
			return http.StatusForbidden, false, user
		}
		return http.StatusUnauthorized, false, user
	}
	return 0, true, user
}

// readDottedPath reads a value from a decoded JSON object using a dotted path, e.g. "data.valid".
func readDottedPath(m map[string]any, path string) (any, bool) {
	if path == "" || m == nil {
		return nil, false
	}
	var cur any = m
	for _, seg := range strings.Split(path, ".") {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := mm[seg]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

func readDottedPathStr(m map[string]any, path string) string {
	if v, ok := readDottedPath(m, path); ok {
		return cast.ToString(v)
	}
	return ""
}

// Load DynAuthRoute from configuration.
//
// E.g.,
//
//	root-prop:
//	  - name: "myauth1"
//	    type: "bearer"
//	    bearer: "mybearer1"
//	    trace:
//	      username: "trace-user"
//	      role: "myrole"
//	    path-patterns:
//	      - "/path1"
//	      - "/path2"
//	      - "/path3"
//	  - name: "myauth2"
//	    type: "basic"
//	    username: "myuser"
//	    password: "mypassword"
//	    trace:
//	      username: "trace-user"
//	      role: "myrole"
//	    path-patterns:
//	      - "/path4"
//	      - "/path5"
//	      - "/path6"
//	  - name: "myauth3"
//	    type: "remote"
//	    path-patterns: # may be omitted, remote routes without path-patterns match all paths
//	      - "/path7"
//	    remote:
//	      path: "lb://auth-service/open/api/auth/check" # lb:// prefix resolves the address via service discovery
//	      body-map:
//	        authorization: "token"
//	        path: "url"
//	        method: "method"
//	      decision-field: "data.valid" # dotted path into the response body, doesn't have to be GnResp-wrapped
//	      user:
//	        userno: "data.userno"
//	        username: "data.username"
//	        roleno: "data.roleno"
//	        role: "data.role"
func (h *HttpProxy) LoadDynAuthRouteFromProp(rootProp string) []DynAuthRoute {
	p := UnmarshalFromPropKeyAs[[]DynAuthRoute](rootProp)
	return filterDynAuthRoutes(p)
}

type ReqTimeLogUnit struct {
	Dur  time.Duration
	Name string
}

func (h *HttpProxy) AddReqTimeLogFilter(exclPath func(proxyPath string) bool, unit ...ReqTimeLogUnit) {
	h.AddFilter(func(pc *ProxyContext, next func()) {
		_, r := pc.Inb.Unwrap()

		if exclPath(pc.ProxyPath) {
			next()
			return
		}

		start := time.Now()
		pc.Rail.Infof("Receive '%v %v' [%v]", r.Method, r.RequestURI, r.RemoteAddr)
		next()
		took := time.Since(start)
		u, ok := slutil.First(unit)
		if ok {
			pc.Rail.Infof("Processed '%v %v' [%.4f%v]", r.Method, r.RequestURI, float64(took/u.Dur), u.Name)
		} else {
			pc.Rail.Infof("Processed '%v %v' [%v]", r.Method, r.RequestURI, took)
		}
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
			case "/debug/trace/recorder/snapshot":
				HandleFlightRecorderSnapshot(pc.Inb)
			default:
				pc.Inb.Status(404)
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

	healthy := true
	var prevCheck time.Time
	prevMu := &sync.Mutex{}

	h.AddPathFilter([]string{hcUrl}, func(pc *ProxyContext, next func()) {
		prevMu.Lock()
		defer prevMu.Unlock()

		// rate limit, once per second
		if prevCheck.IsZero() || time.Since(prevCheck) > time.Second*1 {
			healthy = IsHealthcheckPass(*pc.Rail)
			prevCheck = time.Now()
		}

		// check if instance is healthy,  we don't really return anything, so it's fine to expose it
		w, _ := pc.Inb.Unwrap()
		if healthy {
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
	copy := slutil.Copy(pi)
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
