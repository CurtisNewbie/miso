package llm

import (
	"regexp"
	"strings"
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
func WithTagExtracted(tag string, f func(original string, extracted string)) func(answer string) {
	finalPat := regexp.MustCompile(`(?s)<` + tag + `>(.*)<\/` + tag + `>`)
	halfPat := regexp.MustCompile(`(?s)<` + tag + `>(.*)`)
	return func(answer string) {
		f(answer, parseTag(finalPat, halfPat, answer))
	}
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
