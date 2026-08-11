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
