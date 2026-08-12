package miso

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/curtisnewbie/miso/flow"
	"github.com/spf13/cast"
)

func TestReadDottedPath(t *testing.T) {
	cases := []struct {
		name string
		m    map[string]any
		path string
		want any
		ok   bool
	}{
		{name: "nested map", m: map[string]any{"data": map[string]any{"valid": true}}, path: "data.valid", want: true, ok: true},
		{name: "deep nesting", m: map[string]any{"a": map[string]any{"b": map[string]any{"c": "deep"}}}, path: "a.b.c", want: "deep", ok: true},
		{name: "string value", m: map[string]any{"data": map[string]any{"username": "u1"}}, path: "data.username", want: "u1", ok: true},
		{name: "number value", m: map[string]any{"data": map[string]any{"count": 42}}, path: "data.count", want: 42, ok: true},
		{name: "missing path", m: map[string]any{"data": map[string]any{"valid": true}}, path: "data.missing", want: nil, ok: false},
		{name: "empty path", m: map[string]any{"data": map[string]any{"valid": true}}, path: "", want: nil, ok: false},
		{name: "nil map", m: nil, path: "data.valid", want: nil, ok: false},
		{name: "non-map segment", m: map[string]any{"data": "str"}, path: "data.valid", want: nil, ok: false},
		{name: "missing root", m: map[string]any{}, path: "data.valid", want: nil, ok: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := readDottedPath(c.m, c.path)
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v", ok, c.ok)
			}
			if !ok {
				return
			}
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}

	// readDottedPathStr on non-string values
	if got := readDottedPathStr(map[string]any{"data": map[string]any{"count": 42}}, "data.count"); got != "42" {
		t.Fatalf("readDottedPathStr(42) = '%v', want '42'", got)
	}
	if got := readDottedPathStr(map[string]any{"data": map[string]any{}}, "data.missing"); got != "" {
		t.Fatalf("readDottedPathStr(missing) = '%v', want ''", got)
	}

	// cast.ToBool on the decision value
	if !cast.ToBool(true) {
		t.Fatal("cast.ToBool(true) should be true")
	}
}

func TestLoadDynAuthRouteRemote(t *testing.T) {
	SetProp("test.dynremote.routes", []any{
		map[string]any{
			"name":          "remote-ok",
			"type":          "remote",
			"path-patterns": []string{"/api/**"},
			"remote": map[string]any{
				"path":           "lb://auth-service/open/api/auth",
				"decision-field": "data.valid",
				"body-map":       map[string]any{"authorization": "token", "path": "url", "method": "method"},
				"user":           map[string]any{"userno": "data.userNo", "username": "data.username", "roleno": "data.roleNo", "role": "data.role"},
			},
		},
		map[string]any{
			"name":          "remote-no-path",
			"type":          "remote",
			"path-patterns": []string{"/api/**"},
			"remote":        map[string]any{},
		},
		map[string]any{
			"name":          "remote-unknown-bodymap-key",
			"type":          "remote",
			"path-patterns": []string{"/api/**"},
			"remote": map[string]any{
				"path":           "lb://auth-service/open/api/auth",
				"decision-field": "data.valid",
				"body-map":       map[string]any{"bad-key": "token"},
			},
		},
		map[string]any{
			"name": "remote-no-path-patterns",
			"type": "remote",
			"remote": map[string]any{
				"path":           "lb://auth-service/open/api/auth",
				"decision-field": "data.valid",
			},
		},
		map[string]any{
			"name": "remote-ws-token",
			"type": "remote",
			"remote": map[string]any{
				"path":            "lb://auth-service/open/api/auth",
				"token-query-key": "token",
				"decision-field":  "data.valid",
				"body-map":        map[string]any{"token": "token"},
			},
		},
		map[string]any{
			"name": "remote-ws-token-no-body-map",
			"type": "remote",
			"remote": map[string]any{
				"path":            "lb://auth-service/open/api/auth",
				"token-query-key": "token",
				"decision-field":  "data.valid",
			},
		},
	})

	h := &HttpProxy{}
	routes := h.LoadDynAuthRouteFromProp("test.dynremote.routes")
	// remote-no-path is filtered out; unknown body-map key is ignored; remote without path-patterns is kept;
	// remote with token-query-key but no body-map token destination is filtered out
	if len(routes) != 4 {
		t.Fatalf("expected exactly 4 valid remote routes, got %v", len(routes))
	}

	d := routes[0]
	if d.Type != "remote" {
		t.Fatalf("type = '%v', want 'remote'", d.Type)
	}
	if d.Name != "remote-ok" {
		t.Fatalf("name = '%v', want 'remote-ok'", d.Name)
	}
	if d.Remote.Path != "lb://auth-service/open/api/auth" || d.Remote.DecisionField != "data.valid" {
		t.Fatalf("remote config not loaded correctly: %+v", d.Remote)
	}
	if d.Remote.BodyMap.Authorization != "token" || d.Remote.BodyMap.Path != "url" || d.Remote.BodyMap.Method != "method" {
		t.Fatalf("body-map not loaded correctly: %+v", d.Remote.BodyMap)
	}
	if d.Remote.User.UserNo != "data.userNo" || d.Remote.User.Username != "data.username" || d.Remote.User.RoleNo != "data.roleNo" || d.Remote.User.Role != "data.role" {
		t.Fatalf("user mapping not loaded correctly: %+v", d.Remote.User)
	}
	t.Logf("loaded route: %+v", d)

	d = routes[3]
	if d.Name != "remote-ws-token" {
		t.Fatalf("routes[3] = '%v', want 'remote-ws-token'", d.Name)
	}
	if d.Remote.TokenQueryKey != "token" || d.Remote.BodyMap.Token != "token" {
		t.Fatalf("ws token config not loaded correctly: %+v", d.Remote)
	}
}

