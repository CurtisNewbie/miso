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
	"github.com/getkin/kin-openapi/openapi3"
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
		"*util.ETime": "int64",
		"util.ETime":  "int64",
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
	PostServerBootstrap(func(rail Rail) error {
		if IsProdMode() || !GetPropBool(PropServerGenerateEndpointDocEnabled) || GetPropBool(PropAppTestEnv) {
			return nil
		}
		outf := GetPropStr(PropServerGenerateEndpointDocFile)
		if outf == "" {
			return nil
		}
		_ = util.MkdirParentAll(outf)
		f, err := util.ReadWriteFile(outf)
		if err != nil {
			rail.Debugf("Failed to open API doc file, %v, %v", outf, err)
			return nil
		}
		f.Truncate(0)
		defer f.Close()

		routes := GetHttpRoutes()
		docs := buildHttpRouteDoc(routes)
		var pipelineDoc []PipelineDoc
		for _, f := range getPipelineDocFuncs {
			pipelineDoc = append(pipelineDoc, f()...)
		}
		markdown := genMarkDownDoc(docs.routeDocs, pipelineDoc)
		_, _ = f.WriteString(markdown)

		if oapiFile := GetPropStr(PropServerGenerateEndpointDocOpenApiSpecFile); oapiFile != "" {
			f, err := util.ReadWriteFile(oapiFile)
			if err != nil {
				rail.Debugf("Failed to openapi spec json file, %v, %v", oapiFile, err)
				return nil
			}
			f.Truncate(0)
			defer f.Close()

			openApiJson, _ := json.SWriteIndent(docs.openapi)
			openApiJson = json.SIndent(openApiJson)
			f.WriteString(openApiJson)
		}

		return nil
	})
}

type GetPipelineDocFunc func() []PipelineDoc

type httpRouteDoc struct {
	Name                  string           // api func name
	Url                   string           // http request url
	Method                string           // http method
	Extra                 map[string][]any // extra metadata
	Desc                  string           // description of the route (metadata).
	Scope                 string           // the documented access scope of the route, it maybe "PUBLIC" or something else (metadata).
	Resource              string           // the documented resource that the route should be bound to (metadata).
	Headers               []ParamDoc       // the documented header parameters that will be used by the endpoint (metadata).
	QueryParams           []ParamDoc       // the documented query parameters that will used by the endpoint (metadata).
	JsonRequestValue      *reflect.Value   // reflect.Value of json request object
	JsonRequestDesc       JsonPayloadDesc  // the documented json request type that is expected by the endpoint (metadata).
	JsonResponseValue     *reflect.Value   // reflect.Value of json response object
	JsonResponseDesc      JsonPayloadDesc  // the documented json response type that will be returned by the endpoint (metadata).
	Curl                  string           // curl demo
	JsonReqTsDef          string           // json request type def in ts
	JsonRespTsDef         string           // json response type def in ts
	JsonTsDef             string           // json requests & response type def in ts
	JsonReqGoDef          string           // json request type def in go
	JsonReqGoDefTypeName  string           // json request type name in go
	JsonRespGoDef         string           // json response type def in go
	JsonRespGoDefTypeName string           // json response type name in go
	NgHttpClientDemo      string           // angular http client demo
	NgTableDemo           string           // angular table demo
	MisoTClientDemo       string           // miso TClient demo
	OpenApiDoc            string
}

type FieldDesc struct {
	FieldName             string      // field name in golang
	Name                  string      // field name in json
	TypeName              string      // type name in golang or type name alias translated
	TypePkg               string      // pkg path of the type in golang
	OriginTypeName        string      // type name in golang (reflect.Type.Name()) without import path
	OriginTypeNameWithPkg string      // type name in golang with import pkg
	Desc                  string      // `desc` tag value
	JsonTag               string      // `json` tag value
	Valid                 string      // `validate` tag value
	Fields                []FieldDesc // struct fields
	isSliceOrArray        bool
	isMap                 bool
	isPointer             bool
}

func (f FieldDesc) isMisoPkg() bool {
	return strings.HasPrefix(f.TypePkg, "github.com/curtisnewbie/miso")
}

