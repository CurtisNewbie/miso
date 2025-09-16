package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ChimeraCoder/gojson"
	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/util/cli"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/version"
	"golang.design/x/clipboard"
)

var (
	Debug bool
)

func main() {
	flag.Usage = func() {
		cli.Printlnf("\nmisocurl - automatically miso.TClient code based on curl in clipboard\n")
		cli.Printlnf("  Supported miso version: %v\n", version.Version)
		cli.Printlnf("Usage of %s:", os.Args[0])
		flag.PrintDefaults()
	}
	flag.BoolVar(&Debug, "debug", false, "Debug")
	flag.Parse()

	var curl string
	err := clipboard.Init()
	cli.DebugPrintlnf(Debug, "clipboard init")
	if err == nil {
		txt := clipboard.Read(clipboard.FmtText)
		if txt != nil {
			s := util.UnsafeByt2Str(txt)
			if strings.Contains(strings.ToLower(s), "curl") {
				curl = s
			}
		}
	}

	if curl == "" {
		cli.Printlnf("Missing curl command, please copy the curl command to clipboard.")
		return
	}

	inst, ok := ParseCurl(curl)
	if !ok {
		cli.Printlnf("Failed to parse curl command")
		return
	}

	cli.DebugPrintlnf(Debug, "%#v", inst)

	py := GenRequests(inst)
	print(py)
	println()
}

func GenRequests(inst Instruction) string {
	headers := ""
	var call string
	var callType string
	if len(inst.Form) > 0 {
		sb := strings.Builder{}
		sb.WriteString("\n")
		sb.WriteString("\t\tPostForm(map[string][]string{")
		for k, v := range inst.Form {
			sb.WriteString(fmt.Sprintf("\n\t\t\t\"%v\": []string{%v},", k, v))
		}
		sb.WriteString("\n\t\t}).")
		call = sb.String()
	} else if !util.IsBlankStr(inst.Payload) {

		inst.Payload = unquote(inst.Payload)
		out, err := gojson.Generate(strings.NewReader(inst.Payload), gojson.ParseJson, "Req", "main", []string{"fmt"}, false, true)
		if err == nil {
			callType = string(out)
			callType = strings.ReplaceAll(callType, "package main", "")
			callType = strings.TrimSpace(callType)
			callType = "\n" + util.SAddLineIndent(callType, "\t") + "\n"

			sb := strings.Builder{}
			sb.WriteString("\n")
			if inst.Method == "POST" {
				sb.WriteString("\t\tPostJson(Req{}).")
			} else {
				sb.WriteString("\t\tPutJson(Req{}).")
			}
			call = sb.String()
		}
	} else if inst.Method == "GET" {
		call = "\n\t\tGet()."
	} else if inst.Method == "POST" {
		call = "\n\t\tPost(nil)."
	} else if inst.Method == "PUT" {
		call = "\n\t\tPut(nil)."
	} else if inst.Method == "DELETE" {
		call = "\n\t\tDelete()."
	}

	headersb := strings.Builder{}
	for k, v := range inst.Headers {
		headersb.WriteString(fmt.Sprintf("\n\t\tAddHeader(\"%v\", \"%v\").", k, util.EscapeString(v)))
	}
	headers = headersb.String()

	return util.NamedSprintf(`${callType}
	rail := miso.EmptyRail()
	s, err := miso.NewClient(rail, "${url}").${headers}
		Require2xx().${call}
		Str()
	if err != nil {
		panic(err)
	}
	rail.Infof("Response: %v", s)
	`,
		map[string]any{
			"callType": callType,
			"method":   strings.ToLower(inst.Method),
			"url":      inst.Url,
			"headers":  headers,
			"call":     call,
		})
}

type Instruction struct {
	Url     string
	Method  string
	Headers map[string]string
	Payload string
	Form    map[string]string
}

