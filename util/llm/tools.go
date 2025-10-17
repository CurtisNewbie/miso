package llm

import (
	"regexp"
	"strings"

	"github.com/curtisnewbie/miso/util/errs"
)

func parseTag(finalPat *regexp.Regexp, halfPat *regexp.Regexp, s string) string {
	fr := finalPat.FindStringSubmatch(s)
	if len(fr) < 1 {
		fr = halfPat.FindStringSubmatch(s)
		if len(fr) < 1 {
			return ""
		}
		return fr[1]
	}
	return fr[1]
}

// Extract content in <tag>...</tag>.
func MustTagExtractor(tag string) func(answer string) (original string, extracted string) {
	t, err := TagExtractor(tag)
	if err != nil {
		panic(err)
	}
	return t
}

// Extract content in <tag>...</tag>.
func TagExtractor(tag string) (func(answer string) (original string, extracted string), error) {
	finalPat, err := regexp.Compile(`(?s)<` + tag + `>(.*)<\/` + tag + `>`)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	halfPat, err := regexp.Compile(`(?s)<` + tag + `>(.*)`)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return func(answer string) (original string, extracted string) {
		return answer, parseTag(finalPat, halfPat, answer)
	}, nil
}

func NewMsgDelta() *MsgDelta {
	return &MsgDelta{}
}

type MsgDelta struct {
	prev string
}

func (m *MsgDelta) Delta(s string) string {
	t, _ := strings.CutPrefix(s, m.prev)
	m.prev = s
	return t
}

func (m *MsgDelta) Complete() string {
	return m.prev
}