func (f FieldDesc) pureGoTypeName() string {
	n := f.OriginTypeName
	if f.isMap {
		return n
	}
	return pureGoTypeName(n)
}

func (f FieldDesc) comment() string {
	var desc string = f.Desc
	var comment string
	if desc != "" {
		comment = " // " + desc
	}
	if f.Valid != "" {
		if comment != "" {
			comment = strings.TrimSpace(comment) + ", " + f.Valid
		} else {
			comment = " // " + f.Valid
		}
	}
	return comment
}

func (f FieldDesc) goFieldTypeName() string {
	if f.isMisoPkg() {
		return f.OriginTypeNameWithPkg
	}
	ptn := f.pureGoTypeName()
	if f.isSliceOrArray {
		return "[]" + ptn
	}
	if f.isPointer {
		return "*" + ptn
	}
	return f.OriginTypeName
}

type JsonPayloadDesc struct {
	TypeName string
	TypePkg  string
	IsSlice  bool
	Fields   []FieldDesc
}

func (f JsonPayloadDesc) toOpenApiReq(reqName string) *openapi3.SchemaRef {
	var ref *openapi3.SchemaRef
	if len(f.Fields) > 0 {
		sec := &openapi3.Schema{}
		f.buildSchema(f.Fields, sec)
		ref = &openapi3.SchemaRef{Value: sec}
	} else {
		ref = f.simpleTypeRef(f.TypeName)
	}
	ref.Value.Description = reqName
	return ref
}

func (j JsonPayloadDesc) buildSchema(fields []FieldDesc, sec *openapi3.Schema) {
	if len(fields) < 1 {
		return
	}
	for _, f := range fields {
		var ref *openapi3.SchemaRef = j.buildSchemaRef(f)
		sec.WithPropertyRef(f.Name, ref)
	}
}

func (j JsonPayloadDesc) buildSchemaRef(f FieldDesc) *openapi3.SchemaRef {
	// simple types
	if len(f.Fields) < 1 {
		str := j.simpleTypeRef(f.OriginTypeName)
		str.Value.Description = f.Desc
		return str
	}
	var sec *openapi3.Schema
	if f.isSliceOrArray {
		sec = openapi3.NewArraySchema()
		secit := &openapi3.Schema{}
		sec.WithItems(secit)
		j.buildSchema(f.Fields, secit)
	} else {
		sec = openapi3.NewObjectSchema()
		j.buildSchema(f.Fields, sec)
	}
	sec.Description = f.Desc

	ref := &openapi3.SchemaRef{Value: sec}
	return ref
}

func (j JsonPayloadDesc) simpleTypeRef(typeName string) *openapi3.SchemaRef {
	var ref *openapi3.SchemaRef
	switch strings.TrimSpace(typeName) {
	case "string", "*string":
		ref = &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}
	case "int", "int8", "int16", "int32", "int64", "*int", "*int8", "*int16", "*int32", "*int64":
		ref = &openapi3.SchemaRef{Value: openapi3.NewIntegerSchema()}
	case "float32", "float64", "*float32", "*float64":
		ref = &openapi3.SchemaRef{Value: openapi3.NewFloat64Schema()}
	case "bool", "*bool":
		ref = &openapi3.SchemaRef{Value: openapi3.NewBoolSchema()}
	case "byte", "*byte":
		ref = &openapi3.SchemaRef{Value: openapi3.NewBytesSchema()}
	default:
		// util.Printlnf("> typename: %v", typeName)
		ref = &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}
	}
	return ref
}

func (j JsonPayloadDesc) toOpenApiResp(respName string) *openapi3.Response {
	var ref *openapi3.SchemaRef
	r := &openapi3.Response{}
	if len(j.Fields) > 0 {
		sec := &openapi3.Schema{}
		j.buildSchema(j.Fields, sec)
		ref = &openapi3.SchemaRef{Value: sec}
	} else {
		ref = j.simpleTypeRef(j.TypeName)
	}
	ref.Value.Description = respName
	r.Description = &j.TypeName
	r.WithJSONSchemaRef(ref)
	return r
}

func (j JsonPayloadDesc) pureGoTypeName() string {
	n := j.TypeName
	return pureGoTypeName(n)
}

