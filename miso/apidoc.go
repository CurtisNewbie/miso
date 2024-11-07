package miso

import (
	_ "embed"
	"fmt"
	"html/template"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/encoding/json"
	"github.com/curtisnewbie/miso/util"
	jsoniter "github.com/json-iterator/go"
)

const (
	TagApiDocDesc  = "desc"
	TagApiDocXDesc = "xdesc"
)

var (
	ApiDocTypeAlias = map[string]string{
		"ETime":       "int64",
		"*ETime":      "int64",
		"*miso.ETime": "int64",
		"*util.ETime": "int64",
	}
	apiDocEndpointDisabled = false

	ignoredJsonDocTag = []string{"form", "header"}

	// callback funcs to retrieve pipeline definitions
	getPipelineDocFuncs []GetPipelineDocFunc

	// extra descriptions
	xdescs map[string]string

	apiDocJsoniterConfig = jsoniter.Config{
		EscapeHTML:  true,
		SortMapKeys: true,
	}.Froze()
)

var (
	apiDocTmpl          *template.Template
	buildApiDocTmplOnce sync.Once
)

func init() {
	SetDefProp(PropServerGenerateEndpointDocEnabled, true)

	PostServerBootstrap(func(rail Rail) error {
		if !GetPropBool(PropServerGenerateEndpointDocEnabled) || apiDocEndpointDisabled {
			return nil
		}
		outf := GetPropStr(PropServerGenerateEndpointDocFile)
		if outf == "" {
			return nil
		}
		f, err := util.ReadWriteFile(outf)
		if err != nil {
			rail.Debugf("Failed to open API doc file, %v, %v", outf, err)
			return nil
		}
		f.Truncate(0)
		defer f.Close()

		httpRouteDoc := buildHttpRouteDoc(GetHttpRoutes())
		var pipelineDoc []PipelineDoc
		for _, f := range getPipelineDocFuncs {
			pipelineDoc = append(pipelineDoc, f()...)
		}
		markdown := genMarkDownDoc(httpRouteDoc, pipelineDoc)
		_, _ = f.WriteString(markdown)
		return nil
	})
}

type GetPipelineDocFunc func() []PipelineDoc

type httpRouteDoc struct {
	Url              string           // http request url
	Method           string           // http method
	Extra            map[string][]any // extra metadata
	Desc             string           // description of the route (metadata).
	Scope            string           // the documented access scope of the route, it maybe "PUBLIC" or something else (metadata).
	Resource         string           // the documented resource that the route should be bound to (metadata).
	Headers          []ParamDoc       // the documented header parameters that will be used by the endpoint (metadata).
	QueryParams      []ParamDoc       // the documented query parameters that will used by the endpoint (metadata).
	JsonRequestDesc  JsonPayloadDesc  // the documented json request type that is expected by the endpoint (metadata).
	JsonResponseDesc JsonPayloadDesc  // the documented json response type that will be returned by the endpoint (metadata).
	Curl             string           // curl demo
	JsonReqTsDef     string           // json request type def in ts
	JsonRespTsDef    string           // json response type def in ts
	NgHttpClientDemo string           // angular http client demo
	NgTableDemo      string           // angular table demo
	MisoTClientDemo  string           // miso TClient demo
}

type FieldDesc struct {
	FieldName      string
	Name           string
	TypeName       string
	OriginTypeName string
	Desc           string
	Fields         []FieldDesc
}

type JsonPayloadDesc struct {
	IsSlice bool
	Fields  []FieldDesc
}

func buildHttpRouteDoc(hr []HttpRoute) []httpRouteDoc {
	docs := make([]httpRouteDoc, 0, len(hr))

	for _, r := range hr {
		d := httpRouteDoc{
			Url:         r.Url,
			Method:      r.Method,
			Extra:       r.Extra,
			Desc:        r.Desc,
			Scope:       r.Scope,
			Resource:    r.Resource,
			Headers:     r.Headers,
			QueryParams: r.QueryParams,
		}

		var jsonRequestVal any
		if l, ok := r.Extra[ExtraJsonRequest]; ok && len(l) > 0 {
			jsonRequestVal = l[0]
		}

		var jsonResponseVal any
		if l, ok := r.Extra[ExtraJsonResponse]; ok && len(l) > 0 {
			jsonResponseVal = l[0]
		}

		// json stuff
		var reqTypeName string
		if jsonRequestVal != nil {
			rv := reflect.ValueOf(jsonRequestVal)
			d.JsonRequestDesc, reqTypeName = BuildJsonPayloadDesc(rv)
			d.JsonReqTsDef = genJsonTsDef(reqTypeName, d.JsonRequestDesc)
		}

		var respTypeName string
		if jsonResponseVal != nil {
			rv := reflect.ValueOf(jsonResponseVal)
			d.JsonResponseDesc, respTypeName = BuildJsonPayloadDesc(rv)
			d.JsonRespTsDef = genJsonTsDef(respTypeName, d.JsonResponseDesc)
		}

		// curl
		d.Curl = genRouteCurl(d)

		// ng http client
		d.NgHttpClientDemo = genNgHttpClientDemo(d, reqTypeName, respTypeName)

		// ng table demo
		if _, ok := r.Extra[ExtraNgTable]; ok {
			d.NgTableDemo = genNgTableDemo(d, respTypeName)
		}

		// miso http TClient
		d.MisoTClientDemo = genTClientDemo(d, reqTypeName, respTypeName)

		docs = append(docs, d)
	}
	return docs
}

