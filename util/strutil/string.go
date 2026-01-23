package strutil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	doublestar "github.com/bmatcuk/doublestar/v4"
	"github.com/curtisnewbie/miso/util/constraint"
	"github.com/curtisnewbie/miso/util/pair"
	"github.com/curtisnewbie/miso/util/rfutil"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/utillog"
	"github.com/spf13/cast"
	"golang.org/x/text/width"
)

var (
	namedFmtPat = regexp.MustCompile(`\${[a-zA-Z0-9\/\-\_\. ]+}`)
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
	n := max
	l := len(ru)
	if l < max {
		n = l
	}
	return string(ru[:n])
}

// Check if s has the prefix in a case-insensitive way.
func HasPrefixIgnoreCase(s string, prefix string) bool {
	prefix = strings.ToLower(prefix)
	s = strings.ToLower(s)
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// Cut prefix from s in a case-insensitive way.
func CutPrefixIgnoreCase(s string, prefix string) (string, bool) {
	prefix = strings.ToLower(prefix)
	sl := strings.ToLower(s)
	ok := len(sl) >= len(prefix) && sl[0:len(prefix)] == prefix
	if ok {
		return s[len(prefix):], true
	}
	return s, false
}

// Cut any prefix from s in a case-insensitive way.
func CutPrefixIgnoreCaseAny(s string, prefix ...string) (string, bool) {
	for _, p := range prefix {
		if v, ok := CutPrefixIgnoreCase(s, p); ok {
			return v, true
		}
	}
	return s, false
}

// Check if s has the suffix in a case-insensitive way.
func HasSuffixIgnoreCase(s string, suffix string) bool {
	suffix = strings.ToLower(suffix)
	s = strings.ToLower(s)
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// Cut suffix from s in a case-insensitive way.
func CutSuffixIgnoreCase(s string, suffix string) (string, bool) {
	suffix = strings.ToLower(suffix)
	sl := strings.ToLower(s)
	ok := len(sl) >= len(suffix) && sl[len(sl)-len(suffix):] == suffix
	if ok {
		return s[:len(sl)-len(suffix)], true
	}
	return s, false
}

// Cut any suffix from s in a case-insensitive way.
func CutSuffixIgnoreCaseAny(s string, suffix ...string) (string, bool) {
	for _, p := range suffix {
		if v, ok := CutSuffixIgnoreCase(s, p); ok {
			return v, true
		}
	}
	return s, false
}

func ToStr(v any) string {
	return cast.ToString(v)
}

func Spaces(count int) string {
	return strings.Repeat(" ", count)
}

func Tabs(count int) string {
	return strings.Repeat("\t", count)
}

// Enhanced [strings.Builder].
//
// Use [NewBuilder] instantiate.
type Builder struct {
	*strings.Builder

	lineSuffix string
	linePrefix string

	indentc int
	indents string
}

func NewBuilder() *Builder {
	return &Builder{
		Builder: &strings.Builder{},
		indentc: 0,
		indents: "",
	}
}

func (b *Builder) buildIndent() string {
	return strings.Repeat(b.indents, b.indentc)
}

func (b *Builder) Printf(st string, args ...any) *Builder {
	b.Builder.WriteString(fmt.Sprintf(st, args...))
	return b
}

func (b *Builder) NewLine(st string) *Builder {
	b.Builder.WriteRune('\n')
	return b
}

// Add new content.
//
// Always add line break at the end of the line.
func (b *Builder) Println(st string) *Builder {
	b.Builder.WriteString(b.linePrefix)
	b.Builder.WriteString(b.buildIndent())
	b.Builder.WriteString(st)
	b.Builder.WriteString(b.lineSuffix)
	b.Builder.WriteRune('\n')
	return b
}

// Add new content.
//
// Only add line break at the start if the builder is not empty.
func (b *Builder) PPrintln(st string) *Builder {
	if b.Builder.Len() > 0 {
		b.Builder.WriteRune('\n')
	}
	b.Builder.WriteString(b.linePrefix)
	b.Builder.WriteString(b.buildIndent())
	b.Builder.WriteString(st)
	b.Builder.WriteString(b.lineSuffix)
	return b
}

// Add new formatted content.
//
// Always add line break at the end of the line.
func (b *Builder) Printlnf(st string, args ...any) *Builder {
	return b.Println(fmt.Sprintf(st, args...))
}

// Add new formatted content
//
// Only add line break at the start if the builder is not empty.
func (b *Builder) PPrintlnf(st string, args ...any) *Builder {
	return b.PPrintln(fmt.Sprintf(st, args...))
}

func (b *Builder) StepIn(f func(b *Builder)) *Builder {
	b.indentc += 1
	f(b)
	b.indentc -= 1
	return b
}

func (b *Builder) WithLinePrefix(p string) *Builder {
	b.linePrefix = p
	return b
}

func (b *Builder) WithLineSuffix(p string) *Builder {
	b.lineSuffix = p
	return b
}

func (b *Builder) WithIndentStr(ind string) {
	b.indents = ind
}

func (b *Builder) WithIndentCnt(ind int) {
	if ind < 0 {
		ind = 0
	}
	b.indentc = ind
}

func (b *Builder) WithIndent(s string, c int) *Builder {
	b.WithIndentStr(s)
	b.WithIndentCnt(c)
	return b
}

func (b *Builder) IncrIndent() *Builder {
	b.indentc += 1
	return b
}

func (b *Builder) DecrIndent() *Builder {
	if b.indentc < 1 {
		return b
	}
	b.indentc -= 1
	return b
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

func (s *SLPinter) Println(st string) {
	if s.Builder == nil {
		s.Builder = &strings.Builder{}
	}
	if s.Builder.Len() > 0 {
		s.Builder.WriteString(s.LineSuffix + "\n")
	}
	s.Builder.WriteString(s.LinePrefix + st)
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

// NoLbWritef(..) when condition is true else Writef(..).
func (i *IndentWriter) NoLbWritefWhen(condition bool, pat string, args ...any) *IndentWriter {
	if condition {
		i.NoLbWritef(pat, args...)
	} else {
		i.Writef(pat, args...)
	}
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

// Format message using named args, e.g., '${startTime} ${message}'
func NamedSprintf(pat string, p map[string]any) string {
	return namedFmtPat.ReplaceAllStringFunc(pat, func(s string) string {
		key := s[2 : len(s)-1]
		v := p[key]
		if vs, ok := v.(string); ok {
			return vs
		}
		return cast.ToString(v)
	})
}

// Format message using fields in struct, e.g., '${startTime} ${message}'
//
// Equivalent to following code:
//
//	NamedSprintf(pat, ReflectGenMap(myStruct))
func NamedSprintfv(pat string, v any) string {
	return NamedSprintf(pat, rfutil.ReflectGenMap(v))
}

// Format message using fields in struct, e.g., '${startTime} ${message}'
//
// e.g.,
//
//	NamedSprintfkv("my name is ${name}", "name", "slim shady")
func NamedSprintfkv[T constraint.BasicValue](pat string, kv ...T) string {
	if len(kv) < 1 {
		return pat
	}
	p := make([]pair.StrPair, 0, len(kv)/2)
	var last string
	for i, v := range kv {
		vs := cast.ToString(v)
		if i%2 == 0 {
			last = vs
		} else {
			p = append(p, pair.New(last, vs))
		}
	}
	return NamedSprintf(pat, slutil.MergeMapKVAny(p))
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

// Pad spaces.
//
// if n > 0, pad left, else pad right.
func PadSpace(n int, s string) string {
	return PadToken(n, s, " ")
}

func RuneWidth(r rune) int {
	k := width.LookupRune(r).Kind()
	switch k {
	case width.EastAsianWide, width.EastAsianFullwidth, width.EastAsianAmbiguous:
		return 2
	default:
		return 1
	}
}

func StrWidth(s string) int {
	n := 0
	for _, r := range s {
		n += RuneWidth(r)
	}
	return n
}

func PadToken(n int, s string, tok string) string {
	rl := StrWidth(s)
	an := n
	if n < 0 {
		an = n * -1
	}
	if rl >= an {
		return s
	}
	pad := an - rl
	if n < 0 {
		return s + strings.Repeat(tok, pad)
	}
	return strings.Repeat(tok, pad) + s
}

// Splist kv pair. Returns false if token is not found or key is absent.
func SplitKV(s string, token string) (string, string, bool) {
	tokens := strings.SplitN(s, token, 2)
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

func SAddLineIndent(s string, indentChar string) string {
	b := strings.Builder{}
	sp := strings.Split(s, "\n")
	for k, spt := range sp {
		if spt == "" {
			b.WriteString(spt)
		} else {
			b.WriteString(indentChar + spt)
		}
		if k < len(sp)-1 {
			b.WriteRune('\n')
		}
	}
	return b.String()
}

// Match any of the path pattern
//
// See [MatchPathAnyVal].
func MatchPathAny(pattern []string, s string) bool {
	for _, p := range pattern {
		if MatchPath(p, s) {
			return true
		}
	}
	return false
}

// Match any of the path pattern.
func MatchPathAnyVal(pattern []string, s string) (string, bool) {
	for _, p := range pattern {
		if MatchPath(p, s) {
			return p, true
		}
	}
	return "", false
}

func MatchPath(pattern, s string) bool {
	ok, err := doublestar.Match(pattern, s)
	if err != nil {
		utillog.ErrorLog("Path Pattern is invalid, pattern: '%v', %v", pattern, err)
		return false
	}
	return ok
}

func ContainsAnyStr(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func ContainsAnyStrIgnoreCase(s string, substrings ...string) bool {
	s = strings.ToLower(s)
	for _, sub := range substrings {
		if strings.Contains(s, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

func QuoteStr(s string) string {
	return "\"" + s + "\""
}

func UnquoteStr(s string) string {
	ru := []rune(s)
	if len(ru) < 2 {
		return s
	}
	r1 := ru[0]
	if (r1 == '"' || r1 == '\'') && ru[len(ru)-1] == r1 {
		return string(ru[1 : len(ru)-1])
	}
	return s
}

func EqualAnyStr(s string, canditates ...string) bool {
	for _, c := range canditates {
		if s == c {
			return true
		}
	}
	return false
}

func EqualAnyStrSlice(s string, canditates []string) bool {
	for _, c := range canditates {
		if s == c {
			return true
		}
	}
	return false
}

func EqualAnyFold(s string, canditates ...string) bool {
	for _, c := range canditates {
		if strings.EqualFold(s, c) {
			return true
		}
	}
	return false
}

func HasAnySuffix(s string, suf ...string) bool {
	for _, c := range suf {
		if strings.HasSuffix(s, c) {
			return true
		}
	}
	return false
}

func HasAnyPrefix(s string, suf ...string) bool {
	for _, c := range suf {
		if strings.HasPrefix(s, c) {
			return true
		}
	}
	return false
}

func TrimStrSlice(s []string) {
	slutil.UpdateSliceValue(s, func(v string) string {
		return strings.TrimSpace(v)
	})
}

func SplitStr(s, sep string) []string {
	tok := strings.Split(s, sep)
	cp := make([]string, 0, len(tok))
	for _, v := range tok {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		cp = append(cp, v)
	}
	return cp
}

func TrimSpace(s string, extra ...rune) string {
	if len(extra) < 1 {
		return strings.TrimSpace(s)
	}
	return strings.TrimFunc(s, func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		for _, ru := range extra {
			if ru == r {
				return true
			}
		}
		return false
	})
}

func TrimSpaceLeft(s string, extra ...rune) string {
	return strings.TrimLeftFunc(s, func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		for _, ru := range extra {
			if ru == r {
				return true
			}
		}
		return false
	})
}

func TrimSpaceRight(s string, extra ...rune) string {
	return strings.TrimRightFunc(s, func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		for _, ru := range extra {
			if ru == r {
				return true
			}
		}
		return false
	})
}

func SplitStrAnyRune(s, runes string) []string {
	st := map[rune]struct{}{}
	for _, c := range runes {
		st[c] = struct{}{}
	}
	tok := strings.FieldsFunc(s, func(r rune) bool {
		_, ok := st[r]
		return ok
	})
	cp := make([]string, 0, len(tok))
	for _, v := range tok {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		cp = append(cp, v)
	}
	return cp
}

// Deprecated: since v0.4.4, use [TrimSpace] instead.
func TrimSpaceAnd(s string, extraRunes string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		for _, ru := range extraRunes {
			if ru == r {
				return true
			}
		}
		return false
	})
}

func NewReplacer(oldnew ...pair.StrPair) *strings.Replacer {
	kw := make([]string, 0, len(oldnew)*2)
	for _, p := range oldnew {
		kw = append(kw, p.Left, p.Right)
	}
	return strings.NewReplacer(kw...)
}

func ReplaceAll(s string, oldnew ...pair.StrPair) string {
	if len(oldnew) < 1 {
		return s
	}
	if len(oldnew) == 1 {
		f := oldnew[0]
		return strings.ReplaceAll(s, f.Left, f.Right)
	}
	return NewReplacer(oldnew...).Replace(s)
}

func CutAfterLast(s string, sep string) string {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return ""
	}
	return s[i+1:]
}