func (j JsonPayloadDesc) isMisoPkg() bool {
	return strings.HasPrefix(j.TypePkg, "github.com/curtisnewbie/miso")
}

func pureGoTypeName(n string) string {
	if len(n) == 0 {
		return n
	}

	// []Mytype -> MyType
	v, ok := strings.CutPrefix(n, "[]")
	if ok {
		if len(n) == 2 {
			return n
		}
		n = v
	}

	// MyType[...] -> MyType
	if n[len(n)-1] == ']' {
		j := strings.IndexByte(n, '[')
		if j > -1 {
			n = n[:j]
		}
	}

	// xxx.MyType -> MyType
	i := strings.LastIndexByte(n, '.')
	if i > -1 {
		n = n[i+1:]
	}

	// *MyType -> MyType
	if v, ok := strings.CutPrefix(n, "*"); ok {
		n = v
	}
	return n
}

type httpRouteDocs struct {
	routeDocs []httpRouteDoc
	openapi   *openapi3.T
}

func buildHttpRouteDoc(hr []HttpRoute) httpRouteDocs {
	docs := make([]httpRouteDoc, 0, len(hr))
	filteredPathPatterns := []string{
		"/debug/pprof/**",
		"/doc/api/**",
	}

	rootSpec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   GetPropStr(PropAppName),
			Version: "1.0.0",
		},
	}
	if v := GetPropStr(PropServerGenerateEndpointDocOpenApiSpecServer); v != "" {
		rootSpec.AddServer(&openapi3.Server{URL: v})
	}
	patterns := GetPropStrSlice(PropServerGenerateEndpointDocOpenApiSpecPathPatterns)

	for _, r := range hr {
		excl := false
		for _, filtered := range filteredPathPatterns {
			if util.MatchPath(filtered, r.Url) {
				excl = true
				break
			}
		}
		if excl {
			continue
		}

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

		if v, ok := util.SliceFirst(r.Extra[ExtraName]); ok {
			if s, ok := v.(string); ok {
				d.Name = s
			}
		}

		if l, ok := r.Extra[ExtraJsonRequest]; ok && len(l) > 0 {
			jsonRequestVal := l[0]
			if jsonRequestVal != nil {
				v := reflect.ValueOf(jsonRequestVal)
				d.JsonRequestValue = &v
			}
		}

		if l, ok := r.Extra[ExtraJsonResponse]; ok && len(l) > 0 {
			jsonResponseVal := l[0]
			if jsonResponseVal != nil {
				v := reflect.ValueOf(jsonResponseVal)
				d.JsonResponseValue = &v
			}
		}

		// json stuff
		if d.JsonRequestValue != nil {
			d.JsonRequestDesc = BuildJsonPayloadDesc(*d.JsonRequestValue)
			d.JsonReqTsDef = genJsonTsDef(d.JsonRequestDesc)
			d.JsonReqGoDef, d.JsonReqGoDefTypeName = genJsonGoDef(d.JsonRequestDesc)
			d.JsonTsDef = d.JsonReqTsDef
		}

		if d.JsonResponseValue != nil {
			d.JsonResponseDesc = BuildJsonPayloadDesc(*d.JsonResponseValue)
			d.JsonRespTsDef = genJsonTsDef(d.JsonResponseDesc)
			d.JsonRespGoDef, d.JsonRespGoDefTypeName = genJsonGoDef(d.JsonResponseDesc)
			if d.JsonTsDef != "" {
				d.JsonTsDef += "\n"
			}
			d.JsonTsDef += d.JsonRespTsDef
		}

		// curl
		d.Curl = genRouteCurl(d)

		// ng http client
		d.NgHttpClientDemo = genNgHttpClientDemo(d)

		// ng table demo
		if _, ok := r.Extra[ExtraNgTable]; ok {
			d.NgTableDemo = genNgTableDemo(d)
		}

		// miso http TClient
		d.MisoTClientDemo = genTClientDemo(d)
		if d.JsonRespGoDef != "" {
			d.MisoTClientDemo = d.JsonRespGoDef + "\n" + d.MisoTClientDemo
		}
		if d.JsonReqGoDef != "" {
			d.MisoTClientDemo = d.JsonReqGoDef + "\n" + d.MisoTClientDemo
		}

		// openapi 3.0.0
		var matchSpecPattern bool = true
		if len(patterns) > 0 {
			matchSpecPattern = util.MatchPathAny(patterns, d.Url)
		}

		if matchSpecPattern {
			d.OpenApiDoc = genOpenApiDoc(d, rootSpec)
		} else {
			d.OpenApiDoc = genOpenApiDoc(d, nil)
		}

		docs = append(docs, d)
	}
	return httpRouteDocs{
		routeDocs: docs,
		openapi:   rootSpec,
	}
}

