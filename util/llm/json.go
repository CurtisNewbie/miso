package llm

import (
	jsonrepair "github.com/RealAlexandreAI/json-repair"
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
	if err == nil {
		p, err := json.SParseJsonAs[T](strutil.UnsafeByt2Str(b))
		if err == nil {
			return p, nil
		}
	}
	// hujson or json parse failed; attempt json-repair on the original string
	fixed, err := jsonrepair.RepairJSON(s)
	if err != nil {
		var t T
		return t, err
	}
	return json.SParseJsonAs[T](fixed)
}
