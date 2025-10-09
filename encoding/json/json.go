package json

import (
	"bytes"
	jso "encoding/json"
	"io"
	"strings"
	"unicode"

	"github.com/curtisnewbie/miso/util"
	jsoniter "github.com/json-iterator/go"
)

var (
	config                  = jsoniter.Config{EscapeHTML: true}.Froze()
	NamingStrategyTranslate = LowercaseNamingStrategy
)

func init() {
	config.RegisterExtension(&namingStrategyExtension{jsoniter.DummyExtension{}})
}

// Parse json bytes.
func ParseJson(body []byte, ptr any) error {
	e := config.Unmarshal(body, ptr)
	return e
}

// Parse json bytes.
func ParseJsonAs[T any](body []byte) (T, error) {
	var t T
	return t, ParseJson(body, &t)
}

// Parse json bytes.
func SParseJsonAs[T any](body string) (T, error) {
	var t T
	return t, SParseJson(body, &t)
}

// Parse json string.
func SParseJson(body string, ptr any) error {
	return ParseJson(util.UnsafeStr2Byt(body), ptr)
}

// Write json as bytes.
func WriteJson(body any) ([]byte, error) {
	return config.Marshal(body)
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

// Write json as string.
func TrySWriteJson(body any) string {
	if v, ok := body.(string); ok {
		return v
	}
	buf, err := WriteJson(body)
	if err != nil {
		return ""
	}
	return util.UnsafeByt2Str(buf)
}

func SWriteIndent(body any) (string, error) {
	if v, ok := body.(string); ok {
		return v, nil
	}
	buf, err := config.MarshalIndent(body, "", "  ")
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
	return config.NewDecoder(reader).Decode(ptr)
}

// Encode json.
func EncodeJson(writer io.Writer, body any) error {
	return config.NewEncoder(writer).Encode(body)
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

type namingStrategyExtension struct {
	jsoniter.DummyExtension
}

func (extension *namingStrategyExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	for _, binding := range structDescriptor.Fields {
		if unicode.IsLower(rune(binding.Field.Name()[0])) || binding.Field.Name()[0] == '_' {
			continue
		}
		tag, hastag := binding.Field.Tag().Lookup("json")
		if hastag {
			tagParts := strings.Split(tag, ",")
			if tagParts[0] == "-" {
				continue // hidden field
			}
			if tagParts[0] != "" {
				continue // field explicitly named
			}
		}
		binding.ToNames = []string{NamingStrategyTranslate(binding.Field.Name())}
		binding.FromNames = []string{NamingStrategyTranslate(binding.Field.Name())}
	}
}

func IsValidJson(s []byte) bool {
	return config.Valid(s)
}

func IsValidJsonStr(s string) bool {
	return IsValidJson(util.UnsafeStr2Byt(s))
}

func Indent(b []byte) string {
	var buf bytes.Buffer
	_ = jso.Indent(&buf, b, "", "\t")
	return buf.String()
}

func SIndent(b string) string {
	return Indent(util.UnsafeStr2Byt(b))
}

func EscapeString(s string) string {
	b, err := jso.Marshal(s)
	if err != nil {
		return s
	}
	s = string(b)
	return s[1 : len(s)-1]
}
