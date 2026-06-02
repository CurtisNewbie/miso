package miso

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/curtisnewbie/miso/util/json"
	"github.com/curtisnewbie/miso/util/strutil"
)

func GenMarkDownDoc(hr []HttpRouteDoc, pd []PipelineDoc) string {
	b := strings.Builder{}
	b.WriteString("# API Endpoints\n")

	b.WriteString("\n## Contents\n")
	for _, r := range hr {
		tag := fmt.Sprintf("#%s %s", r.Method, r.Url)
		tag = strings.ToLower(tag)
		tag = strings.TrimSpace(tag)
		tag = strings.ReplaceAll(tag, " ", "-")
		tag = regexp.MustCompile(`[/:]`).ReplaceAllString(tag, "")
		b.WriteString(fmt.Sprintf("\n- [%s %s](%s)", r.Method, r.Url, tag))
	}
	b.WriteRune('\n')

	for _, r := range hr {
		b.WriteString(fmt.Sprintf("\n## %s %s\n", r.Method, r.Url))
		if r.Desc != "" {
			b.WriteRune('\n')
			b.WriteString("- Description: ")
			b.WriteString(r.Desc)
		}
		if r.Scope != "" {
			b.WriteRune('\n')
			b.WriteString("- Expected Access Scope: ")
			b.WriteString(r.Scope)
		}
		if r.Resource != "" {
			b.WriteRune('\n')
			b.WriteString("- Bound to Resource: `\"")
			b.WriteString(r.Resource)
			b.WriteString("\"`")
		}
		if len(r.Headers) > 0 {
			b.WriteRune('\n')
			b.WriteString("- Header Parameter:")
			for _, h := range r.Headers {
				b.WriteRune('\n')
				b.WriteString(strutil.Spaces(2))
				b.WriteString("- \"")
				b.WriteString(h.Name)
				b.WriteString("\": ")
				b.WriteString(h.Desc)
			}
		}
		if len(r.QueryParams) > 0 {
			b.WriteRune('\n')
			b.WriteString("- Query Parameter:")
			for _, q := range r.QueryParams {
				b.WriteRune('\n')
				b.WriteString(strutil.Spaces(2))
				b.WriteString("- \"")
				b.WriteString(q.Name)
				b.WriteString("\": ")
				b.WriteString(q.Desc)
			}
		}
		if len(r.JsonRequestDesc.Fields) > 0 {
			b.WriteRune('\n')
			b.WriteString("- JSON Request:")
			if r.JsonRequestDesc.IsSlice {
				b.WriteString(" (array)")
			}
			appendJsonPayloadDoc(&b, r.JsonRequestDesc.Fields, 2)
		}
		if len(r.JsonResponseDesc.Fields) > 0 {
			b.WriteRune('\n')
			b.WriteString("- JSON Response:")
			if r.JsonResponseDesc.IsSlice {
				b.WriteString(" (array)")
			}
			appendJsonPayloadDoc(&b, r.JsonResponseDesc.Fields, 2)
		}

		if r.Curl != "" {
			b.WriteRune('\n')
			b.WriteString("- cURL:\n")
			b.WriteString(strutil.Spaces(2) + "```sh\n")
			b.WriteString(strutil.SAddLineIndent(r.Curl, strutil.Spaces(2)))
			b.WriteString(strutil.Spaces(2) + "```\n")
		}

		if r.MisoTClientDemo != "" && !GetPropBool(PropServerGenerateEndpointDocFileExclTClientDemo) {
			b.WriteRune('\n')
			b.WriteString("- Miso HTTP Client (experimental, demo may not work):\n")
			b.WriteString(strutil.Spaces(2) + "```go\n")
			b.WriteString(strutil.SAddLineIndent(r.MisoTClientDemo+"\n", strutil.Spaces(2)))
			b.WriteString(strutil.Spaces(2) + "```\n")
		}

		if r.JsonTsDef != "" {
			b.WriteRune('\n')
			b.WriteString("- JSON Request / Response Object In TypeScript:\n")
			b.WriteString(strutil.Spaces(2) + "```ts\n")
			b.WriteString(strutil.SAddLineIndent(r.JsonTsDef, strutil.Spaces(2)))
			b.WriteString(strutil.Spaces(2) + "```\n")
		}

		if !GetPropBool(PropServerGenerateEndpointDocFileExclNgClientDemo) {
			if r.NgHttpClientDemo != "" {
				b.WriteRune('\n')
				b.WriteString("- Angular HttpClient Demo:\n")
				b.WriteString(strutil.Spaces(2) + "```ts\n")
				b.WriteString(strutil.SAddLineIndent(r.NgHttpClientDemo, strutil.Spaces(2)))
				b.WriteString(strutil.Spaces(2) + "```\n")
			}

			if r.NgTableDemo != "" {
				b.WriteRune('\n')
				b.WriteString("- Angular NgTable Demo:\n")
				b.WriteString(strutil.Spaces(2) + "```html\n")
				b.WriteString(strutil.SAddLineIndent(r.NgTableDemo+"\n", strutil.Spaces(2)))
				b.WriteString(strutil.Spaces(2) + "```\n")
			}
		}

		if r.OpenApiDoc != "" && !GetPropBool(PropServerGenerateEndpointDocFileExclOpenApi) {
			b.WriteRune('\n')
			b.WriteString("- Open Api (experimental, demo may not work):\n")
			b.WriteString(strutil.Spaces(2) + "```json\n")
			b.WriteString(strutil.SAddLineIndent(r.OpenApiDoc+"\n", strutil.Spaces(2)))
			b.WriteString(strutil.Spaces(2) + "```\n")
		}
	}

	if len(pd) > 0 {

		b.WriteString("\n# Event Pipelines\n")
		sort.Slice(pd, func(i, j int) bool { return pd[i].Queue < pd[j].Queue })

		for _, p := range pd {
			b.WriteString("\n- ")
			b.WriteString(p.Name)

			if p.Desc != "" {
				b.WriteRune('\n')
				b.WriteString(strutil.Spaces(2))
				b.WriteString("- Description: ")
				b.WriteString(p.Desc)
			}

			if p.Queue != "" {
				b.WriteRune('\n')
				b.WriteString(strutil.Spaces(2))
				b.WriteString("- RabbitMQ Queue: `")
				b.WriteString(p.Queue)
				b.WriteString("`")
			}

			if p.Exchange != "" {
				b.WriteRune('\n')
				b.WriteString(strutil.Spaces(2))
				b.WriteString("- RabbitMQ Exchange: `")
				b.WriteString(p.Exchange)
				b.WriteString("`")
			}

			if p.RoutingKey != "" {
				b.WriteRune('\n')
				b.WriteString(strutil.Spaces(2))
				b.WriteString("- RabbitMQ RoutingKey: `")
				b.WriteString(p.RoutingKey)
				b.WriteString("`")
			}

			if len(p.PayloadDesc.Fields) > 0 {
				b.WriteRune('\n')
				b.WriteString(strutil.Spaces(2))
				b.WriteString("- Event Payload:")
				if p.PayloadDesc.IsSlice {
					b.WriteString(" (array)")
				}
				appendJsonPayloadDoc(&b, p.PayloadDesc.Fields, 2)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func appendJsonPayloadDoc(b *strings.Builder, jds []FieldDesc, indent int) {
	for _, jd := range jds {
		b.WriteString(fmt.Sprintf("\n%s- \"%s\": (%s) %s", strutil.Spaces(indent+2), jd.JsonName, jd.TypeNameAlias, jd.comment(false)))

		if len(jd.Fields) > 0 {
			appendJsonPayloadDoc(b, jd.Fields, indent+2)
		}
	}
}

func GenRouteCurl(d HttpRouteDoc, port string) string {
	sl := new(strutil.SLPinter)
	sl.LineSuffix = " \\"
	var qp string
	for i, q := range d.QueryParams {
		if qp == "" {
			qp = "?"
		}
		qp += fmt.Sprintf("%s=", q.Name)
		if i < len(d.QueryParams)-1 {
			qp += "&"
		}
	}
	sl.Printlnf("curl -X %s 'http://localhost:%s%s%s'", d.Method, port, d.Url, qp)
	sl.LinePrefix = "  "

	for _, h := range d.Headers {
		sl.Printlnf("-H '%s: '", h.Name)
	}

	if len(d.JsonRequestDesc.Fields) > 0 {
		sl.Printlnf("-H 'Content-Type: application/json'")

		jm := map[string]any{}
		genJsonReqMap(jm, d.JsonRequestDesc.Fields)
		sj, err := json.CustomSWriteJson(apiDocJsoniterConfig, jm)
		if err == nil {
			if d.JsonRequestDesc.IsSlice {
				sl.Printlnf("-d '[ %s ]'", sj)
			} else {
				sl.Printlnf("-d '%s'", sj)
			}
		}
	}
	sl.WriteString("\n")
	return sl.String()
}

func genJsonReqMap(jm map[string]any, descs []FieldDesc) {
	for _, d := range descs {
		if d.IsSliceOrArray {
			jm[d.JsonName] = make([]any, 0)
		} else {
			if len(d.Fields) > 0 {
				t := map[string]any{}
				genJsonReqMap(t, d.Fields)
				jm[d.JsonName] = t
			} else {
				var v any
				switch d.TypeNameAlias {
				case "string", "*string":
					v = ""
				case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
					"*int", "*int8", "*int16", "*int32", "*int64", "*uint", "*uint8", "*uint16", "*uint32", "*uint64":
					if strutil.EqualAnyStr(d.OriginTypeName, "Time", "*Time") ||
						strings.HasSuffix(d.OriginTypeName, ".Time") {
						v = 1768184753983 // epochmilli
					} else {
						v = 0
					}
				case "float32", "float64", "*float32", "*float64":
					v = 0.0
				case "bool", "*bool":
					v = false
				case "[]bool":
					v = []bool{}
				case "[]string":
					v = []string{}
				case "[]int", "[]int8", "[]int16", "[]int32", "[]int64":
					v = []int{}
				case "[]float32", "[]float64":
					v = []float32{}
				}
				jm[d.JsonName] = v
			}
		}
	}
}