func genMarkDownDoc(hr []httpRouteDoc, pd []PipelineDoc) string {
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
				b.WriteString(util.Spaces(2))
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
				b.WriteString(util.Spaces(2))
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
			b.WriteString(util.Spaces(2) + "```sh\n")
			b.WriteString(util.SAddLineIndent(r.Curl, util.Spaces(2)))
			b.WriteString(util.Spaces(2) + "```\n")
		}

		if r.MisoTClientDemo != "" && !GetPropBool(PropServerGenerateEndpointDocFileExclTClientDemo) {
			b.WriteRune('\n')
			b.WriteString("- Miso HTTP Client (experimental, demo may not work):\n")
			b.WriteString(util.Spaces(2) + "```go\n")
			b.WriteString(util.SAddLineIndent(r.MisoTClientDemo+"\n", util.Spaces(2)))
			b.WriteString(util.Spaces(2) + "```\n")
		}

		if r.NgHttpClientDemo != "" && !GetPropBool(PropServerGenerateEndpointDocFileExclNgClientDemo) {
			if r.JsonTsDef != "" {
				b.WriteRune('\n')
				b.WriteString("- JSON Request / Response Object In TypeScript:\n")
				b.WriteString(util.Spaces(2) + "```ts\n")
				b.WriteString(util.SAddLineIndent(r.JsonTsDef, util.Spaces(2)))
				b.WriteString(util.Spaces(2) + "```\n")
			}

			b.WriteRune('\n')
			b.WriteString("- Angular HttpClient Demo:\n")
			b.WriteString(util.Spaces(2) + "```ts\n")
			b.WriteString(util.SAddLineIndent(r.NgHttpClientDemo, util.Spaces(2)))
			b.WriteString(util.Spaces(2) + "```\n")
		}

		if r.OpenApiDoc != "" && !GetPropBool(PropServerGenerateEndpointDocFileExclOpenApi) {
			b.WriteRune('\n')
			b.WriteString("- Open Api (experimental, demo may not work):\n")
			b.WriteString(util.Spaces(2) + "```json\n")
			b.WriteString(util.SAddLineIndent(r.OpenApiDoc+"\n", util.Spaces(2)))
			b.WriteString(util.Spaces(2) + "```\n")
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
func BuildJsonPayloadDesc(v reflect.Value) JsonPayloadDesc {
	switch v.Kind() {
	case reflect.Pointer:
		v = reflect.New(v.Type().Elem()).Elem()
		return BuildJsonPayloadDesc(v)
	case reflect.Struct:
		return JsonPayloadDesc{Fields: buildJsonDesc(v, nil), TypeName: v.Type().Name(), TypePkg: v.Type().PkgPath()}
	case reflect.Slice:
		et := v.Type().Elem()
		if et.Kind() == reflect.Struct {
			ev := reflect.New(et).Elem()
			return JsonPayloadDesc{IsSlice: true, Fields: buildJsonDesc(ev, nil), TypeName: et.Name(), TypePkg: et.PkgPath()}
		}
	}
	return JsonPayloadDesc{TypeName: v.Type().Name(), TypePkg: v.Type().PkgPath()}
}

func buildJsonDesc(v reflect.Value, seen *util.Set[reflect.Type]) []FieldDesc {
	if seen == nil {
		st := util.NewSet[reflect.Type]()
		seen = &st
	}

	t := v.Type()
	seen.Add(t)
	defer seen.Del(t)
	if t.Kind() == reflect.Pointer {
		rt := reflect.New(t.Elem()).Elem().Type()
		seen.Add(rt)
		defer seen.Del(rt)
	}

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

		var jsonName string
		if v := f.Tag.Get("json"); v != "" {
			if v == "-" {
				continue
			}
			jsonName = v
		} else {
			jsonName = json.LowercaseNamingStrategy(f.Name)
		}

		originTypeName := util.TypeName(f.Type)
		typeAlias, typeAliasMatched := ApiDocTypeAlias[originTypeName]
		var typeName string
		if typeAliasMatched {
			typeName = typeAlias
		} else {
			typeName = originTypeName
		}

		jsonTag := f.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = jsonName
		}
		jd := FieldDesc{
			FieldName:             f.Name,
			Name:                  jsonName,
			TypeName:              typeName,
			TypePkg:               f.Type.PkgPath(),
			OriginTypeName:        originTypeName,
			OriginTypeNameWithPkg: f.Type.String(),
			Desc:                  getTagDesc(f.Tag),
			Valid:                 getTagValid(f.Tag),
			JsonTag:               jsonTag,
			Fields:                []FieldDesc{},
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
					jd.TypePkg = et.PkgPath()
					jd.OriginTypeName = jd.TypeName
					switch et.Kind() {
					case reflect.Slice, reflect.Array:
						jd.isSliceOrArray = true
					case reflect.Map:
						jd.isMap = true
					case reflect.Pointer:
						jd.isPointer = true
					}
					if !seen.Has(et) {
						jd.Fields = reflectAppendJsonDesc(et, ele, jd.Fields, seen)
					}
				}
			} else {
				appendable = false // e.g., the any field in GnResp[any]{}
				Tracef("reflect.Value is zero or nil, not displayed in api doc, type: %v, field: %v", t.Name(), jd.Name)
			}
		} else {
			switch fv.Kind() {
			case reflect.Slice, reflect.Array:
				jd.isSliceOrArray = true
			case reflect.Map:
				jd.isMap = true
			case reflect.Pointer:
				jd.isPointer = true
			}
			if !seen.Has(f.Type) {
				jd.Fields = reflectAppendJsonDesc(f.Type, fv, jd.Fields, seen)
			}
		}

		if appendable {
			jds = append(jds, jd)
		}
	}
	return jds
}

