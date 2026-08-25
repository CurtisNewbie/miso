package miso

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curtisnewbie/miso/flow"
	"github.com/gin-gonic/gin"
)

func TestDynAuthRouteCheckAuth(t *testing.T) {
	cases := []struct {
		name  string
		route DynAuthRoute
		auth  string
		want  bool
	}{
		// basic
		{name: "basic valid creds", route: DynAuthRoute{Type: "basic", Username: "u", Password: "p"}, auth: "Basic dTpw", want: true},
		{name: "basic wrong password", route: DynAuthRoute{Type: "basic", Username: "u", Password: "p"}, auth: "Basic dTp4", want: false},
		{name: "basic empty creds must not authenticate", route: DynAuthRoute{Type: "basic", Username: "", Password: ""}, auth: "Basic Og==", want: false},
		{name: "basic empty username", route: DynAuthRoute{Type: "basic", Username: "", Password: "p"}, auth: "Basic OnA=", want: false},
		{name: "basic empty password", route: DynAuthRoute{Type: "basic", Username: "u", Password: ""}, auth: "Basic dTo=", want: false},
		{name: "basic missing header", route: DynAuthRoute{Type: "basic", Username: "u", Password: "p"}, auth: "", want: false},
		// bearer
		{name: "bearer valid", route: DynAuthRoute{Type: "bearer", Bearer: "tok"}, auth: "Bearer tok", want: true},
		{name: "bearer mismatch", route: DynAuthRoute{Type: "bearer", Bearer: "tok"}, auth: "Bearer other", want: false},
		{name: "bearer empty token in route", route: DynAuthRoute{Type: "bearer", Bearer: ""}, auth: "Bearer x", want: false},
		// remote routes are never checked via CheckAuth, must fail closed
		{name: "remote route fails closed", route: DynAuthRoute{Type: "remote"}, auth: "Bearer x", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.route.CheckAuth(&dynAuthReq{auth: c.auth}); got != c.want {
				t.Fatalf("CheckAuth() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestIsSuspiciousProxyPath(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		// traversal vectors: rejected
		{name: "single .. segment", path: "/open/../admin", want: true},
		{name: "nested .. segments", path: "/open/api/../admin/delete-all-users", want: true},
		{name: "escapes above root", path: "/../../etc/passwd", want: true},
		{name: "leading .. segment", path: "/../admin", want: true},
		{name: "path-param variant ..;", path: "/open/api/..;/admin", want: true},
		{name: "double dot prefix", path: "/open/..hidden", want: true},
		{name: "backslash", path: "/open\\admin", want: true},
		{name: "backslash segment", path: "/open/api/..\\admin", want: true},

		// normal paths: accepted
		{name: "plain path", path: "/open/api/v1/products", want: false},
		{name: "root", path: "/", want: false},
		{name: "empty", path: "", want: false},
		{name: "double slash", path: "/open//api", want: false},
		{name: "health", path: "/health", want: false},
		{name: "single dot segment", path: "/open/api/./products", want: false},
		{name: "dotfile segment", path: "/.well-known/openid-configuration", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isSuspiciousProxyPath(c.path); got != c.want {
				t.Fatalf("isSuspiciousProxyPath(%q) = %v, want %v", c.path, got, c.want)
			}
		})
	}
}

func TestWithDynAuthCheckWWWAuthenticate(t *testing.T) {
	const wantChallenge = "Basic realm=\"Username and Password\""
	const proxyPath = "/protected/foo"

	run := func(t *testing.T, authHeader string, routes []DynAuthRoute) (int, http.Header) {
		t.Helper()
		h := &HttpProxy{}
		checkAuth := h.WithDynAuthCheck(func() []DynAuthRoute { return routes })

		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, proxyPath, nil)
		if authHeader != "" {
			r.Header.Set("Authorization", authHeader)
		}
		rail := flow.NewRail(context.Background())
		inb := &Inbound{erail: rail, w: rec, r: r}
		pc := newProxyContext(&rail, inb)
		pc.ProxyPath = proxyPath

		code, ok := checkAuth(pc)
		if !ok {
			if code == 0 {
				code = http.StatusUnauthorized // AddAccessFilter default
			}
		} else {
			code = http.StatusOK
		}
		return code, rec.Header()
	}

	basicRoute := DynAuthRoute{Type: "basic", Username: "u", Password: "p", PathPatterns: []string{"/protected/**"}}
	bearerRoute := DynAuthRoute{Type: "bearer", Bearer: "tok", PathPatterns: []string{"/protected/**"}}

	t.Run("basic credentials rejected -> WWW-Authenticate challenge", func(t *testing.T) {
		code, hdr := run(t, "Basic dTp4", []DynAuthRoute{basicRoute}) // wrong password
		if code != http.StatusUnauthorized {
			t.Fatalf("status = %v, want %v", code, http.StatusUnauthorized)
		}
		if got := hdr.Get("WWW-Authenticate"); got != wantChallenge {
			t.Fatalf("WWW-Authenticate = %q, want %q", got, wantChallenge)
		}
	})

	t.Run("bearer rejected -> no basic challenge", func(t *testing.T) {
		code, hdr := run(t, "Bearer wrong", []DynAuthRoute{bearerRoute})
		if code != http.StatusUnauthorized {
			t.Fatalf("status = %v, want %v", code, http.StatusUnauthorized)
		}
		if got := hdr.Get("WWW-Authenticate"); got != "" {
			t.Fatalf("WWW-Authenticate = %q, want empty", got)
		}
	})

	t.Run("path not matched -> no challenge", func(t *testing.T) {
		code, hdr := run(t, "Basic dTp4", []DynAuthRoute{DynAuthRoute{Type: "basic", Username: "u", Password: "p", PathPatterns: []string{"/other/**"}}})
		if code != http.StatusUnauthorized {
			t.Fatalf("status = %v, want %v", code, http.StatusUnauthorized)
		}
		if got := hdr.Get("WWW-Authenticate"); got != "" {
			t.Fatalf("WWW-Authenticate = %q, want empty", got)
		}
	})

	t.Run("missing authorization header but basic path matched -> challenge", func(t *testing.T) {
		code, hdr := run(t, "", []DynAuthRoute{basicRoute})
		if code != http.StatusUnauthorized {
			t.Fatalf("status = %v, want %v", code, http.StatusUnauthorized)
		}
		if got := hdr.Get("WWW-Authenticate"); got != wantChallenge {
			t.Fatalf("WWW-Authenticate = %q, want %q", got, wantChallenge)
		}
	})

	t.Run("missing authorization header and no basic route matched -> no challenge", func(t *testing.T) {
		code, hdr := run(t, "", []DynAuthRoute{bearerRoute})
		if code != http.StatusUnauthorized {
			t.Fatalf("status = %v, want %v", code, http.StatusUnauthorized)
		}
		if got := hdr.Get("WWW-Authenticate"); got != "" {
			t.Fatalf("WWW-Authenticate = %q, want empty", got)
		}
	})

	t.Run("basic credentials accepted -> no challenge", func(t *testing.T) {
		code, hdr := run(t, "Basic dTpw", []DynAuthRoute{basicRoute}) // valid creds
		if code != http.StatusOK {
			t.Fatalf("status = %v, want %v", code, http.StatusOK)
		}
		if got := hdr.Get("WWW-Authenticate"); got != "" {
			t.Fatalf("WWW-Authenticate = %q, want empty", got)
		}
	})

	t.Run("basic path matched but rejected, bearer route authenticates -> no challenge", func(t *testing.T) {
		code, hdr := run(t, "Bearer tok", []DynAuthRoute{basicRoute, bearerRoute})
		if code != http.StatusOK {
			t.Fatalf("status = %v, want %v", code, http.StatusOK)
		}
		if got := hdr.Get("WWW-Authenticate"); got != "" {
			t.Fatalf("WWW-Authenticate = %q, want empty", got)
		}
	})
}

func TestWithDynAuthCheckBasicTraceUsername(t *testing.T) {
	const proxyPath = "/protected/foo"

	run := func(t *testing.T, route DynAuthRoute) string {
		t.Helper()
		h := &HttpProxy{}
		checkAuth := h.WithDynAuthCheck(func() []DynAuthRoute { return []DynAuthRoute{route} })

		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, proxyPath, nil)
		r.Header.Set("Authorization", "Basic dTpw") // u:p
		rail := flow.NewRail(context.Background())
		inb := &Inbound{erail: rail, w: rec, r: r}
		pc := newProxyContext(&rail, inb)
		pc.ProxyPath = proxyPath

		if _, ok := checkAuth(pc); !ok {
			t.Fatalf("expected auth to pass")
		}
		return pc.Rail.Username()
	}

	t.Run("Trace.Username empty falls back to basic auth username", func(t *testing.T) {
		route := DynAuthRoute{Type: "basic", Username: "u", Password: "p", PathPatterns: []string{"/protected/**"}}
		if got := run(t, route); got != "u" {
			t.Fatalf("trace username = %q, want %q", got, "u")
		}
	})

	t.Run("Trace.Username set takes precedence over basic auth username", func(t *testing.T) {
		route := DynAuthRoute{Type: "basic", Username: "u", Password: "p", PathPatterns: []string{"/protected/**"}, Trace: DynAuthTrace{Username: "trace-user"}}
		if got := run(t, route); got != "trace-user" {
			t.Fatalf("trace username = %q, want %q", got, "trace-user")
		}
	})

	t.Run("bearer route with empty Trace.Username does not fall back", func(t *testing.T) {
		route := DynAuthRoute{Type: "bearer", Bearer: "tok", PathPatterns: []string{"/protected/**"}}
		h := &HttpProxy{}
		checkAuth := h.WithDynAuthCheck(func() []DynAuthRoute { return []DynAuthRoute{route} })

		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, proxyPath, nil)
		r.Header.Set("Authorization", "Bearer tok")
		rail := flow.NewRail(context.Background())
		inb := &Inbound{erail: rail, w: rec, r: r}
		pc := newProxyContext(&rail, inb)
		pc.ProxyPath = proxyPath

		if _, ok := checkAuth(pc); !ok {
			t.Fatalf("expected auth to pass")
		}
		if got := pc.Rail.Username(); got != "" {
			t.Fatalf("trace username = %q, want empty", got)
		}
	})
}

