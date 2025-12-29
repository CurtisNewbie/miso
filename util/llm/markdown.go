package llm

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	latexRegex = regexp.MustCompile(`(?:([^\\])\$|^()\$)`)
)

func PrettifyMarkdownDoc(markdown string) string {
	sb := strings.Builder{}
	sb.Grow(len(markdown))

	isNotSpecialChars := func(r rune) bool {
		switch r {
		case ' ', '\n', '\t', '*', '，', '：', '。', ',', ':', '.', '(', ')', '（', '）', '、', '"', '“', '”', '\'', '—':
			return false
		}
		return true
	}
	var prev rune
	for i, c := range markdown {
		if i == 0 {
			prev = c
			sb.WriteRune(c)
			continue
		}

		// insert extra space between english and chinese
		if unicode.Is(unicode.Han, c) != unicode.Is(unicode.Han, prev) && isNotSpecialChars(prev) && isNotSpecialChars(c) {
			sb.WriteRune(' ')
		}

		sb.WriteRune(c)
		prev = c
	}

	return sb.String()
}

func EscapeMarkdownLatex(markdown string) string {
	return latexRegex.ReplaceAllString(markdown, `$1\$`)
}
