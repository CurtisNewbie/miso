package miso

import (
	"net/http"
	"strings"
)

const (
	Bearer = "Bearer"
)

func ParseBearer(authorization string) (string, bool) {
	authorization = strings.TrimSpace(authorization)
	if authorization == "" {
		return "", false
	}
	if !strings.HasPrefix(authorization, Bearer) {
		return "", false
	}
	return strings.TrimSpace(authorization[len(Bearer):]), true
}

func BearerAuth(delegate http.Handler, getExpectedToken func() string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		token, ok := ParseBearer(authorization)
		if !ok || token != getExpectedToken() {
			Debugf("Bearer authorization failed, missing bearer token or token mismatch, %v %v",
				r.Method, r.RequestURI)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		delegate.ServeHTTP(w, r)
	}
}