func genMarkDownDoc(hr []httpRouteDoc, pd []PipelineDoc) string {
	b := strings.Builder{}
	b.WriteString("# API Endpoints\n")

	for _, r := range hr {
		b.WriteString("\n- ")
		b.WriteString(r.Method)
		b.WriteString(" ")
		b.WriteString(r.Url)
		if r.Desc != "" {
			b.WriteRune('\n')
			b.WriteString(util.Spaces(2))
			b.WriteString("- Description: ")
			b.WriteString(r.Desc)
		}
		if r.Scope != "" {
			b.WriteRune('\n')
			b.WriteString(util.Spaces(2))
			b.WriteString("- Expected Access Scope: ")
			b.WriteString(r.Scope)
		}
		if r.Resource != "" {
			b.WriteRune('\n')
			b.WriteString(util.Spaces(2))
			b.WriteString("- Bound to Resource: `\"")
			b.WriteString(r.Resource)
			b.WriteString("\"`")
		}
		if len(r.Headers) > 0 {
			b.WriteRune('\n')
			b.WriteString(util.Spaces(2))
			b.WriteString("- Header Parameter:")
			for _, h := range r.Headers {
				b.WriteRune('\n')
				b.WriteString(util.Spaces(4))
				b.WriteString("- \"")
				b.WriteString(h.Name)
				b.WriteString("\": ")
				b.WriteString(h.Desc)
			}
		}
		if len(r.QueryParams) > 0 {
			b.WriteRune('\n')
			b.WriteString(util.Spaces(2))
			b.WriteString("- Query Parameter:")
			for _, q := range r.QueryParams {
				b.WriteRune('\n')
				b.WriteString(util.Spaces(4))
				b.WriteString("- \"")
				b.WriteString(q.Name)
				b.WriteString("\": ")
				b.WriteString(q.Desc)
			}
		}
		if len(r.JsonRequestDesc.Fields) > 0 {
			b.WriteRune('\n')
			b.WriteString(util.Spaces(2))
			b.WriteString("- JSON Request:")
			if r.JsonRequestDesc.IsSlice {
				b.WriteString(" (array)")
			}
			appendJsonPayloadDoc(&b, r.JsonRequestDesc.Fields, 2)
		}
		if len(r.JsonResponseDesc.Fields) > 0 {
			b.WriteRune('\n')
			b.WriteString(util.Spaces(2))
			b.WriteString("- JSON Response:")
			if r.JsonResponseDesc.IsSlice {
				b.WriteString(" (array)")
			}
			appendJsonPayloadDoc(&b, r.JsonResponseDesc.Fields, 2)
		}

		if r.Curl != "" {
			b.WriteRune('\n')
			b.WriteString(util.Spaces(2))
			b.WriteString("- cURL:\n")
			b.WriteString(util.Spaces(4) + "```sh\n")
			b.WriteString(util.SAddLineIndent(r.Curl, util.Spaces(4)))
			b.WriteString(util.Spaces(4) + "```\n")
		}

		// if r.MisoTClientDemo != "" {
		// 	b.WriteRune('\n')
		// 	b.WriteString(util.Spaces(2))
		// 	b.WriteString("- Miso HTTP Client:\n")
		// 	b.WriteString(util.Spaces(4) + "```go\n")
		// 	b.WriteString(util.SAddLineIndent(r.MisoTClientDemo, util.Spaces(4)))
		// 	b.WriteString(util.Spaces(4) + "```\n")
		// }

		if r.JsonReqTsDef != "" {
			b.WriteRune('\n')
			b.WriteString(util.Spaces(2))
			b.WriteString("- JSON Request Object In TypeScript:\n")
			b.WriteString(util.Spaces(4) + "```ts\n")
			b.WriteString(util.SAddLineIndent(r.JsonReqTsDef, util.Spaces(4)))
			b.WriteString(util.Spaces(4) + "```\n")
		}

		if r.JsonRespTsDef != "" {
			b.WriteRune('\n')
			b.WriteString(util.Spaces(2))
			b.WriteString("- JSON Response Object In TypeScript:\n")
			b.WriteString(util.Spaces(4) + "```ts\n")
			b.WriteString(util.SAddLineIndent(r.JsonRespTsDef, util.Spaces(4)))
			b.WriteString(util.Spaces(4) + "```\n")
		}

		if r.NgHttpClientDemo != "" {
			b.WriteRune('\n')
			b.WriteString(util.Spaces(2))
			b.WriteString("- Angular HttpClient Demo:\n")
			b.WriteString(util.Spaces(4) + "```ts\n")
			b.WriteString(util.SAddLineIndent(r.NgHttpClientDemo, util.Spaces(4)))
			b.WriteString(util.Spaces(4) + "```\n")
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
				b.WriteString(util.Spaces(2))
				b.WriteString("- Description: ")
				b.WriteString(p.Desc)
			}

			if p.Queue != "" {
				b.WriteRune('\n')
				b.WriteString(util.Spaces(2))
				b.WriteString("- RabbitMQ Queue: `")
				b.WriteString(p.Queue)
				b.WriteString("`")
			}

			if p.Exchange != "" {
				b.WriteRune('\n')
				b.WriteString(util.Spaces(2))
				b.WriteString("- RabbitMQ Exchange: `")
				b.WriteString(p.Exchange)
				b.WriteString("`")
			}

			if p.RoutingKey != "" {
				b.WriteRune('\n')
				b.WriteString(util.Spaces(2))
				b.WriteString("- RabbitMQ RoutingKey: `")
				b.WriteString(p.RoutingKey)
				b.WriteString("`")
			}

			if len(p.PayloadDesc.Fields) > 0 {
				b.WriteRune('\n')
				b.WriteString(util.Spaces(2))
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
		b.WriteString(fmt.Sprintf("\n%s- \"%s\": (%s) %s", util.Spaces(indent+2), jd.Name, jd.TypeName, jd.Desc))

		if len(jd.Fields) > 0 {
			appendJsonPayloadDoc(b, jd.Fields, indent+2)
		}
	}
}

// Parse value's type information to build json style description.
//
// Only supports struct, pointer and slice.
func BuildJsonPayloadDesc(v reflect.Value) (jpd JsonPayloadDesc, typeName string) {
	typeName = v.Type().Name()
	switch v.Kind() {
	case reflect.Pointer:
		v = reflect.New(v.Type().Elem()).Elem()
		jpd, typeName = BuildJsonPayloadDesc(v)
		return
	case reflect.Struct:
		jpd = JsonPayloadDesc{Fields: buildJsonDesc(v)}
		return
	case reflect.Slice:
		et := v.Type().Elem()
		if et.Kind() == reflect.Struct {
			typeName = et.Name()
			ev := reflect.New(et).Elem()
			jpd = JsonPayloadDesc{IsSlice: true, Fields: buildJsonDesc(ev)}
			return
		}
	}
	return
}

func buildJsonDesc(v reflect.Value) []FieldDesc {
	t := v.Type()
	if t.Kind() != reflect.Struct {
		return []FieldDesc{}
	}
	jds := make([]FieldDesc, 0, 5)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if util.IsVoid(f.Type) {
			continue
		}

		skipped := false
		for _, it := range ignoredJsonDocTag {
			if v := f.Tag.Get(it); v != "" {
				skipped = true
				break
			}
		}
		if skipped {
			continue
		}

		var name string
		if v := f.Tag.Get("json"); v != "" {
			if v == "-" {
				continue
			}
			name = v
		} else {
			name = json.LowercaseNamingStrategy(f.Name)
		}

		originTypeName := util.TypeName(f.Type)
		typeAlias, typeAliasMatched := ApiDocTypeAlias[originTypeName]
		var typeName string
		if typeAliasMatched {
			typeName = typeAlias
		} else {
			typeName = originTypeName
		}

		jd := FieldDesc{
			FieldName:      f.Name,
			Name:           name,
			TypeName:       typeName,
			OriginTypeName: originTypeName,
			Desc:           getTagDesc(f.Tag),
			Fields:         []FieldDesc{},
		}

		if typeAliasMatched {
			jds = append(jds, jd)
			continue
		}

		fv := v.Field(i)
		appendable := true
		if f.Type.Kind() == reflect.Interface {
			if !fv.IsZero() && !fv.IsNil() {
				if ele := fv.Elem(); ele.IsValid() {
					et := ele.Type()
					jd.TypeName = util.TypeName(et)
					jd.Fields = reflectAppendJsonDesc(et, ele, jd.Fields)
				}
			} else {
				appendable = false // e.g., the any field in GnResp[any]{}
				Debugf("reflect.Value is zero or nil, not displayed in api doc, type: %v, field: %v", t.Name(), jd.Name)
			}
		} else {
			jd.Fields = reflectAppendJsonDesc(f.Type, fv, jd.Fields)
		}

		if appendable {
			jds = append(jds, jd)
		}
	}
	return jds
}

