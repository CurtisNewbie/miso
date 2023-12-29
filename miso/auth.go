package miso

import "strings"

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