func ParseCurl(curl string) (inst Instruction, ok bool) {
	if util.IsBlankStr(curl) {
		return
	}
	inst.Headers = map[string]string{}
	inst.Form = map[string]string{}
	if util.IsBlankStr(inst.Method) {
		inst.Method = "GET"
	}

	p := NewCurlParser(curl)
	for p.HasNext() {
		tok := p.Next()
		cli.DebugPrintlnf(Debug, "next tok: %v", tok)
		switch tok {
		case "-H":
			k, v, ok := util.SplitKV(unquote(p.Next()), ":")
			if ok {
				inst.Headers[k] = v
			}
		case "-X":
			inst.Method = unquote(p.Next())
		case "-b": // don't need it yet
			p.Next()
		case "-F":
			k, v, ok := util.SplitKV(unquote(p.Next()), "=")
			if ok {
				inst.Form[k] = v
			}
		case "-d", "--data-raw":
			inst.Payload = p.Next()
		case "curl":
		default:
			cli.DebugPrintlnf(Debug, "default tok: %v", tok)
			if tok != "" {
				inst.Url = unquote(tok)
			}
		}
	}
	if inst.Method == "GET" && inst.Payload != "" {
		inst.Method = "POST"
	}

	for k, v := range inst.Headers {
		if strings.ToLower(k) == "authorization" {
			if strings.HasPrefix(strings.TrimSpace(v), "Bearer") {
				inst.Headers[k] = "Bearer {token}"
			}
		}
	}

	cli.DebugPrintlnf(Debug, "inst: %+v", inst)
	ok = true
	return
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	v := []rune(s)
	if len(v) >= 2 && (v[0] == '\'' || v[0] == '"') {
		return string(v[1 : len(v)-1])
	}
	return strings.TrimSpace(string(v))
}

func NewCurlParser(curl string) *CurlParser {
	rc := []rune(curl)
	return &CurlParser{curl: curl, rcurl: rc, pos: 0, rlen: len(rc)}
}

type CurlParser struct {
	curl  string
	rcurl []rune
	rlen  int
	pos   int
}

func (c *CurlParser) HasNext() bool {
	return c.pos < len(c.rcurl)
}

func (c *CurlParser) inRange(n int) bool {
	return c.pos+n < len(c.rcurl)
}

func (c *CurlParser) peek(n int) rune {
	return c.rcurl[c.pos+n]
}

func (c *CurlParser) move(n int) {
	c.pos += n
}

func (c *CurlParser) parseCmdKey() string {
	i := 0
	for c.inRange(i) {
		switch c.peek(i) {
		case ' ', '\t', '\n':
			s := c.rcurl[c.pos : c.pos+i]
			c.move(i)
			return string(s)
		}
		i++
	}
	return ""
}

func (c *CurlParser) parseStr() string {
	stack := slutil.NewStack[rune](10)
	cur := c.peek(0)
	stack.Push(cur)
	i := 1
	escape := false
	for c.inRange(i) && !stack.Empty() {
		p := c.peek(i)
		switch p {
		case '\'', '"':
			if escape {
				escape = false
			} else {
				if p == cur {
					stack.Pop()
					cur, _ = stack.Peek()
				} else {
					stack.Push(p)
					cur = p
				}
			}
		case '\\':
			escape = !escape
		}
		i++
	}
	s := c.rcurl[c.pos : c.pos+i]
	c.move(i)
	vs := string(s)
	cli.DebugPrintlnf(Debug, "parseStr, s: %v", vs)
	return vs
}

func (c *CurlParser) isSpace(n int) bool {
	return c.peek(n) == ' ' || c.peek(n) == '\n' || c.peek(n) == '\t' || c.peek(n) == '\\'
}

func (c *CurlParser) skipSpaces() {
	for c.HasNext() && c.isSpace(0) {
		c.move(1)
	}
}

func (c *CurlParser) parseWords() string {
	c.skipSpaces()
	i := 0
	for c.inRange(i) && !c.isSpace(i) {
		i++
	}
	s := c.rcurl[c.pos : c.pos+i]
	c.move(i)
	return string(s)
}

func (c *CurlParser) Next() (tok string) {
	c.skipSpaces()

	if !c.HasNext() {
		return tok
	}

	curr := c.peek(0)
	switch curr {
	case '-':
		return c.parseCmdKey()
	case '\'', '"':
		return c.parseStr()
	case '$':
		c.move(1)
		return c.parseStr()
	default:
		return c.parseWords()
	}
}