func TestWithDynAuthCheckPathPatternGroup(t *testing.T) {
	const proxyPath = "/protected/foo"

	run := func(t *testing.T, authHeader string, routes []DynAuthRoute, loadGroups ...func() []PathPatternGroup) bool {
		t.Helper()
		h := &HttpProxy{}
		checkAuth := h.WithDynAuthCheck(func() []DynAuthRoute { return routes }, loadGroups...)

		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, proxyPath, nil)
		if authHeader != "" {
			r.Header.Set("Authorization", authHeader)
		}
		rail := flow.NewRail(context.Background())
		inb := &Inbound{erail: rail, w: rec, r: r}
		pc := newProxyContext(&rail, inb)
		pc.ProxyPath = proxyPath

		_, ok := checkAuth(pc)
		return ok
	}

	t.Run("basic route matches via group only", func(t *testing.T) {
		route := DynAuthRoute{Type: "basic", Username: "u", Password: "p", PathPatternGroup: "g1"}
		loadGroups := func() []PathPatternGroup {
			return []PathPatternGroup{{Name: "g1", PathPatterns: []string{"/protected/**"}}}
		}
		if ok := run(t, "Basic dTpw", []DynAuthRoute{route}, loadGroups); !ok {
			t.Fatalf("expected match via group, got no match")
		}
	})

	t.Run("basic route with unresolved group and no literal patterns never matches", func(t *testing.T) {
		route := DynAuthRoute{Type: "basic", Username: "u", Password: "p", PathPatternGroup: "missing"}
		if ok := run(t, "Basic dTpw", []DynAuthRoute{route}); ok {
			t.Fatalf("expected no match, unresolved group must fail closed")
		}
	})

	t.Run("basic route merges literal PathPatterns and group patterns", func(t *testing.T) {
		route := DynAuthRoute{Type: "basic", Username: "u", Password: "p", PathPatterns: []string{"/other/**"}, PathPatternGroup: "g1"}
		loadGroups := func() []PathPatternGroup {
			return []PathPatternGroup{{Name: "g1", PathPatterns: []string{"/protected/**"}}}
		}
		if ok := run(t, "Basic dTpw", []DynAuthRoute{route}, loadGroups); !ok {
			t.Fatalf("expected match via merged group pattern")
		}
	})

	t.Run("remote route with unresolved group fails closed, does not match-all", func(t *testing.T) {
		route := DynAuthRoute{Type: "remote", PathPatternGroup: "missing"}
		if ok := run(t, "Bearer x", []DynAuthRoute{route}); ok {
			t.Fatalf("expected no match, unresolved group on remote route must fail closed, not match-all")
		}
	})

	t.Run("duplicate group name across loaders, first loader wins", func(t *testing.T) {
		route := DynAuthRoute{Type: "basic", Username: "u", Password: "p", PathPatternGroup: "g1"}
		loadA := func() []PathPatternGroup {
			return []PathPatternGroup{{Name: "g1", PathPatterns: []string{"/protected/**"}}}
		}
		loadB := func() []PathPatternGroup {
			return []PathPatternGroup{{Name: "g1", PathPatterns: []string{"/other/**"}}}
		}
		if ok := run(t, "Basic dTpw", []DynAuthRoute{route}, loadA, loadB); !ok {
			t.Fatalf("expected match via first loader's group pattern")
		}
	})
}

