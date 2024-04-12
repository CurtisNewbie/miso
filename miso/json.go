package miso

import (
	"io"
	"unicode"

	jsoniter "github.com/json-iterator/go"
	"github.com/json-iterator/go/extra"
)

func init() {
	extra.SetNamingStrategy(LowercaseNamingStrategy)
}

// Parse JSON using jsoniter.
func ParseJson(body []byte, ptr any) error {
	e := jsoniter.Unmarshal(body, ptr)
	return e
}

// Write JSON using jsoniter.
func WriteJson(body any) ([]byte, error) {
	return jsoniter.Marshal(body)
}

// Write JSON as string using jsoniter.
func SWriteJson(body any) (string, error) {
	if v, ok := body.(string); ok {
		return v, nil
	}
	buf, err := WriteJson(body)
	if err != nil {
		return "", err
	}
	return UnsafeByt2Str(buf), nil
}

// Decode JSON using jsoniter.
func DecodeJson(reader io.Reader, ptr any) error {
	return jsoniter.NewDecoder(reader).Decode(ptr)
}

// Encode JSON using jsoniter.
func EncodeJson(writer io.Writer, body any) error {
	return jsoniter.NewEncoder(writer).Encode(body)
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
