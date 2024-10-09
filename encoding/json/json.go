package json

import (
	"io"
	"unicode"

	"github.com/curtisnewbie/miso/util"
	jsoniter "github.com/json-iterator/go"
	"github.com/json-iterator/go/extra"
)

func init() {
	extra.SetNamingStrategy(LowercaseNamingStrategy)
}

// Parse json bytes.
func ParseJson(body []byte, ptr any) error {
	e := jsoniter.Unmarshal(body, ptr)
	return e
}

// Parse json string.
func SParseJson(body string, ptr any) error {
	return ParseJson(util.UnsafeStr2Byt(body), ptr)
}

// Write json as bytes.
func WriteJson(body any) ([]byte, error) {
	return jsoniter.Marshal(body)
}

// Write json as string.
func SWriteJson(body any) (string, error) {
	if v, ok := body.(string); ok {
		return v, nil
	}
	buf, err := WriteJson(body)
	if err != nil {
		return "", err
	}
	return util.UnsafeByt2Str(buf), nil
}

// Write json as string using customized jsoniter.Config.
func CustomSWriteJson(c jsoniter.API, body any) (string, error) {
	if v, ok := body.(string); ok {
		return v, nil
	}
	buf, err := c.Marshal(body)
	if err != nil {
		return "", err
	}
	return util.UnsafeByt2Str(buf), nil
}

// Decode json.
func DecodeJson(reader io.Reader, ptr any) error {
	return jsoniter.NewDecoder(reader).Decode(ptr)
}

// Encode json.
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