func reflectAppendJsonDesc(t reflect.Type, v reflect.Value, fields []FieldDesc, seen *util.Set[reflect.Type]) []FieldDesc {
	if t.Kind() == reflect.Struct {
		fields = append(fields, buildJsonDesc(v, seen)...)
	} else if t.Kind() == reflect.Slice {
		et := t.Elem()
		if et.Kind() == reflect.Struct {
			ev := reflect.New(et).Elem()
			fields = append(fields, buildJsonDesc(ev, seen)...)
		}
	} else if t.Kind() == reflect.Pointer {
		seen.Add(t)
		defer seen.Del(t)
		ev := reflect.New(t.Elem()).Elem()
		if ev.Kind() == reflect.Struct {
			fields = append(fields, buildJsonDesc(ev, seen)...)
		}
	} else if t.Kind() == reflect.Interface {
		if !v.IsZero() && !v.IsNil() {
			if ele := v.Elem(); ele.IsValid() {
				et := ele.Type()
				fields = reflectAppendJsonDesc(et, ele, fields, seen)
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

	notGenerateDoc := !GetPropBool(PropServerGenerateEndpointDocEnabled)
	notRegisterDocWeb := !GetPropBool(PropServerGenerateEndpointDocApiEnabled)
	if IsProdMode() || notGenerateDoc || apiDocEndpointDisabled || notRegisterDocWeb {
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

	HttpGet("/doc/api", RawHandler(
		func(inb *Inbound) {
			defer DebugTimeOp(rail, time.Now(), "gen api doc")

			docs := buildHttpRouteDoc(GetHttpRoutes())
			var pipelineDoc []PipelineDoc
			for _, f := range getPipelineDocFuncs {
				pipelineDoc = append(pipelineDoc, f()...)
			}
			markdown := genMarkDownDoc(docs.routeDocs, pipelineDoc)

			w, _ := inb.Unwrap()
			if err := apiDocTmpl.ExecuteTemplate(w, "apiDocTempl",
				struct {
					App         string
					HttpDoc     []httpRouteDoc
					PipelineDoc []PipelineDoc
					Markdown    string
				}{
					App:         GetPropStr(PropAppName),
					HttpDoc:     docs.routeDocs,
					PipelineDoc: pipelineDoc,
					Markdown:    markdown,
				}); err != nil {
				rail.Errorf("failed to serve apiDocTmpl, %v", err)
			}
		})).
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

/*
type structFieldVal struct {
	v reflect.Value
	t reflect.StructField
}

func collectStructFieldValues(rv reflect.Value) []structFieldVal {
	if rv.Kind() != reflect.Struct {
		return []structFieldVal{}
	}

	fields := make([]structFieldVal, 0, rv.NumField())
	t := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i)
		ft := t.Field(i)
		if ft.IsExported() {
			fields = append(fields, structFieldVal{
				v: fv,
				t: ft,
			})
		}

	}
	return fields
}
*/

// generate one or more golang type definitions.
func genJsonGoDef(rv JsonPayloadDesc) (string, string) {
	if rv.TypeName == "any" || len(rv.Fields) < 1 {
		return "", ""
	}
	seenTypeDef := util.NewSet[string]()

	if rv.TypeName == "Resp" || rv.TypeName == "GnResp" {
		for _, f := range rv.Fields {
			if f.FieldName == "Data" {
				if f.OriginTypeName == "any" || len(f.Fields) < 1 {
					return "", ""
				}
				inclTypeDef := inclGoTypeDef(f, seenTypeDef)
				sb, writef := util.NewIndWritef("\t")
				ptn := f.pureGoTypeName()
				if inclTypeDef {
					writef(0, "type %s struct {", ptn)
				}
				deferred := make([]func(), 0, 10)
				genJsonGoDefRecur(1, writef, &deferred, f.Fields, inclTypeDef, seenTypeDef)
				if inclTypeDef {
					writef(0, "}")
				}
				for i := 0; i < len(deferred); i++ {
					deferred[i]()
				}
				return sb.String(), ptn
			}
		}
		return "", ""
	} else {
		sb, writef := util.NewIndWritef("\t")
		inclTypeDef := inclGoTypeDef(rv, seenTypeDef)
		ptn := rv.pureGoTypeName()
		if inclTypeDef {
			writef(0, "type %s struct {", ptn)
		}
		deferred := make([]func(), 0, 10)
		genJsonGoDefRecur(1, writef, &deferred, rv.Fields, inclTypeDef, seenTypeDef)
		if inclTypeDef {
			writef(0, "}")
		}

		for i := 0; i < len(deferred); i++ {
			deferred[i]()
		}
		return sb.String(), ptn
	}
}

func inclGoTypeDef(f interface {
	isMisoPkg() bool
	pureGoTypeName() string
}, seenTypeDef util.Set[string]) bool {
	if !seenTypeDef.Add(f.pureGoTypeName()) {
		return false
	}
	if !f.isMisoPkg() {
		return true
	}
	switch f.pureGoTypeName() {
	case "PageRes", "Paging":
		return false
	}
	return true
}

func genJsonGoDefRecur(indentc int, writef util.IndWritef, deferred *[]func(), fields []FieldDesc, writeField bool, seenTypeDef util.Set[string]) {
	for _, f := range fields {
		var jsonTag string
		if f.JsonTag != "" {
			jsonTag = fmt.Sprintf(" `json:\"%v\"`", f.JsonTag)
		}
		ffields := f.Fields

		if len(ffields) > 0 {

			if writeField {
				fieldTypeName := f.goFieldTypeName()
				writef(indentc, "%s %s%s", f.FieldName, fieldTypeName, jsonTag)
			}

			*deferred = append(*deferred, func() {
				inclTypeDef := inclGoTypeDef(f, seenTypeDef)
				if inclTypeDef {
					writef(0, "")
					writef(0, "type %s struct {", f.pureGoTypeName())
				}
				genJsonGoDefRecur(1, writef, deferred, f.Fields, inclTypeDef, seenTypeDef)
				if inclTypeDef {
					writef(0, "}")
				}
			})
		} else {
			if !writeField {
				continue
			}
			fieldTypeName := f.goFieldTypeName()
			var comment string = f.comment()
			if comment != "" {
				fieldDec := fmt.Sprintf("%s %s%s", f.FieldName, fieldTypeName, jsonTag)
				writef(indentc, "%-30s%s", fieldDec, comment)
			} else {
				writef(indentc, "%s %s%s", f.FieldName, fieldTypeName, jsonTag)
			}
		}
	}
}

// generate one or more typescript interface definitions based on a set of jsonDesc.
func genJsonTsDef(payload JsonPayloadDesc) string {
	var typeName string = payload.TypeName
	if len(payload.Fields) < 1 && typeName == "" {
		return ""
	}
	sb, writef := util.NewIndWritef("  ")
	seenType := util.NewSet[string]()
	tsTypeName := guessTsItfName(typeName)
	seenType.Add(tsTypeName)
	writef(0, "export interface %s {", tsTypeName)
	deferred := make([]func(), 0, 10)
	genJsonTsDefRecur(1, writef, &deferred, payload.Fields, seenType)
	writef(0, "}")

	for i := 0; i < len(deferred); i++ {
		writef(0, "")
		deferred[i]()
	}
	return sb.String()
}

func genJsonTsDefRecur(indentc int, writef util.IndWritef, deferred *[]func(), descs []FieldDesc, seenType util.Set[string]) {
	for i := range descs {
		d := descs[i]

		if len(d.Fields) > 0 {
			tsTypeName := guessTsItfName(d.TypeName)
			n := tsTypeName
			if strings.HasPrefix(d.TypeName, "[]") {
				n += "[]"
			}
			writef(indentc, "%s?: %s;", d.Name, n)

			if seenType.Add(tsTypeName) {
				*deferred = append(*deferred, func() {
					writef(0, "export interface %s {", tsTypeName)
					genJsonTsDefRecur(1, writef, deferred, d.Fields, seenType)
					writef(0, "}")
				})
			}

		} else {
			var tname string = guessTsPrimiTypeName(d.TypeName)
			var comment string = d.comment()
			if comment != "" {
				fieldDec := fmt.Sprintf("%s?: %s", d.Name, tname)
				writef(indentc, "%-30s%s", fieldDec+";", comment)
			} else {
				writef(indentc, "%s?: %s;", d.Name, tname)
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
			tname = guessTsItfName(v) + "[]"
		} else {
			tname = guessTsItfName(typeName)
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

func genNgTableDemo(d httpRouteDoc) string {
	var respTypeName string = d.JsonResponseDesc.TypeName
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
	sl.Printlnf(util.Tabs(1)+"<tr mat-row *matRowDef=\"let row; columns: %v;\"></tr>", colstr)
	sl.Printlnf(util.Tabs(1)+"<tr mat-header-row *matHeaderRowDef=\"%s\"></tr>", colstr)
	sl.Printlnf(`</table>`)
	return sl.String()
}

func genNgHttpClientDemo(d httpRouteDoc) string {
	var reqTypeName, respTypeName string = d.JsonRequestDesc.TypeName, d.JsonResponseDesc.TypeName
	sl := new(util.SLPinter)
	sl.Printlnf("import { MatSnackBar } from \"@angular/material/snack-bar\";")
	sl.Printlnf("import { HttpClient } from \"@angular/common/http\";")
	sl.Printlnf("")
	sl.Printlnf("constructor(")
	sl.Printlnf(util.Spaces(2) + "private snackBar: MatSnackBar,")
	sl.Printlnf(util.Spaces(2) + "private http: HttpClient")
	sl.Printlnf(") {}")
	sl.Printlnf("")

	var mn string = "sendRequest"
	if d.Name != "" {
		if strings.HasPrefix(strings.ToLower(d.Name), "api") {
			dr := []rune(d.Name)
			if len(dr) > 3 {
				mn = util.CamelCase(string(dr[3:]))
			} else {
				mn = util.CamelCase(d.Name)
			}
		} else {
			mn = util.CamelCase(d.Name)
		}
	} else if reqTypeName != "" {
		if len(reqTypeName) > 1 {
			mn = fmt.Sprintf("send%s%s", strings.ToUpper(string(reqTypeName[0])), string(reqTypeName[1:]))
		}
	}
	sl.Printlnf("%s() {", mn)
	sl.LinePrefix = "  "

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

	var url string
	if GetPropBool(PropServerGenerateEndpointDocInclPrefix) {
		app := GetPropStr(PropAppName)
		if app != "" {
			app = "/" + app
		}
		url = "`" + app + d.Url + qp + "`"
	} else {
		url = "`" + d.Url + qp + "`"
	}

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

	lmethod := strings.ToLower(d.Method)
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
	if (lmethod == "post" || lmethod == "put") && reqVar == "" {
		reqVar = ", null"
	}

	n := "any"
	if respTypeName != "" && !isBuiltinResp {
		n = respTypeName
	}
	sl.Printlnf("this.http.%s<%s>(%s%s", lmethod, n, url, reqVar)

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
	sl.Printlnf(util.Spaces(2) + "});")

	sl.LinePrefix = ""
	sl.Printlnf("}\n")

	return sl.String()
}

func genTClientDemo(d httpRouteDoc) (code string) {
	var reqTypeName, respTypeName string = d.JsonRequestDesc.TypeName, d.JsonResponseDesc.TypeName
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

	var mn string = "SendRequest"
	if d.Name != "" {
		mn = d.Name
	} else if reqTypeName != "" {
		if len(reqTypeName) > 1 {
			mn = fmt.Sprintf("Send%s%s", strings.ToUpper(string(reqTypeName[0])), string(reqTypeName[1:]))
		}
	}

	if reqTypeName != "" {
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
			respGeneName = respn
			sl.Printlnf("func %s(rail miso.Rail, req %s%s) (%s, error) {", mn, reqn, qh, respn)
		}
	} else {
		if respGeneName == "any" {
			sl.Printlnf("func %s(rail miso.Rail%s) error {", mn, qh)
		} else {
			respn := respGeneName
			if d.JsonResponseDesc.IsSlice {
				respn = "[]" + respn
			}
			respGeneName = respn
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

// Get tag valid.
func getTagValid(tag reflect.StructTag) string {
	if v, ok := tag.Lookup(TagValidationV2); ok {
		return v
	}
	if v, ok := tag.Lookup(TagValidationV1); ok {
		return v
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

func genOpenApiDoc(d httpRouteDoc, root *openapi3.T) string {
	title := d.Desc
	if title == "" {
		title = d.Method + " " + d.Url
	}

	servers := openapi3.Servers{}
	if v := GetPropStr(PropServerGenerateEndpointDocOpenApiSpecServer); v != "" {
		servers = openapi3.Servers{
			&openapi3.Server{URL: v},
		}
	}
	op := &openapi3.Operation{
		Summary:     d.Desc,
		Description: d.Desc,
	}

	for _, v := range d.QueryParams {
		op.AddParameter(&openapi3.Parameter{
			Name:        v.Name,
			In:          "query",
			Required:    false,
			Description: v.Desc,
			Schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"string"}, // TODO
				},
			},
		})
	}

	for _, v := range d.Headers {
		op.AddParameter(&openapi3.Parameter{
			Name:        v.Name,
			In:          "header",
			Required:    false,
			Description: v.Desc,
			Schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"string"}, // TODO
				},
			},
		})
	}

	if d.JsonRequestValue != nil {
		if p := d.JsonRequestDesc.toOpenApiReq(d.JsonReqGoDefTypeName); p != nil {
			op.RequestBody = &openapi3.RequestBodyRef{}
			op.RequestBody.Value = &openapi3.RequestBody{}
			op.RequestBody.Value.WithJSONSchemaRef(p)
		}
	}

	if p := d.JsonResponseDesc.toOpenApiResp(d.JsonRespGoDefTypeName); p != nil {
		op.AddResponse(200, p)
	}

	doc := openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   title,
			Version: "1.0.0",
		},
		Servers: servers,
	}
	doc.AddOperation(d.Url, d.Method, op)

	if root != nil {
		root.AddOperation(d.Url, d.Method, op)
	}

	j, _ := json.SWriteJson(doc)
	return j
}