func reflectAppendJsonDesc(t reflect.Type, v reflect.Value, fields []FieldDesc) []FieldDesc {
	if t.Kind() == reflect.Struct {
		fields = append(fields, buildJsonDesc(v)...)
	} else if t.Kind() == reflect.Slice {
		et := t.Elem()
		if et.Kind() == reflect.Struct {
			ev := reflect.New(et).Elem()
			fields = append(fields, buildJsonDesc(ev)...)
		}
	} else if t.Kind() == reflect.Pointer {
		ev := reflect.New(t.Elem()).Elem()
		if ev.Kind() == reflect.Struct {
			fields = append(fields, buildJsonDesc(ev)...)
		}
	} else if t.Kind() == reflect.Interface {
		if !v.IsZero() && !v.IsNil() {
			if ele := v.Elem(); ele.IsValid() {
				et := ele.Type()
				fields = reflectAppendJsonDesc(et, ele, fields)
			}
		}
	}
	return fields
}

var (
	//go:embed apidoc_template.html
	apidocTemplate string
)

func serveApiDocTmpl(rail Rail) error {

	if !GetPropBool(PropServerGenerateEndpointDocEnabled) || apiDocEndpointDisabled {
		return nil
	}

	var err error
	buildApiDocTmplOnce.Do(func() {
		t, er := template.New("").Parse(apidocTemplate)
		if er != nil {
			err = er
			return
		}
		apiDocTmpl = t
	})
	if err != nil {
		return err
	}

	RawGet("/doc/api",
		func(inb *Inbound) {
			defer DebugTimeOp(rail, time.Now(), "gen api doc")

			docs := buildHttpRouteDoc(GetHttpRoutes())
			var pipelineDoc []PipelineDoc
			for _, f := range getPipelineDocFuncs {
				pipelineDoc = append(pipelineDoc, f()...)
			}
			markdown := genMarkDownDoc(docs, pipelineDoc)

			w, _ := inb.Unwrap()
			if err := apiDocTmpl.ExecuteTemplate(w, "apiDocTempl",
				struct {
					App         string
					HttpDoc     []httpRouteDoc
					PipelineDoc []PipelineDoc
					Markdown    string
				}{
					App:         GetPropStr(PropAppName),
					HttpDoc:     docs,
					PipelineDoc: pipelineDoc,
					Markdown:    markdown,
				}); err != nil {
				rail.Errorf("failed to serve apiDocTmpl, %v", err)
			}
		}).
		Desc("Serve the generated API documentation webpage").
		Public()

	PostServerBootstrap(func(rail Rail) error {
		rail.Infof("Exposing API Documentation on http://localhost:%v/doc/api", GetPropInt(PropServerActualPort))
		return nil
	})

	return nil
}