func TestLoadDynAuthRoutePathPatternGroupRequirement(t *testing.T) {
	SetProp("test.dynauth.pprule", []any{
		map[string]any{
			"name":               "basic-with-group-only",
			"type":               "basic",
			"username":           "u",
			"password":           "p",
			"path-pattern-group": "shared",
		},
		map[string]any{
			"name":     "basic-without-any-selector",
			"type":     "basic",
			"username": "u",
			"password": "p",
		},
		map[string]any{
			"name":   "bearer-without-any-selector",
			"type":   "bearer",
			"bearer": "tok",
		},
	})

	h := &HttpProxy{}
	routes := h.LoadDynAuthRouteFromProp("test.dynauth.pprule")
	// only the route with path-pattern-group set satisfies the "at least one path selector" rule;
	// basic/bearer routes without PathPatterns nor PathPatternGroup must be filtered out
	if len(routes) != 1 {
		t.Fatalf("expected exactly 1 valid route, got %v: %+v", len(routes), routes)
	}
	if routes[0].Name != "basic-with-group-only" || routes[0].PathPatternGroup != "shared" {
		t.Fatalf("unexpected route kept: %+v", routes[0])
	}
}

func TestAddConfDynAccessFilterPathPatternGroup(t *testing.T) {
	SetProp("test.dynaccess.group", map[string]any{
		"whitelist": []string{},
		"path-pattern-groups": []any{
			map[string]any{
				"name":          "shared",
				"path-patterns": []string{"/protected/**"},
			},
		},
		"auth-routes": []any{
			map[string]any{
				"name":               "grouped-basic",
				"type":               "basic",
				"username":           "u",
				"password":           "p",
				"path-pattern-group": "shared",
			},
		},
	})

	h := &HttpProxy{}
	h.AddConfDynAccessFilter("test.dynaccess.group", 0)
	if len(h.filters) != 1 {
		t.Fatalf("expected 1 filter registered, got %d", len(h.filters))
	}

	// config keys must unmarshal into DynAccessFilterConfig, including the new fields
	cfg := UnmarshalFromPropKeyAs[DynAccessFilterConfig]("test.dynaccess.group")
	if len(cfg.PathPatternGroups) != 1 || cfg.PathPatternGroups[0].Name != "shared" || len(cfg.PathPatternGroups[0].PathPatterns) != 1 {
		t.Fatalf("path-pattern-groups not unmarshaled correctly: %+v", cfg.PathPatternGroups)
	}
	if len(cfg.AuthRoutes) != 1 || cfg.AuthRoutes[0].PathPatternGroup != "shared" {
		t.Fatalf("auth route path-pattern-group not unmarshaled correctly: %+v", cfg.AuthRoutes)
	}

	// end-to-end: the registered filter must resolve the auth route's PathPatternGroup against
	// the path-pattern-groups loaded from the same config, and authenticate accordingly
	run := func(t *testing.T, authHeader, path string) int {
		t.Helper()
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if authHeader != "" {
			r.Header.Set("Authorization", authHeader)
		}
		rail := flow.NewRail(context.Background())
		inb := &Inbound{erail: rail, w: rec, r: r}
		pc := newProxyContext(&rail, inb)
		pc.ProxyPath = path
		nextCalled := false
		h.filters[0](pc, func() { nextCalled = true })
		if nextCalled {
			return http.StatusOK
		}
		return rec.Code
	}

	if code := run(t, "Basic dTpw", "/protected/foo"); code != http.StatusOK {
		t.Fatalf("valid creds on grouped path: code = %v, want 200 (next called)", code)
	}
	if code := run(t, "Basic dTp4", "/protected/foo"); code != http.StatusUnauthorized {
		t.Fatalf("wrong creds on grouped path: code = %v, want 401", code)
	}
	if code := run(t, "Basic dTpw", "/unrelated/foo"); code != http.StatusUnauthorized {
		t.Fatalf("valid creds but path outside group: code = %v, want 401", code)
	}
}

