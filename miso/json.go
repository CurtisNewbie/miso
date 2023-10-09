package miso

import (
	"unicode"

	jsoniter "github.com/json-iterator/go"
	"github.com/json-iterator/go/extra"
)

func init() {
	extra.SetNamingStrategy(LowercaseNamingStrategy)
}

// Parse JSON using jsoniter.
func ParseJson(body []byte, ptr any) error {
	e := jsoniter.Unmarshal([]byte(body), ptr)
	return e
}

// Write JSON using jsoniter.
func WriteJson(body any) ([]byte, error) {
	return jsoniter.Marshal(body)
}

// Change first rune to lower case.
func LowercaseNamingStrategy(name string) string {
	ru := []rune(name)
	if len(ru) < 1 {
		return name
	}
	ru[0] = unicode.ToLower(ru[0])
	return string(ru)
}
