package miso

import "testing"

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