func TestAddAccessFilterOnRejectHook(t *testing.T) {
	run := func(t *testing.T, checkAuthOk bool, checkAuthCode int, hooks ...AccessFilterHooks) int {
		t.Helper()
		h := &HttpProxy{}
		h.AddAccessFilter(func() []string { return nil }, func(pc *ProxyContext) (int, bool) {
			return checkAuthCode, checkAuthOk
		}, hooks...)

		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/protected/foo", nil)
		rail := flow.NewRail(context.Background())
		inb := &Inbound{erail: rail, w: rec, r: r}
		pc := newProxyContext(&rail, inb)
		pc.ProxyPath = "/protected/foo"

		nextCalled := false
		h.filters[0](pc, func() { nextCalled = true })
		if nextCalled {
			return http.StatusOK
		}
		return rec.Code
	}

	t.Run("no hook -> default 401", func(t *testing.T) {
		if code := run(t, false, 0); code != http.StatusUnauthorized {
			t.Fatalf("code = %v, want 401", code)
		}
	})

	t.Run("hook overrides status code on reject", func(t *testing.T) {
		hooks := AccessFilterHooks{OnReject: func() int { return http.StatusForbidden }}
		if code := run(t, false, 0, hooks); code != http.StatusForbidden {
			t.Fatalf("code = %v, want 403 (hook override)", code)
		}
	})

	t.Run("hook returning 0 does not override checkAuth status", func(t *testing.T) {
		hooks := AccessFilterHooks{OnReject: func() int { return 0 }}
		if code := run(t, false, http.StatusPaymentRequired, hooks); code != http.StatusPaymentRequired {
			t.Fatalf("code = %v, want 402 (checkAuth status preserved)", code)
		}
	})

	t.Run("hook not called when access allowed", func(t *testing.T) {
		called := false
		hooks := AccessFilterHooks{OnReject: func() int { called = true; return http.StatusForbidden }}
		if code := run(t, true, 0, hooks); code != http.StatusOK {
			t.Fatalf("code = %v, want 200", code)
		}
		if called {
			t.Fatalf("OnReject must not be called when access is allowed")
		}
	})
}