func TestAddConfDynAccessFilter(t *testing.T) {
	SetProp("test.dynaccess", map[string]any{
		"whitelist": []string{"/health", "/open/api/**"},
		"auth-routes": []any{
			map[string]any{
				"name":          "remote-ok",
				"type":          "remote",
				"path-patterns": []string{"/api/**"},
				"remote": map[string]any{
					"path":           "lb://auth-service/open/api/auth",
					"decision-field": "data.valid",
				},
			},
			map[string]any{
				"name": "remote-no-decision-field",
				"type": "remote",
				"remote": map[string]any{
					"path": "lb://auth-service/open/api/auth",
				},
			},
		},
	})

	h := &HttpProxy{}
	h.AddConfDynAccessFilter("test.dynaccess", 0)
	if len(h.filters) != 1 {
		t.Fatalf("expected 1 filter registered, got %d", len(h.filters))
	}

	// config keys must unmarshal into DynAccessFilterConfig
	cfg := UnmarshalFromPropKeyAs[DynAccessFilterConfig]("test.dynaccess")
	if len(cfg.Whitelist) != 2 {
		t.Fatalf("expected 2 whitelist patterns, got %+v", cfg.Whitelist)
	}
	if len(cfg.AuthRoutes) != 2 {
		t.Fatalf("expected 2 auth routes, got %+v", cfg.AuthRoutes)
	}
}

func TestMapRemoteAuthResult(t *testing.T) {
	cfg := DynRemoteAuthConfig{
		DecisionField: "data.valid",
		User: DynRemoteUserMapping{
			UserNo:   "data.userno",
			Username: "data.username",
			RoleNo:   "data.roleno",
			Role:     "data.role",
		},
	}
	withUser := map[string]any{"data": map[string]any{"valid": true, "userno": "u1", "username": "un1", "roleno": "r1", "role": "admin"}}
	noUser := map[string]any{"data": map[string]any{"valid": true}}
	deniedWithUser := map[string]any{"data": map[string]any{"valid": false, "userno": "u1", "username": "un1", "roleno": "r1"}}
	deniedNoUser := map[string]any{"data": map[string]any{"valid": false}}

	cases := []struct {
		name     string
		data     map[string]any
		wantCode int
		wantOk   bool
		wantUser string // userno, empty means no user expected
	}{
		{name: "allowed with user", data: withUser, wantCode: 0, wantOk: true, wantUser: "u1"},
		{name: "allowed without user", data: noUser, wantCode: 0, wantOk: true, wantUser: ""},
		{name: "denied with user -> 403", data: deniedWithUser, wantCode: 403, wantOk: false, wantUser: "u1"},
		{name: "denied without user -> 401", data: deniedNoUser, wantCode: 401, wantOk: false, wantUser: ""},
		{name: "missing decision field -> 502", data: map[string]any{"data": map[string]any{}}, wantCode: 502, wantOk: false, wantUser: ""},
		{name: "decision as string", data: map[string]any{"data": map[string]any{"valid": "true", "userno": "u2"}}, wantCode: 0, wantOk: true, wantUser: "u2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, ok, user := mapRemoteAuthResult(EmptyRail(), c.data, cfg)
			if code != c.wantCode || ok != c.wantOk {
				t.Fatalf("code = %v, ok = %v, want %v, %v", code, ok, c.wantCode, c.wantOk)
			}
			if user.UserNo != c.wantUser {
				t.Fatalf("user.UserNo = '%v', want '%v'", user.UserNo, c.wantUser)
			}
		})
	}
}

func TestMapRemoteAuthResultPlainResponse(t *testing.T) {
	// plain (non-GnResp-wrapped) response body, paths resolved against the full body
	cfg := DynRemoteAuthConfig{
		DecisionField: "valid",
		User: DynRemoteUserMapping{
			UserNo:   "userno",
			Username: "username",
			RoleNo:   "roleno",
			Role:     "role",
		},
	}
	cases := []struct {
		name     string
		data     map[string]any
		wantCode int
		wantOk   bool
		wantUser string
	}{
		{name: "plain allowed", data: map[string]any{"valid": true, "userno": "u1", "username": "un1", "roleno": "r1", "role": "admin"}, wantCode: 0, wantOk: true, wantUser: "u1"},
		{name: "plain denied with user -> 403", data: map[string]any{"valid": false, "userno": "u1"}, wantCode: 403, wantOk: false, wantUser: "u1"},
		{name: "plain denied no user -> 401", data: map[string]any{"valid": false}, wantCode: 401, wantOk: false, wantUser: ""},
		{name: "plain missing decision field -> 502", data: map[string]any{"userno": "u1"}, wantCode: 502, wantOk: false, wantUser: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, ok, user := mapRemoteAuthResult(EmptyRail(), c.data, cfg)
			if code != c.wantCode || ok != c.wantOk {
				t.Fatalf("code = %v, ok = %v, want %v, %v", code, ok, c.wantCode, c.wantOk)
			}
			if user.UserNo != c.wantUser {
				t.Fatalf("user.UserNo = '%v', want '%v'", user.UserNo, c.wantUser)
			}
		})
	}
}

