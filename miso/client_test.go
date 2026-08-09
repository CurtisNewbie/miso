package miso

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientLbUrlSyntax(t *testing.T) {
	SetProp("client.addr.auth-service.host", "10.0.0.1")
	SetProp("client.addr.auth-service.port", 8080)

	cases := []struct {
		name string
		url  string
		want string
	}{
		{name: "lb:// syntax", url: "lb://auth-service/api/users", want: "http://10.0.0.1:8080/api/users"},
		{name: "lb: syntax", url: "lb:auth-service/api/users", want: "http://10.0.0.1:8080/api/users"},
		{name: "lb:// without path", url: "lb://auth-service", want: "http://10.0.0.1:8080/"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got string
			client := NewClient(EmptyRail(), c.url).UseClient(&http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					got = req.URL.String()
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
				}),
			})
			resp := client.Get()
			defer resp.Close()
			if resp.Err != nil {
				t.Fatalf("unexpected error: %v", resp.Err)
			}
			if got != c.want {
				t.Fatalf("resolved url = '%v', want '%v'", got, c.want)
			}
		})
	}
}

func TestClientLbUrlMalformed(t *testing.T) {
	resp := NewClient(EmptyRail(), "lb://").Get()
	defer resp.Close()
	if resp.Err == nil {
		t.Fatal("expected error for empty service name")
	}
}