func parseQueryDoc(t reflect.Type) []ParamDoc {
	if t == nil {
		return nil
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	pds := []ParamDoc{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if util.IsVoid(f.Type) {
			continue
		}

		query := f.Tag.Get(TagQueryParam)
		if query == "" {
			continue
		}
		desc := getTagDesc(f.Tag)
		pds = append(pds, ParamDoc{
			Name: query,
			Desc: desc,
		})
	}
	return pds
}

func parseHeaderDoc(t reflect.Type) []ParamDoc {
	if t == nil {
		return nil
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	pds := []ParamDoc{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if util.IsVoid(f.Type) {
			continue
		}

		header := f.Tag.Get(TagHeaderParam)
		if header == "" {
			continue
		}
		header = strings.ToLower(header)
		desc := getTagDesc(f.Tag)
		pds = append(pds, ParamDoc{
			Name: header,
			Desc: desc,
		})
	}
	return pds
}

func genRouteCurl(d httpRouteDoc) string {
	sl := new(util.SLPinter)
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
	sl.Printlnf("curl -X %s 'http://localhost:%s%s%s'", d.Method, GetPropStr(PropServerPort), d.Url, qp)
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
		if len(d.Fields) > 0 {
			t := map[string]any{}
			genJsonReqMap(t, d.Fields)
			jm[d.Name] = t
		} else {
			var v any
			switch d.TypeName {
			case "string", "*string":
				v = ""
			case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
				"*int", "*int8", "*int16", "*int32", "*int64", "*uint", "*uint8", "*uint16", "*uint32", "*uint64":
				v = 0
			case "float32", "float64", "*float32", "*float64":
				v = 0.0
			case "bool", "*bool":
				v = false
			default:
				if strings.HasPrefix(d.TypeName, "[]") {
					v = make([]any, 0)
				}
			}
			jm[d.Name] = v
		}
	}
}

// generate one or more typescript interface definitions based on a set of jsonDesc.
func genJsonTsDef(typeName string, payload JsonPayloadDesc) string {
	if len(payload.Fields) < 1 {
		return ""
	}
	sb, writef := util.NewIndWritef("  ")
	writef(0, "export interface %s {", guessTsItfName(typeName))
	deferred := make([]func(), 0, 10)
	genJsonTsDefRecur(1, writef, &deferred, payload.Fields)
	writef(0, "}")

	for i := 0; i < len(deferred); i++ {
		writef(0, "")
		deferred[i]()
	}
	return sb.String()
}

func genJsonTsDefRecur(indentc int, writef util.IndWritef, deferred *[]func(), descs []FieldDesc) {
	for i := range descs {
		d := descs[i]

		if len(d.Fields) > 0 {
			tsTypeName := guessTsItfName(d.TypeName)
			n := tsTypeName
			if strings.HasPrefix(d.TypeName, "[]") {
				n += "[]"
			}
			writef(indentc, "%s?: %s", d.Name, n)

			*deferred = append(*deferred, func() {
				writef(0, "export interface %s {", tsTypeName)
				genJsonTsDefRecur(1, writef, deferred, d.Fields)
				writef(0, "}")
			})

		} else {
			var tname string = guessTsPrimiTypeName(d.TypeName)
			var comment string
			if d.Desc != "" {
				comment = " // " + d.Desc
				fieldDec := fmt.Sprintf("%s?: %s", d.Name, tname)
				writef(indentc, "%-30s%s", fieldDec, comment)
			} else {
				writef(indentc, "%s?: %s", d.Name, tname)
			}
		}
	}
}

// try to convert golang type name to typescript primitive type name.
func guessTsPrimiTypeName(typeName string) string {
	var tname string
	switch typeName {
	case "string", "*string":
		tname = "string"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
		"*int", "*int8", "*int16", "*int32", "*int64", "*uint", "*uint8", "*uint16", "*uint32", "*uint64":
		tname = "number"
	case "float32", "float64", "*float32", "*float64":
		tname = "number"
	case "bool", "*bool":
		tname = "boolean"
	default:
		if v, ok := strings.CutPrefix(typeName, "[]"); ok {
			tname = guessTsPrimiTypeName(v) + "[]"
		} else {
			tname = typeName
		}
	}
	return tname
}

// try to convert golang type (incl struct name) name to typescript interface name.
func guessTsItfName(n string) string {
	if len(n) == 0 {
		return n
	}

	// cp := n
	v, ok := strings.CutPrefix(n, "[]")
	if ok {
		if len(n) == 2 {
			return n
		}
		n = v
	}

	if n[len(n)-1] == ']' {
		j := strings.IndexByte(n, '[')
		if j > -1 {
			n = n[:j]
		}
	}

	i := strings.LastIndexByte(n, '.')
	if i > -1 {
		n = n[i+1:]
	}
	// Debugf("guessing typescript interface name: %v -> %v", cp, n)
	return n
}

func guessGoTypName(n string) string {
	tsTypeName := guessTsItfName(n)
	if strings.HasPrefix(n, "[]") {
		return "[]" + tsTypeName
	}
	return tsTypeName
}

func genNgTableDemo(d httpRouteDoc, respTypeName string) string {
	sl := new(util.SLPinter)
	sl.Println(`<table mat-table [dataSource]="tabdata" class="mb-4" style="width: 100%;">`)

	var cols []string

	if respTypeName != "" {
		respTypeName = guessTsItfName(respTypeName)

		// Resp.Data -> PageRes -> PageRes.Payload
		if respTypeName == "Resp" {
			pl, hasData := util.SliceFilterFirst(d.JsonResponseDesc.Fields, func(j FieldDesc) bool {
				return j.FieldName == "Data"
			})
			if hasData {
				pl, hasPayload := util.SliceFilterFirst(pl.Fields, func(j FieldDesc) bool {
					return j.FieldName == "Payload"
				})
				if hasPayload {
					for _, f := range pl.Fields {
						sl.Printlnf(util.Tabs(1)+"<ng-container matColumnDef=\"%v\">", f.Name)
						sl.Printlnf(util.Tabs(2)+"<th mat-header-cell *matHeaderCellDef> %s </th>", f.FieldName)
						if f.OriginTypeName == "ETime" || f.OriginTypeName == "*ETime" || f.OriginTypeName == "util.ETime" || f.OriginTypeName == "*util.ETime" {
							sl.Printlnf(util.Tabs(2)+"<td mat-cell *matCellDef=\"let u\"> {{u.%s | date: 'yyyy-MM-dd HH:mm:ss'}} </td>", f.Name)
						} else {
							sl.Printlnf(util.Tabs(2)+"<td mat-cell *matCellDef=\"let u\"> {{u.%s}} </td>", f.Name)
						}
						sl.Printlnf(util.Tabs(1) + "</ng-container>")
						cols = append(cols, "'"+f.Name+"'")
					}
				}
			}
		}
	}

	colstr := "[" + strings.Join(cols, ",") + "]"
	sl.Printlnf(util.Tabs(1)+"<tr mat-header-row *matHeaderRowDef=\"%s\"></tr>", colstr)
	sl.Printlnf(`</table>`)
	return sl.String()
}

func genNgHttpClientDemo(d httpRouteDoc, reqTypeName string, respTypeName string) string {
	sl := new(util.SLPinter)
	sl.Printlnf("import { MatSnackBar } from \"@angular/material/snack-bar\";")
	sl.Printlnf("import { HttpClient } from \"@angular/common/http\";")
	sl.Printlnf("")
	sl.Printlnf("constructor(")
	sl.Printlnf(util.Spaces(2) + "private snackBar: MatSnackBar,")
	sl.Printlnf(util.Spaces(2) + "private http: HttpClient")
	sl.Printlnf(") {}")
	sl.Printlnf("")

	var qp string
	for i, q := range d.QueryParams {
		cname := util.CamelCase(q.Name)
		sl.Printlnf("let %s: any | null = null;", cname)

		if qp == "" {
			qp = "?"
		}
		qp += fmt.Sprintf("%s=${%s}", q.Name, cname)
		if i < len(d.QueryParams)-1 {
			qp += "&"
		}
	}

	app := GetPropStr(PropAppName)
	if app != "" {
		app = "/" + app
	}
	url := "`" + app + d.Url + qp + "`"

	for _, h := range d.Headers {
		sl.Printlnf("let %s: any | null = null;", util.CamelCase(h.Name))
	}

	isBuiltinResp := false
	hasData := false
	if respTypeName != "" {
		respTypeName = guessTsItfName(respTypeName)
		if respTypeName == "Resp" || respTypeName == "GnResp" {
			hasErrorCode := false
			hasError := false
			for _, d := range d.JsonResponseDesc.Fields {
				if d.FieldName == "Data" {
					hasData = true
				} else if d.FieldName == "Error" {
					hasError = true
				} else if d.FieldName == "ErrorCode" {
					hasErrorCode = true
				}
			}
			isBuiltinResp = hasErrorCode && hasError
		}
	}

	reqVar := ""
	if reqTypeName != "" {
		reqTypeName = guessTsItfName(reqTypeName)
		{
			n := reqTypeName
			if d.JsonRequestDesc.IsSlice {
				n = n + "[]"
			}
			sl.Printlnf("let req: %s | null = null;", n)
		}

		reqVar = ", req"
	}
	n := "any"
	if respTypeName != "" && !isBuiltinResp {
		n = respTypeName
	}
	sl.Printlnf("this.http.%s<%s>(%s%s", strings.ToLower(d.Method), n, url, reqVar)

	if len(d.Headers) > 0 {
		sl.Printf(",")
		sl.Printlnf(util.Spaces(2) + "{")
		sl.Printlnf(util.Spaces(4) + "headers: {")
		for _, h := range d.Headers {
			sl.Printlnf(util.Spaces(6)+"\"%s\": %s", h.Name, util.CamelCase(h.Name))
		}
		sl.Printlnf(util.Spaces(4) + "}")
		sl.Printlnf(util.Spaces(2) + "})")
	} else {
		sl.Printf(")")
	}
	sl.Printlnf(util.Spaces(2) + ".subscribe({")

	if respTypeName != "" {
		sl.Printlnf(util.Spaces(4) + "next: (resp) => {")
		if isBuiltinResp {
			sl.Printlnf(util.Spaces(6) + "if (resp.error) {")
			sl.Printlnf(util.Spaces(8) + "this.snackBar.open(resp.msg, \"ok\", { duration: 6000 })")
			sl.Printlnf(util.Spaces(8) + "return;")
			sl.Printlnf(util.Spaces(6) + "}")
			if hasData {
				if dataField, ok := util.SliceFilterFirst(d.JsonResponseDesc.Fields,
					func(d FieldDesc) bool { return d.FieldName == "Data" }); ok {
					sl.Printlnf(util.Spaces(6)+"let dat: %s = resp.data;", guessTsTypeName(dataField))
				}
			}
		}
		sl.Printlnf(util.Spaces(4) + "},")
	} else {
		sl.Printlnf(util.Spaces(4) + "next: () => {")
		sl.Printlnf(util.Spaces(4) + "},")
	}

	sl.Printlnf(util.Spaces(4) + "error: (err) => {")
	sl.Printlnf(util.Spaces(6) + "console.log(err)")
	sl.Printlnf(util.Spaces(6) + "this.snackBar.open(\"Request failed, unknown error\", \"ok\", { duration: 3000 })")
	sl.Printlnf(util.Spaces(4) + "}")
	sl.Printlnf(util.Spaces(2) + "});\n")
	return sl.String()
}

func genTClientDemo(d httpRouteDoc, reqTypeName string, respTypeName string) string {
	sl := new(util.SLPinter)

	respGeneName := respTypeName
	if respGeneName == "" {
		respGeneName = "any"
	} else {
		respGeneName = guessGoTypName(respTypeName)
		if respGeneName == "Resp" {
			for _, n := range d.JsonResponseDesc.Fields {
				if n.FieldName == "Data" {
					respGeneName = guessGoTypName(n.TypeName)
					break
				}
			}
		}
		if respGeneName == "Resp" {
			respGeneName = "any"
		}
	}

	qhp := make([]string, 0, len(d.QueryParams)+len(d.Headers))
	for _, s := range d.QueryParams {
		qhp = append(qhp, fmt.Sprintf("%s string", util.CamelCase(s.Name)))
	}
	for _, s := range d.Headers {
		qhp = append(qhp, fmt.Sprintf("%s string", util.CamelCase(s.Name)))
	}

	qh := ""
	if len(qhp) > 0 {
		qh = ", " + strings.Join(qhp, ", ")
	}

	if reqTypeName != "" {
		mn := "SendRequest"
		if len(reqTypeName) > 1 {
			mn = fmt.Sprintf("Send%s%s", strings.ToUpper(string(reqTypeName[0])), string(reqTypeName[1:]))
		}
		reqn := reqTypeName
		if d.JsonRequestDesc.IsSlice {
			reqn = "[]" + reqn
		}
		if respGeneName == "any" {
			sl.Printlnf("func %s(rail miso.Rail, req %s%s) error {", mn, reqn, qh)
		} else {
			respn := respGeneName
			if d.JsonResponseDesc.IsSlice {
				respn = "[]" + respn
			}
			sl.Printlnf("func %s(rail miso.Rail, req %s%s) (%s, error) {", mn, reqn, qh, respn)
		}
	} else {
		mn := "SendRequest"
		if respGeneName == "any" {
			sl.Printlnf("func %s(rail miso.Rail%s) error {", mn, qh)
		} else {
			respn := respGeneName
			if d.JsonResponseDesc.IsSlice {
				respn = "[]" + respn
			}
			sl.Printlnf("func %s(rail miso.Rail%s) (%s, error) {", mn, qh, respn)
		}
	}

	sl.LinePrefix = "\t"
	sl.Printlnf("var res miso.GnResp[%s]", respGeneName)
	sl.Printf("\n%serr := miso.NewDynTClient(rail, \"%s\", \"%s\")", util.Tabs(1), d.Url, GetPropStr(PropAppName))

	for _, q := range d.QueryParams {
		cname := util.CamelCase(q.Name)
		sl.Printf(".\n%sAddQueryParams(\"%s\", %s)", util.Tabs(2), cname, cname)
	}

	for _, h := range d.Headers {
		cname := util.CamelCase(h.Name)
		sl.Printf(".\n%sAddHeader(\"%s\", %s)", util.Tabs(2), cname, cname)
	}

	httpCall := d.Method
	if len(httpCall) > 1 {
		httpCall = strings.ToUpper(string(d.Method[0])) + strings.ToLower(string(d.Method[1:]))
	}
	um := strings.ToUpper(d.Method)
	if reqTypeName != "" {
		if um == "POST" {
			sl.Printf(".\n%sPostJson(req)", util.Tabs(2))
		} else if um == "PUT" {
			sl.Printf(".\n%sPutJson(req)", util.Tabs(2))
		}
	} else {
		if um == "POST" {
			sl.Printf(".\n%sPost(nil)", util.Tabs(2))
		} else if um == "PUT" {
			sl.Printf(".\n%sPut(nil)", util.Tabs(2))
		} else {
			sl.Printf(".\n%s%s()", util.Tabs(2), httpCall)
		}
	}
	sl.Printf(".\n%sJson(&res)", util.Tabs(2))

	sl.Printlnf("if err != nil {")
	sl.Printlnf("%srail.Errorf(\"Request failed, %%v\", err)", util.Tabs(1))
	if respGeneName == "any" {
		sl.Printlnf("%sreturn err", util.Tabs(1))
	} else {
		if strings.HasPrefix(respGeneName, "*") {
			sl.Printlnf("%sreturn nil, err", util.Tabs(1))
		} else {
			dat := "dat"
			switch respGeneName {
			case "string":
				dat = "\"\""
			case "int", "int8", "int16", "int32", "int64", "float", "float32", "float64":
				dat = "0"
			case "bool":
				dat = "false"
			default:
				sl.Printlnf("%svar dat %s", util.Tabs(1), respGeneName)
			}
			sl.Printlnf("%sreturn %s, err", util.Tabs(1), dat)
		}
	}
	sl.Printlnf("}")

	if respGeneName == "any" {
		sl.Printlnf("err = res.Err()")
		sl.Printlnf("if err != nil {")
		sl.Printlnf("%srail.Errorf(\"Request failed, %%v\", err)", util.Tabs(1))
		sl.Printlnf("}")
		sl.Printlnf("return err")
		sl.Printf("\n}")
		return sl.String()
	}

	sl.Printlnf("dat, err := res.Res()")
	sl.Printlnf("if err != nil {")
	sl.Printlnf("%srail.Errorf(\"Request failed, %%v\", err)", util.Tabs(1))
	sl.Printlnf("}")
	sl.Printlnf("return dat, err")
	sl.Printf("\n}")
	return sl.String()
}

type PipelineDoc struct {
	Name        string
	Desc        string
	Exchange    string
	RoutingKey  string
	Queue       string
	PayloadDesc JsonPayloadDesc
}

// Register func to supply PipelineDoc.
func AddGetPipelineDocFunc(f GetPipelineDocFunc) {
	getPipelineDocFuncs = append(getPipelineDocFuncs, f)
}

func guessTsTypeName(d FieldDesc) string {
	if len(d.Fields) > 0 {
		tsTypeName := guessTsItfName(d.TypeName)
		if strings.HasPrefix(d.TypeName, "[]") {
			return tsTypeName + "[]"
		}
		return tsTypeName
	} else {
		return guessTsPrimiTypeName(d.TypeName)
	}
}

// Associate xdesc value with the code appeared in field tag `xdesc:"..."`.
func AddXDesc(code string, desc string) {
	if xdescs == nil {
		xdescs = map[string]string{}
	}
	xdescs[code] = singleLine(desc)
}

// Get tag api description.
//
// If tag `desc` appears, returns the value first. If not, it looks for tag `xdesc`,
// and returns the value stored in xdescs map.
func getTagDesc(tag reflect.StructTag) string {
	if desc, ok := tag.Lookup(TagApiDocDesc); ok {
		return desc
	}
	if xt, ok := tag.Lookup(TagApiDocXDesc); ok {
		if v, ok := xdescs[xt]; ok {
			return v
		}
	}
	return ""
}

var singleLineRegex = regexp.MustCompile(` *[\t\n]+`)

func singleLine(v string) string {
	v = strings.TrimSpace(v)
	return singleLineRegex.ReplaceAllString(v, " ")
}

// Disable apidoc endpoint handler.
func DisableApidocEndpointRegister() {
	apiDocEndpointDisabled = true
}