func TestMapRemoteAuthResultGnResp(t *testing.T) {
	// GnResp-wrapped response {error, errorCode, msg, data}, paths resolved against the full body
	cfg := DynRemoteAuthConfig{
		DecisionField: "data.valid",
		User: DynRemoteUserMapping{
			UserNo: "data.userno",
		},
	}
	cases := []struct {
		name     string
		data     map[string]any
		wantCode int
		wantOk   bool
		wantUser string
	}{
		{name: "wrapped allowed", data: map[string]any{"error": false, "data": map[string]any{"valid": true, "userno": "u1"}}, wantCode: 0, wantOk: true, wantUser: "u1"},
		{name: "wrapped denied -> 401", data: map[string]any{"error": false, "data": map[string]any{"valid": false}}, wantCode: 401, wantOk: false, wantUser: ""},
		{name: "wrapped missing decision field -> 502", data: map[string]any{"error": false, "data": map[string]any{}}, wantCode: 502, wantOk: false, wantUser: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, ok, user := mapRemoteAuthResult(EmptyRail(), c.data, cfg)
			if code != c.wantCode || ok != c.wantOk {
				t.Fatalf("code = %v, ok = %v, want %v, %v", code, ok, c.wantCode, c.wantOk)
			}
			if user.UserNo != c.wantUser {
				t.Fatalf("user.UserNo = '%v', want '%v'", user.UserNo, c.wantUser)
			}
		})
	}
}

func TestCheckRemoteAuthTokenQueryKey(t *testing.T) {
	// downstream auth service captures the request body and allows when the decision field is true
	var gotBody map[string]any
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"valid":true}}`))
	}))
	defer srv.Close()

	run := func(t *testing.T, cfg DynRemoteAuthConfig, setup func(r *http.Request)) (int, bool) {
		t.Helper()
		h := &HttpProxy{}
		rail := flow.NewRail(context.Background())
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/ws/chat?token=query-token", nil)
		if setup != nil {
			setup(r)
		}
		inb := &Inbound{erail: rail, w: rec, r: r}
		pc := newProxyContext(&rail, inb)
		pc.ProxyPath = "/ws/chat"
		return h.checkRemoteAuth(pc, DynAuthRoute{Type: "remote", Remote: cfg})
	}

	base := DynRemoteAuthConfig{
		Path:          srv.URL + "/auth/check",
		DecisionField: "data.valid",
		BodyMap: DynRemoteBodyMap{
			Token:  "token",
			Path:   "url",
			Method: "method",
		},
	}

	t.Run("token read from query param", func(t *testing.T) {
		gotBody = nil
		calls.Store(0)
		cfg := base
		cfg.TokenQueryKey = "token"
		code, ok := run(t, cfg, nil)
		if !ok || code != 0 {
			t.Fatalf("code = %v, ok = %v, want allowed", code, ok)
		}
		if calls.Load() != 1 {
			t.Fatalf("downstream called %v times, want 1", calls.Load())
		}
		if gotBody == nil || gotBody["token"] != "query-token" || gotBody["url"] != "/ws/chat" || gotBody["method"] != http.MethodGet {
			t.Fatalf("unexpected auth request body: %+v", gotBody)
		}
	})

	t.Run("missing query token -> 401, downstream not called", func(t *testing.T) {
		calls.Store(0)
		cfg := base
		cfg.TokenQueryKey = "missing-key"
		code, ok := run(t, cfg, nil)
		if ok || code != http.StatusUnauthorized {
			t.Fatalf("code = %v, ok = %v, want 401 denied", code, ok)
		}
		if calls.Load() != 0 {
			t.Fatalf("downstream called %v times, want 0", calls.Load())
		}
	})

	t.Run("authorization header still used when token-query-key empty", func(t *testing.T) {
		gotBody = nil
		calls.Store(0)
		cfg := base
		cfg.TokenQueryKey = ""
		cfg.BodyMap.Token = ""
		cfg.BodyMap.Authorization = "token"
		code, ok := run(t, cfg, func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer header-token")
		})
		if !ok || code != 0 {
			t.Fatalf("code = %v, ok = %v, want allowed", code, ok)
		}
		if calls.Load() != 1 {
			t.Fatalf("downstream called %v times, want 1", calls.Load())
		}
		if gotBody == nil || gotBody["token"] != "Bearer header-token" {
			t.Fatalf("unexpected auth request body: %+v", gotBody)
		}
	})
}
