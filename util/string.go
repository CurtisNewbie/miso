package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/spf13/cast"
)

var (
	namedFmtPat = regexp.MustCompile(`\${[a-zA-Z0-9\\-\\_\.]+}`)
)

func PadNum(n int, digit int) string {
	var cnt int
	var v int = n
	for v > 0 {
		cnt += 1
		v /= 10
	}
	pad := digit - cnt
	num := strconv.Itoa(n)
	if pad > 0 {
		if pad == digit {
			return strings.Repeat("0", pad)
		}
		return strings.Repeat("0", pad) + num
	}
	return num
}

// Check if the string is blank
func IsBlankStr(s string) bool {
	return s == "" || strings.TrimSpace(s) == ""
}

// Substring such that len(s) <= max
func MaxLenStr(s string, max int) string {
	ru := []rune(s)
	return string(ru[:MinInt(len(ru), max)])
}

// Check if s has the prefix in a case-insensitive way.
func HasPrefixIgnoreCase(s string, prefix string) bool {
	prefix = strings.ToLower(prefix)
	s = strings.ToLower(s)
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// Check if s has the suffix in a case-insensitive way.
func HasSuffixIgnoreCase(s string, suffix string) bool {
	suffix = strings.ToLower(suffix)
	s = strings.ToLower(s)
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func ToStr(v any) string {
	switch v.(type) {
	case float32, float64:
		return fmt.Sprintf("%f", v)
	}
	return fmt.Sprintf("%v", v)
}

func Spaces(count int) string {
	return strings.Repeat(" ", count)
}

func Tabs(count int) string {
	return strings.Repeat("\t", count)
}

type SLPinter struct {
	*strings.Builder
	LineSuffix string
	LinePrefix string
}

func (s *SLPinter) Printf(st string, args ...any) {
	if s.Builder == nil {
		s.Builder = &strings.Builder{}
	}
	s.Builder.WriteString(fmt.Sprintf(st, args...))
}

func (s *SLPinter) Printlnf(st string, args ...any) {
	if s.Builder == nil {
		s.Builder = &strings.Builder{}
	}
	if s.Builder.Len() > 0 {
		s.Builder.WriteString(s.LineSuffix + "\n")
	}
	s.Builder.WriteString(s.LinePrefix + fmt.Sprintf(st, args...))
}

type IndWritef = func(indentCnt int, pat string, args ...any)

// Wrap strings.Builder, and returns a Writef func that automatically adds indentation.
func NewIndWritef(indentStr string) (*strings.Builder, IndWritef) {
	sb := strings.Builder{}
	return &sb, func(indentCnt int, pat string, args ...any) {
		sb.WriteString(strings.Repeat(indentStr, indentCnt) + fmt.Sprintf(pat+"\n", args...))
	}
}

type IndentWriter struct {
	*strings.Builder
	writef    IndWritef
	indentc   int
	indentStr string
}

func (i *IndentWriter) SetIndent(ind int) {
	i.indentc = ind
}

func (i *IndentWriter) IncrIndent() {
	i.indentc += 1
}

func (i *IndentWriter) DecrIndent() {
	i.indentc -= 1
}

func (i *IndentWriter) StepIn(f func(iw *IndentWriter)) *IndentWriter {
	i.indentc += 1
	f(i)
	i.indentc -= 1
	return i
}

func (i *IndentWriter) Writef(pat string, args ...any) *IndentWriter {
	i.writef(i.indentc, pat, args...)
	return i
}

// Writef with indentation and without line break.
func (i *IndentWriter) NoLbWritef(pat string, args ...any) *IndentWriter {
	i.WriteString(strings.Repeat(i.indentStr, i.indentc) + fmt.Sprintf(pat, args...))
	return i
}

// Writef without indentation and with line break.
func (i *IndentWriter) NoIndWritef(pat string, args ...any) *IndentWriter {
	i.WriteString(fmt.Sprintf(pat, args...))
	return i
}

func NewIndentWriter(indentStr string) IndentWriter {
	b, writef := NewIndWritef(indentStr)
	iw := IndentWriter{
		Builder:   b,
		writef:    writef,
		indentc:   0,
		indentStr: indentStr,
	}
	return iw
}

func CamelCase(s string) string {
	upper := false
	b := strings.Builder{}
	for i, r := range s {
		if i == 0 {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		if r == '_' || r == '-' {
			upper = true
		} else {
			if upper {
				b.WriteRune(unicode.ToUpper(r))
				upper = false
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func LastNStr(s string, n int) string {
	ru := []rune(s)
	if len(ru) <= n {
		return s
	}
	return string(ru[len(ru)-n:])
}

// Format message using named args.
//
// Equivalent to NamedFmt(pat).Sprintf(p).
//
// e.g., '${startTime} ${message}'
func NamedSprintf(pat string, p map[string]any) string {
	return NamedFmt(pat).Sprintf(p)
}

// Create reuseable named variables string formatter.
func NamedFmt(pat string) *namedFmt {
	return &namedFmt{pat: pat}
}

type namedFmt struct {
	pat string
}

func (n *namedFmt) get(p map[string]any, k string) string {
	v, ok := p[k]
	if ok {
		return cast.ToString(v)
	}
	return ""
}

// Format message using named args.
//
// e.g., '${startTime} ${message}'
func (n *namedFmt) Sprintf(p map[string]any) string {
	return namedFmtPat.ReplaceAllStringFunc(n.pat, func(s string) string {
		r := []rune(s)
		key := string(r[2 : len(r)-1])
		return n.get(p, key)
	})
}

func FmtFloat(f float64, width int, precision int) string {
	if width == 0 && precision == 0 {
		return fmt.Sprintf("%f", f)
	}

	var ws string
	var ps string
	if width != 0 {
		ws = cast.ToString(width)
	}
	if precision != 0 {
		ps = cast.ToString(precision)
	}
	return fmt.Sprintf("%"+ws+"."+cast.ToString(ps)+"f", f)
}

func PadSpace(n int, s string) string {
	r := []rune(s)
	rl := len(r)
	an := n
	if n < 0 {
		an = n * -1
	}
	if len(r) >= an {
		return s
	}
	pad := an - rl
	if n < 0 {
		return s + strings.Repeat(" ", pad)
	}
	return strings.Repeat(" ", pad) + s
}

// Splist kv pair. Returns false if token is not found or key is absent.
func SplitKV(s string, token string) (string, string, bool) {
	tokens := strings.SplitN(s, ":", 2)
	var ok bool = false
	var k string
	var v string
	if len(tokens) > 1 {
		ok = true
		k = strings.TrimSpace(tokens[0])
		v = strings.TrimSpace(tokens[1])
	}
	if ok && k == "" { // e.g., ' : value'.
		return k, v, false
	}
	return k, v, ok
}
