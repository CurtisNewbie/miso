package llm

import (
	"regexp"
	"strings"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/util/csv"
	"github.com/curtisnewbie/miso/util/osutil"
	"github.com/curtisnewbie/miso/util/slutil"
)

func parseTag(finalPat *regexp.Regexp, halfPat *regexp.Regexp, s string) string {
	_, withoutThink := ParseThink(s)

	// Find all matches in the text without think blocks and use the last one
	allMatches := finalPat.FindAllStringSubmatch(withoutThink, -1)
	if len(allMatches) > 0 {
		lastMatch := allMatches[len(allMatches)-1]
		if len(lastMatch) > 1 {
			return lastMatch[1]
		}
	}

	// Fallback to partial pattern for incomplete tags (also without think blocks)
	fr := halfPat.FindStringSubmatch(withoutThink)
	if len(fr) > 1 {
		return fr[1]
	}
	return ""
}

// Extract content in <tag>...</tag>.
func MustTagExtractor(tag string) tagContentExtractor {
	t, err := TagExtractor(tag)
	if err != nil {
		panic(err)
	}
	return t
}

type tagContentExtractor func(answer string) (original string, extracted string)

func (t tagContentExtractor) Content(answer string) string {
	_, v := t(answer)
	return v
}

// Extract content in <tag>${content}</tag>.
//
// The content between the tags must not include xml closing tags, e.g., </xxx>.
func TagExtractor(tag string) (tagContentExtractor, error) {
	finalPat, err := regexp.Compile(`(?s)<` + tag + `>(.*?)<\/` + tag + `>`)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	partialPat, err := regexp.Compile(`<` + tag + `>([^<]*(?:<[^/][^<]*)*)`)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return func(answer string) (original string, extracted string) {
		return answer, parseTag(finalPat, partialPat, answer)
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

var thinkEnd = "</think>"
var thinkLen = len(thinkEnd)

// Parse </think> tag.
//
// thought contains anything before the </think> tag, including the </think> tag itself.
//
// answer contains anything after the </think> tag.
func ParseThink(m string) (thought string, answer string) {
	i := strings.LastIndex(m, thinkEnd)
	if i < 0 {
		return "", m
	}

	return strings.TrimSpace(m[:i]), strings.TrimSpace(m[i+thinkLen:])
}

// Rewrite CSV content in text. Filter blank columns and rows. Make the content more readable.
func ToCsvTxt(srcPath string, separator ...string) (string, error) {
	sep := slutil.VarArgAny(separator, func() string { return " | " })

	f, err := osutil.OpenRWFile(srcPath)
	if err != nil {
		return "", errs.Wrap(err)
	}
	defer f.Close()

	records, err := csv.ReadAllIgnoreEmpty(f)
	if err != nil {
		return "", errs.Wrap(err)
	}

	b := strings.Builder{}
	for _, r := range records {
		l := strings.Join(slutil.FilterEmptyStr()(r), sep)
		l = strings.ReplaceAll(l, "\n", " ")
		b.WriteString(l + "\n")
	}

	return b.String(), nil
}
