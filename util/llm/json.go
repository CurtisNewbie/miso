package llm

import (
	"github.com/curtisnewbie/miso/util/json"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/tailscale/hujson"
)

func ParseLLMJsonAs[T any](s string) (T, error) {
	s = StripMarkdownFence(s)
	if s == "" {
		var t T
		return t, nil
	}
	b, err := hujson.Standardize(strutil.UnsafeStr2Byt(s))
	if err != nil {
		var t T
		return t, err
	}
	p, err := json.SParseJsonAs[T](strutil.UnsafeByt2Str(b))
	if err != nil {
		var t T
		return t, err
	}
	return p, nil
}