func TestAddConfDynAccessFilterOnRejectHook(t *testing.T) {
	SetProp("test.dynaccess.hook", map[string]any{
		"whitelist": []string{},
		"auth-routes": []any{
			map[string]any{
				"name":          "basic1",
				"type":          "basic",
				"username":      "u",
				"password":      "p",
				"path-patterns": []string{"/protected/**"},
			},
		},
	})

	h := &HttpProxy{}
	hookCalled := false
	h.AddConfDynAccessFilter("test.dynaccess.hook", 0, AccessFilterHooks{
		OnReject: func() int { hookCalled = true; return http.StatusForbidden },
	})

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected/foo", nil)
	r.Header.Set("Authorization", "Basic dTp4") // wrong password
	rail := flow.NewRail(context.Background())
	inb := &Inbound{erail: rail, w: rec, r: r}
	pc := newProxyContext(&rail, inb)
	pc.ProxyPath = "/protected/foo"

	h.filters[0](pc, func() {})
	if !hookCalled {
		t.Fatalf("OnReject hook was not called")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code = %v, want 403 from hook override", rec.Code)
	}
}

// TestProxyWWWAuthenticateOnTheWire verifies end-to-end through a real gin engine that the
// WWW-Authenticate challenge is actually sent on the 401 response.
func TestProxyWWWAuthenticateOnTheWire(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &HttpProxy{}
	h.resolveTarget = func(rail Rail, proxyPath string) (string, error) {
		return "http://localhost:9999" + proxyPath, nil
	}
	h.AddAccessFilter(func() []string { return nil }, h.WithDynAuthCheck(func() []DynAuthRoute {
		return []DynAuthRoute{
			{Type: "basic", Username: "u", Password: "p", PathPatterns: []string{"/protected/**"}},
		}
	}))

	e := gin.New()
	e.Any("/proxy/*proxyPath", newRawTRouteHandler(h.proxyRequestHandler))

	run := func(authHeader string) (int, http.Header) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/proxy/protected/foo", nil)
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		e.ServeHTTP(rec, req)
		return rec.Code, rec.Header()
	}

	const wantChallenge = "Basic realm=\"Username and Password\""

	t.Run("wrong basic credentials", func(t *testing.T) {
		code, hdr := run("Basic dTp4")
		if code != http.StatusUnauthorized {
			t.Fatalf("status = %v, want %v", code, http.StatusUnauthorized)
		}
		if got := hdr.Get("WWW-Authenticate"); got != wantChallenge {
			t.Fatalf("WWW-Authenticate = %q, want %q", got, wantChallenge)
		}
	})

	t.Run("no credentials at all", func(t *testing.T) {
		code, hdr := run("")
		if code != http.StatusUnauthorized {
			t.Fatalf("status = %v, want %v", code, http.StatusUnauthorized)
		}
		if got := hdr.Get("WWW-Authenticate"); got != wantChallenge {
			t.Fatalf("WWW-Authenticate = %q, want %q", got, wantChallenge)
		}
	})
}
