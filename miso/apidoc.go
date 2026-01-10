package miso

import (
	_ "embed"
	"fmt"
	"html/template"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/curtisnewbie/miso/tools"
	"github.com/curtisnewbie/miso/util/async"
	"github.com/curtisnewbie/miso/util/hash"
	"github.com/curtisnewbie/miso/util/json"
	"github.com/curtisnewbie/miso/util/osutil"
	"github.com/curtisnewbie/miso/util/rfutil"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/getkin/kin-openapi/openapi3"
	jsoniter "github.com/json-iterator/go"
)

const (
	demoPkgPath = "github.com/curtisnewbie/miso/demo"
)

const (
	TagApiDocDesc  = "desc"
	TagApiDocXDesc = "xdesc"
)

var (
	golangMapGenericRexp = regexp.MustCompile(`map\[(.*)\](.*)`)

	ApiDocSkipParsingTypes = []ApiDocFuzzType{
		{"github.com/curtisnewbie/miso/middleware/user-vault/common", "User"},
		{"github.com/curtisnewbie/miso/util/hash", "Set"},
		{"github.com/curtisnewbie/miso/util/hash", "SyncSet"},
		{"github.com/curtisnewbie/miso/middleware/money", "Amt"},
	}

	ApiDocNotInclTypes = []ApiDocFuzzType{
		{"github.com/curtisnewbie/miso/miso", "PageRes"},
		{"github.com/curtisnewbie/miso/miso", "Paging"},
	}

	ApiDocTypeAlias = map[string]string{
		"Time":         "int64",
		"*atom.Time":   "int64",
		"Set[any]":     "[]any",
		"Set[string]":  "[]string",
		"Set[int]":     "[]int",
		"Set[int32]":   "[]int32",
		"Set[int64]":   "[]int64",
		"Set[float32]": "[]float32",
		"Set[float64]": "[]float64",
		"*Time":        "int64",

		// TODO: fix following pkg name prefix
		"*hash.Set[any]":     "[]any",
		"*hash.Set[string]":  "[]string",
		"*hash.Set[int]":     "[]int",
		"*hash.Set[int32]":   "[]int32",
		"*hash.Set[int64]":   "[]int64",
		"*hash.Set[float32]": "[]float32",
		"*hash.Set[float64]": "[]float64",
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

		routes := GetHttpRoutes()
		docs := buildHttpRouteDoc(routes)
		var pipelineDoc []PipelineDoc
		for _, f := range getPipelineDocFuncs {
			pipelineDoc = append(pipelineDoc, f()...)
		}

		err := writeApiDocFile(rail, docs.routeDocs, pipelineDoc)
		if err != nil {
			rail.Errorf("Failed to write api-doc markdown file: %v", err)
			return err
		}

		err = writeApiDocOpenApiSpec(rail, docs)
		if err != nil {
			rail.Errorf("Failed to write api-doc open-api 3.0 spec file: %v", err)
			return err
		}

		err = writeApiDocGoFile(rail, docs.globalDoc.GoTypeDef, docs.routeDocs)
		if err != nil {
			rail.Errorf("Failed to write api-doc golang file: %v", err)
			return err
		}

		return nil
	})
}

type ApiDocFuzzType struct {
	PkgPath  string
	TypeName string
}

type GetPipelineDocFunc func() []PipelineDoc

type globalHttpRouteDoc struct {
	GoTypeDef []string
}

type httpRouteDoc struct {
	Name                    string           // api func name
	Url                     string           // http request url
	Method                  string           // http method
	Extra                   map[string][]any // extra metadata
	Desc                    string           // description of the route (metadata).
	Scope                   string           // the documented access scope of the route, it maybe "PUBLIC" or something else (metadata).
	Resource                string           // the documented resource that the route should be bound to (metadata).
	Headers                 []ParamDoc       // the documented header parameters that will be used by the endpoint (metadata).
	QueryParams             []ParamDoc       // the documented query parameters that will used by the endpoint (metadata).
	JsonRequestValue        *reflect.Value   // reflect.Value of json request object
	JsonRequestDesc         TypeDesc         // the documented json request type that is expected by the endpoint (metadata).
	JsonResponseValue       *reflect.Value   // reflect.Value of json response object
	JsonResponseDesc        TypeDesc         // the documented json response type that will be returned by the endpoint (metadata).
	Curl                    string           // curl demo
	JsonReqTsDef            string           // json request type def in ts
	JsonRespTsDef           string           // json response type def in ts
	JsonTsDef               string           // json requests & response type def in ts
	JsonReqGoDef            string           // json request type def in go
	JsonReqGoDefTypeName    string           // json request type name in go
	JsonRespGoDef           string           // json response type def in go
	JsonRespGoDefTypeName   string           // json response type name in go
	NgHttpClientDemo        string           // angular http client demo
	NgTableDemo             string           // angular table demo
	MisoTClientDemo         string           // miso TClient demo
	MisoTClientWithoutTypes string           // miso TClient demo without golang type definitions
	OpenApiDoc              string
}

type FieldDesc struct {
	GoFieldName           string      // field name in golang
	JsonName              string      // field name in json
	TypeNameAlias         string      // type name in golang or type name alias translated
	TypePkg               string      // pkg path of the type in golang
	OriginTypeName        string      // type name in golang (reflect.Type.Name()) without import path
	OriginTypeNameWithPkg string      // type name in golang with import pkg
	DescTag               string      // `desc` tag value
	JsonTag               string      // `json` tag value
	ValidTag              string      // `validate` tag value
	isSliceOrArray        bool        // slice or array []T
	isSliceOfPointer      bool        // slice of pointer []*T
	isMap                 bool        // map
	isPointer             bool        // *T
	Fields                []FieldDesc // struct fields
}

func (f FieldDesc) TypeInfo() (pkg string, name string) {
	return f.TypePkg, f.OriginTypeName
}

func FuzzMatchType(v interface {
	TypeInfo() (pkg string, typeName string)
}, against ApiDocFuzzType) bool {
	p, n := v.TypeInfo()
	return strings.Contains(p, against.PkgPath) && strings.Contains(n, against.TypeName)
}

func FuzzMatchTypes(v interface {
	TypeInfo() (pkg string, typeName string)
}, against []ApiDocFuzzType) bool {
	for _, a := range against {
		if FuzzMatchType(v, a) {
			return true
		}
	}
	return false
}

func (f FieldDesc) guessTsPrimiTypeName() string {
	var tname string
	if f.isMap {
		re := golangMapGenericRexp
		tname = "Map<any, any>"
		if sm := re.FindStringSubmatch(f.goFieldTypeName()); len(sm) > 2 {
			tname = fmt.Sprintf("Map<%v,%v>", guessTsPrimiTypeName(sm[1]), guessTsPrimiTypeName(sm[2]))
		}
	} else {
		tname = guessTsPrimiTypeName(f.TypeNameAlias)
	}
	return tname
}

func (f FieldDesc) isMisoPkg() bool {
	return strings.HasPrefix(f.TypePkg, "github.com/curtisnewbie/miso")
}

func (f FieldDesc) isMisoDemoPkg() bool {
	return strings.HasPrefix(f.typePkg(), demoPkgPath)
}

func (f FieldDesc) typePkg() string {
	return f.TypePkg
}

func (f FieldDesc) isBuiltInType() bool {
	return f.isMap
}

func (f FieldDesc) pureGoTypeName() string {
	n := f.OriginTypeName
	if f.isMap {
		return n
	}
	return pureGoTypeName(n)
}

func (f FieldDesc) comment(withSlash bool) string {
	var desc string = f.DescTag
	var comment string
	appendComment := func(r string) {
		r = strings.TrimSpace(r)
		if comment != "" {
			if !strutil.HasAnySuffix(comment, ".", ",") {
				comment += "."
			}
			comment += " " + r
		} else {
			if withSlash {
				comment += " // "
			}
			comment += r
		}
	}
	if desc != "" {
		appendComment(desc)
	}
	if f.ValidTag != "" {
		var remaining string
		rules := strings.Split(f.ValidTag, ",")
		appendRemaining := func(r string) {
			if remaining != "" && !strutil.HasAnySuffix(remaining, ".", ",") {
				remaining += "."
			}
			remaining += " " + r
		}

		for _, r := range rules {
			ok, pvr := parseValidRule(r)
			if !ok {
				appendRemaining(r)
				continue
			}

			switch pvr.rule {
			case ValidNotEmpty:
				appendRemaining("Required.")
			case ValidMaxLen:
				appendRemaining(fmt.Sprintf("Max length: %v.", pvr.param))
			case ValidMember:
				enums := strings.Split(pvr.param, "|")
				if len(enums) < 1 {
					continue
				}
				enumDesc := strings.Join(slutil.QuoteStrSlice(enums), ",")
				appendComment(fmt.Sprintf("Enums: [%v].", enumDesc))
			default:
				continue
			}
		}
		if remaining != "" {
			appendComment(remaining)
		}
	}
	return comment
}

func (f FieldDesc) goFieldTypeName() string {
	// is miso pkg types, type def not included or type has alias name (e.g., atom.Time)
	if f.isMisoPkg() && !f.isMisoDemoPkg() && (FuzzMatchTypes(f, ApiDocNotInclTypes) || f.TypeNameAlias != f.OriginTypeName) {
		return f.OriginTypeNameWithPkg
	}
	ptn := f.pureGoTypeName()
	if f.isSliceOfPointer {
		return "[]*" + ptn
	}
	if f.isSliceOrArray {
		return "[]" + ptn
	}
	if f.isPointer {
		return "*" + ptn
	}
	return f.OriginTypeName
}

// See [BuildTypeDesc]
type TypeDesc struct {
	TypeName   string
	TypePkg    string
	IsPtr      bool
	IsPtrSlice bool
	IsSlice    bool
	IsSlicePtr bool
	Fields     []FieldDesc
}

func (f TypeDesc) TypeInfo() (pkg string, name string) {
	return f.TypePkg, f.TypeName
}

func (f TypeDesc) toOpenApiReq(reqName string) *openapi3.SchemaRef {
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

func (j TypeDesc) buildSchema(fields []FieldDesc, sec *openapi3.Schema) {
	if len(fields) < 1 {
		return
	}
	for _, f := range fields {
		var ref *openapi3.SchemaRef = j.buildSchemaRef(f)
		sec.WithPropertyRef(f.JsonName, ref)
	}
}

func (j TypeDesc) buildSchemaRef(f FieldDesc) *openapi3.SchemaRef {
	// simple types
	if len(f.Fields) < 1 {
		str := j.simpleTypeRef(f.OriginTypeName)
		str.Value.Description = f.DescTag
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
	sec.Description = f.DescTag

	ref := &openapi3.SchemaRef{Value: sec}
	return ref
}

func (j TypeDesc) simpleTypeRef(typeName string) *openapi3.SchemaRef {
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

func (j TypeDesc) toOpenApiResp(respName string) *openapi3.Response {
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

func (j TypeDesc) pureGoTypeName() string {
	n := j.TypeName
	return pureGoTypeName(n)
}

func (f TypeDesc) isBuiltInType() bool {
	return false
}

func (j TypeDesc) isMisoPkg() bool {
	return strings.HasPrefix(j.TypePkg, "github.com/curtisnewbie/miso")
}

func (j TypeDesc) typePkg() string {
	return j.TypePkg
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
	globalDoc globalHttpRouteDoc
	openapi   *openapi3.T
}

func buildHttpRouteDoc(hr []HttpRoute) httpRouteDocs {
	docs := make([]httpRouteDoc, 0, len(hr))
	filteredPathPatterns := []string{
		"/debug/pprof/**",
		"/doc/api/**",
		"/health",
		"/metrics",
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
	openApiSpecPatterns := GetPropStrSlice(PropServerGenerateEndpointDocOpenApiSpecPathPatterns)

	matchGoFilePathPatterns := matchGoFilePathPatternFunc()
	seenGlobalGoTypeDef := hash.NewSet[string]()
	globalGoTypeDef := []string{}

	for _, r := range hr {
		excl := false
		for _, filtered := range filteredPathPatterns {
			if strutil.MatchPath(filtered, r.Url) {
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

		if v, ok := slutil.First(r.Extra[ExtraName]); ok {
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

		addGlobalGoTypeDef := matchGoFilePathPatterns(r.Url)

		// json stuff
		if d.JsonRequestValue != nil {
			d.JsonRequestDesc = BuildTypeDesc(*d.JsonRequestValue)
			if IsDebugLevel() {
				Debugf("JsonRequestDesc:\n%v", json.TrySWriteJson(d.JsonRequestDesc))
			}

			d.JsonReqTsDef = genTsDef(d.JsonRequestDesc)
			d.JsonReqGoDef, d.JsonReqGoDefTypeName = genGoDef(d.JsonRequestDesc, hash.NewSet[string]())

			if addGlobalGoTypeDef {
				td, _ := genGoDef(d.JsonRequestDesc, seenGlobalGoTypeDef)
				td = strings.TrimSpace(td)
				if td != "" {
					globalGoTypeDef = append(globalGoTypeDef, td)
				}
			}

			d.JsonTsDef = d.JsonReqTsDef
		}

		if d.JsonResponseValue != nil {
			d.JsonResponseDesc = BuildTypeDesc(*d.JsonResponseValue)
			if IsDebugLevel() {
				Debugf("JsonResponseDesc:\n%v", json.TrySWriteJson(d.JsonResponseDesc))
			}

			d.JsonRespTsDef = genTsDef(d.JsonResponseDesc)
			d.JsonRespGoDef, d.JsonRespGoDefTypeName = genGoDef(d.JsonResponseDesc, hash.NewSet[string]())

			if addGlobalGoTypeDef {
				td, _ := genGoDef(d.JsonResponseDesc, seenGlobalGoTypeDef)
				td = strings.TrimSpace(td)
				if td != "" {
					globalGoTypeDef = append(globalGoTypeDef, td)
				}
			}

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
		d.MisoTClientWithoutTypes = d.MisoTClientDemo
		if d.JsonRespGoDef != "" {
			d.MisoTClientDemo = d.JsonRespGoDef + "\n" + d.MisoTClientDemo
		}
		if d.JsonReqGoDef != "" {
			d.MisoTClientDemo = d.JsonReqGoDef + "\n" + d.MisoTClientDemo
		}

		// openapi 3.0.0
		var matchSpecPattern bool = true
		if len(openApiSpecPatterns) > 0 {
			matchSpecPattern = strutil.MatchPathAny(openApiSpecPatterns, d.Url)
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
		globalDoc: globalHttpRouteDoc{
			GoTypeDef: globalGoTypeDef,
		},
		openapi: rootSpec,
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

type JsonPayloadDesc = TypeDesc

// Parse value's type information to build json style description.
//
// Only supports struct, pointer and slice.
func BuildTypeDesc(v reflect.Value) TypeDesc {
	switch v.Kind() {
	case reflect.Pointer:
		v = reflect.New(v.Type().Elem()).Elem()
		jpd := BuildTypeDesc(v)
		jpd.IsPtr = true
		if jpd.IsSlice {
			jpd.IsPtrSlice = true
		}
		return jpd
	case reflect.Struct:
		return TypeDesc{Fields: buildTypeDescRecur(v, nil), TypeName: rfutil.TypeName(v.Type()), TypePkg: rfutil.TypePkgPath(v.Type())}
	case reflect.Slice:
		et := v.Type().Elem()
		switch et.Kind() {
		case reflect.Struct:
			ev := reflect.New(et).Elem()
			d := TypeDesc{IsSlice: true, Fields: buildTypeDescRecur(ev, nil), TypeName: rfutil.TypeName(et),
				TypePkg: rfutil.TypePkgPath(et)}
			return d
		case reflect.Pointer:
			ev := reflect.New(et).Elem()
			d := TypeDesc{IsSlice: true, IsSlicePtr: true, Fields: buildTypeDescRecur(ev, nil), TypeName: rfutil.TypeName(et),
				TypePkg: rfutil.TypePkgPath(et)}
			return d
		}
	}
	return TypeDesc{TypeName: v.Type().Name(), TypePkg: rfutil.TypePkgPath(v.Type())}
}

func buildTypeDescRecur(v reflect.Value, seen *hash.Set[reflect.Type]) []FieldDesc {
	if seen == nil {
		st := hash.NewSet[reflect.Type]()
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
		var jsonTag string
		if v := f.Tag.Get("json"); v != "" {
			if v == "-" {
				continue
			}
			jsonTag = v

			tokz := strings.TrimSpace(strings.Split(v, ",")[0])
			if tokz == "" { // e.g., ',omitEmpty'
				jsonName = json.NamingStrategyTranslate(f.Name)
			} else {
				jsonName = tokz
			}
		} else {
			jsonName = json.NamingStrategyTranslate(f.Name)
		}

		originTypeName := rfutil.TypeName(f.Type)
		typeName, typeAliasMatched := translateTypeAlias(originTypeName)

		if jsonTag == "" {
			jsonTag = jsonName
		}
		jd := FieldDesc{
			GoFieldName:           f.Name,
			JsonName:              jsonName,
			TypeNameAlias:         typeName,
			TypePkg:               rfutil.TypePkgPath(f.Type),
			OriginTypeName:        originTypeName,
			OriginTypeNameWithPkg: f.Type.String(),
			DescTag:               getTagDesc(f.Tag),
			ValidTag:              getTagValid(f.Tag),
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
					jd.OriginTypeName = rfutil.TypeName(et)
					jd.OriginTypeNameWithPkg = et.String()
					jd.TypePkg = rfutil.TypePkgPath(et)
					jd.TypeNameAlias, _ = translateTypeAlias(jd.OriginTypeName)
					switch et.Kind() {
					case reflect.Slice, reflect.Array:
						jd.isSliceOrArray = true
						if et.Elem().Kind() == reflect.Pointer {
							jd.isSliceOfPointer = true
						}
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
				Tracef("reflect.Value is zero or nil, not displayed in api doc, type: %v, field: %v", t.Name(), jd.JsonName)
			}
		} else {
			switch fv.Kind() {
			case reflect.Slice, reflect.Array:
				jd.isSliceOrArray = true
				if fv.Type().Elem().Kind() == reflect.Pointer {
					jd.isSliceOfPointer = true
				}
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

func reflectAppendJsonDesc(t reflect.Type, v reflect.Value, fields []FieldDesc, seen *hash.Set[reflect.Type]) []FieldDesc {
	if t.Kind() == reflect.Struct {
		fields = append(fields, buildTypeDescRecur(v, seen)...)
	} else if t.Kind() == reflect.Slice {
		et := t.Elem()
		if et.Kind() == reflect.Struct {
			ev := reflect.New(et).Elem()
			fields = append(fields, buildTypeDescRecur(ev, seen)...)
		}
	} else if t.Kind() == reflect.Pointer {
		seen.Add(t)
		defer seen.Del(t)
		ev := reflect.New(t.Elem()).Elem()
		if ev.Kind() == reflect.Struct {
			fields = append(fields, buildTypeDescRecur(ev, seen)...)
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
		if d.isSliceOrArray {
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
					v = 0
				case "float32", "float64", "*float32", "*float64":
					v = 0.0
				case "bool", "*bool":
					v = false
				}
				jm[d.JsonName] = v
			}
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

func skipParsingType(f interface {
	TypeInfo() (pkg string, typeName string)
}) bool {
	return FuzzMatchTypes(f, ApiDocSkipParsingTypes)
}

// generate one or more golang type definitions.
func genGoDef(rv TypeDesc, seenTypeDef hash.Set[string]) (string, string) {
	if rv.TypeName == "any" {
		return "", ""
	}

	if rv.TypeName == "Resp" || rv.TypeName == "GnResp" {
		for _, f := range rv.Fields {
			if f.GoFieldName == "Data" {
				if f.OriginTypeName == "any" {
					return "", ""
				}
				deferred := make([]func(), 0, 10)
				sb, writef := strutil.NewIndWritef("\t")

				ptn := f.pureGoTypeName()

				if !skipParsingType(f) {
					inclTypeDef := inclGoTypeDef(f, seenTypeDef)
					if inclTypeDef {
						writef(0, "type %s struct {", ptn)
					}
					genJsonGoDefRecur(1, writef, &deferred, f.Fields, inclTypeDef, seenTypeDef)
					if inclTypeDef {
						writef(0, "}")
					}
				}
				for i := 0; i < len(deferred); i++ {
					deferred[i]()
				}
				return sb.String(), ptn
			}
		}
		return "", ""
	} else {
		deferred := make([]func(), 0, 10)
		sb, writef := strutil.NewIndWritef("\t")
		ptn := rv.pureGoTypeName()

		if !skipParsingType(rv) {
			inclTypeDef := inclGoTypeDef(rv, seenTypeDef)
			if inclTypeDef {
				writef(0, "type %s struct {", ptn)
			}

			genJsonGoDefRecur(1, writef, &deferred, rv.Fields, inclTypeDef, seenTypeDef)
			if inclTypeDef {
				writef(0, "}")
			}
		}

		for i := 0; i < len(deferred); i++ {
			deferred[i]()
		}
		return sb.String(), ptn
	}
}

func inclGoTypeDef(f interface {
	TypeInfo() (pkg string, typeName string)
	isBuiltInType() bool
	pureGoTypeName() string
}, seenTypeDef hash.Set[string]) bool {

	if f.isBuiltInType() { // e.g., map
		return false
	}

	pgn := f.pureGoTypeName()
	p, n := f.TypeInfo()
	Debugf("inclGoTypeDef: %v, %v, %v\n", pgn, p, n)

	// TODO: temp fix
	switch pgn {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "string", "bool", "byte":
		return false
	}

	if FuzzMatchTypes(f, ApiDocNotInclTypes) {
		return false
	}

	if !seenTypeDef.Add(pgn) {
		return false
	}
	return true
}

func genJsonGoDefRecur(indentc int, writef strutil.IndWritef, deferred *[]func(), fields []FieldDesc, writeField bool,
	seenTypeDef hash.Set[string]) {

	for _, f := range fields {
		var jsonTag string
		if f.JsonTag != "" {
			jsonTag = fmt.Sprintf(" `json:\"%v\"`", f.JsonTag)
		}
		ffields := f.Fields

		if len(ffields) > 0 {

			if writeField {
				fieldTypeName := f.goFieldTypeName()
				writef(indentc, "%s %s%s", f.GoFieldName, fieldTypeName, jsonTag)
			}

			// TODO: this is ugly
			if !skipParsingType(f) {
				inclType := inclGoTypeDef(f, seenTypeDef)
				*deferred = append(*deferred, func() {
					if inclType {
						writef(0, "")
						writef(0, "type %s struct {", f.pureGoTypeName())
					}
					genJsonGoDefRecur(1, writef, deferred, f.Fields, inclType, seenTypeDef)
					if inclType {
						writef(0, "}")
					}
				})
			}

		} else {
			if !writeField {
				continue
			}
			fieldTypeName := f.goFieldTypeName()
			var comment string = f.comment(true)
			if comment != "" {
				fieldDec := fmt.Sprintf("%s %s%s", f.GoFieldName, fieldTypeName, jsonTag)
				writef(indentc, "%-30s%s", fieldDec, comment)
			} else {
				writef(indentc, "%s %s%s", f.GoFieldName, fieldTypeName, jsonTag)
			}
		}
	}
}

// generate one or more typescript interface definitions based on a set of jsonDesc.
func genTsDef(payload TypeDesc) string {
	var typeName string = payload.TypeName
	if len(payload.Fields) < 1 && typeName == "" {
		return ""
	}
	sb, writef := strutil.NewIndWritef("  ")
	seenType := hash.NewSet[string]()
	tsTypeName := guessTsItfName(typeName)
	seenType.Add(tsTypeName)
	writef(0, "export interface %s {", tsTypeName)
	deferred := make([]func(), 0, 10)
	genJsonTsDefRecur(1, writef, true, &deferred, payload.Fields, seenType)
	writef(0, "}")

	for i := 0; i < len(deferred); i++ {
		writef(0, "")
		deferred[i]()
	}
	return sb.String()
}

func genJsonTsDefRecur(indentc int, writef strutil.IndWritef, writeField bool, deferred *[]func(), descs []FieldDesc, seenType hash.Set[string]) {
	for i := range descs {
		d := descs[i]

		if len(d.Fields) > 0 {
			tsTypeName := guessTsItfName(d.TypeNameAlias)
			if writeField {
				n := tsTypeName
				if strings.HasPrefix(d.TypeNameAlias, "[]") {
					n += "[]"
				}
				writef(indentc, "%s?: %s;", d.JsonName, n)
			}

			// TODO: this is ugly
			inclType := seenType.Add(tsTypeName)
			stopDesc := false
			if inclType {
				if skipParsingType(d) {
					inclType = false
					stopDesc = true
				}
			}
			if !stopDesc {
				*deferred = append(*deferred, func() {
					if inclType {
						writef(0, "export interface %s {", tsTypeName)
					}
					genJsonTsDefRecur(1, writef, inclType, deferred, d.Fields, seenType)
					if inclType {
						writef(0, "}")
					}
				})
			}
		} else if writeField {
			var tname string = d.guessTsPrimiTypeName()
			var comment string = d.comment(true)
			if comment != "" {
				fieldDec := fmt.Sprintf("%s?: %s", d.JsonName, tname)
				writef(indentc, "%-30s%s", fieldDec+";", comment)
			} else {
				writef(indentc, "%s?: %s;", d.JsonName, tname)
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

func guessGoGenericEleName(n string) string {
	if len(n) < 3 {
		return ""
	}
	if n[len(n)-1] != ']' {
		return ""
	}
	i := strings.IndexByte(n, '[')
	if i < 0 {
		return ""
	}
	v := n[i+1 : len(n)-1]
	return guessGoTypName(v)
}

func guessGoTypName(n string) string {
	tsTypeName := guessTsItfName(n)
	return tsTypeName
}

func genNgTableDemo(d httpRouteDoc) string {
	var respTypeName string = d.JsonResponseDesc.TypeName
	sl := new(strutil.SLPinter)
	sl.Println(`<table mat-table [dataSource]="tabdata" class="mb-4" style="width: 100%;">`)

	var cols []string

	if respTypeName != "" {
		respTypeName = guessTsItfName(respTypeName)

		// Resp.Data -> PageRes -> PageRes.Payload
		if respTypeName == "Resp" {
			pl, hasData := slutil.FirstMatch(d.JsonResponseDesc.Fields, func(j FieldDesc) bool {
				return j.GoFieldName == "Data"
			})
			if hasData {
				pl, hasPayload := slutil.FirstMatch(pl.Fields, func(j FieldDesc) bool {
					return j.GoFieldName == "Payload"
				})
				if hasPayload {
					for _, f := range pl.Fields {
						sl.Printlnf(strutil.Tabs(1)+"<ng-container matColumnDef=\"%v\">", f.JsonName)
						sl.Printlnf(strutil.Tabs(2)+"<th mat-header-cell *matHeaderCellDef> %s </th>", f.GoFieldName)
						if f.OriginTypeName == "Time" || f.OriginTypeName == "*Time" || f.OriginTypeName == "atom.Time" || f.OriginTypeName == "*atom.Time" {
							sl.Printlnf(strutil.Tabs(2)+"<td mat-cell *matCellDef=\"let u\"> {{u.%s | date: 'yyyy-MM-dd HH:mm:ss'}} </td>", f.JsonName)
						} else {
							sl.Printlnf(strutil.Tabs(2)+"<td mat-cell *matCellDef=\"let u\"> {{u.%s}} </td>", f.JsonName)
						}
						sl.Println(strutil.Tabs(1) + "</ng-container>")
						cols = append(cols, "'"+f.JsonName+"'")
					}
				}
			}
		}
	}

	colstr := "[" + strings.Join(cols, ",") + "]"
	sl.Printlnf(strutil.Tabs(1)+"<tr mat-row *matRowDef=\"let row; columns: %v;\"></tr>", colstr)
	sl.Printlnf(strutil.Tabs(1)+"<tr mat-header-row *matHeaderRowDef=\"%s\"></tr>", colstr)
	sl.Printlnf(`</table>`)
	return sl.String()
}

func genNgHttpClientDemo(d httpRouteDoc) string {
	var reqTypeName, respTypeName string = d.JsonRequestDesc.TypeName, d.JsonResponseDesc.TypeName
	sl := new(strutil.SLPinter)
	sl.Printlnf("import { MatSnackBar } from \"@angular/material/snack-bar\";")
	sl.Printlnf("import { HttpClient } from \"@angular/common/http\";")
	sl.Printlnf("")
	sl.Printlnf("constructor(")
	sl.Println(strutil.Spaces(2) + "private snackBar: MatSnackBar,")
	sl.Println(strutil.Spaces(2) + "private http: HttpClient")
	sl.Printlnf(") {}")
	sl.Printlnf("")

	var mn string = "sendRequest"
	if d.Name != "" {
		if strings.HasPrefix(strings.ToLower(d.Name), "api") {
			dr := []rune(d.Name)
			if len(dr) > 3 {
				mn = strutil.CamelCase(string(dr[3:]))
			} else {
				mn = strutil.CamelCase(d.Name)
			}
		} else {
			mn = strutil.CamelCase(d.Name)
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
		cname := strutil.CamelCase(q.Name)
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
		sl.Printlnf("let %s: any | null = null;", strutil.CamelCase(h.Name))
	}

	isBuiltinResp := false
	hasData := false
	if respTypeName != "" {
		respTypeName = guessTsItfName(respTypeName)
		if respTypeName == "Resp" || respTypeName == "GnResp" {
			hasErrorCode := false
			hasError := false
			for _, d := range d.JsonResponseDesc.Fields {
				if d.GoFieldName == "Data" {
					hasData = true
				} else if d.GoFieldName == "Error" {
					hasError = true
				} else if d.GoFieldName == "ErrorCode" {
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
		sl.Println(strutil.Spaces(2) + "{")
		sl.Println(strutil.Spaces(4) + "headers: {")
		for _, h := range d.Headers {
			sl.Printlnf(strutil.Spaces(6)+"\"%s\": %s", h.Name, strutil.CamelCase(h.Name))
		}
		sl.Println(strutil.Spaces(4) + "}")
		sl.Println(strutil.Spaces(2) + "})")
	} else {
		sl.Printf(")")
	}
	sl.Println(strutil.Spaces(2) + ".subscribe({")

	if respTypeName != "" {
		sl.Println(strutil.Spaces(4) + "next: (resp) => {")
		if isBuiltinResp {
			sl.Println(strutil.Spaces(6) + "if (resp.error) {")
			sl.Println(strutil.Spaces(8) + "this.snackBar.open(resp.msg, \"ok\", { duration: 6000 })")
			sl.Println(strutil.Spaces(8) + "return;")
			sl.Println(strutil.Spaces(6) + "}")
			if hasData {
				if dataField, ok := slutil.FirstMatch(d.JsonResponseDesc.Fields,
					func(d FieldDesc) bool { return d.GoFieldName == "Data" }); ok {
					sl.Printlnf(strutil.Spaces(6)+"let dat: %s = resp.data;", guessTsTypeName(dataField))
				}
			}
		}
		sl.Println(strutil.Spaces(4) + "},")
	} else {
		sl.Println(strutil.Spaces(4) + "next: () => {")
		sl.Println(strutil.Spaces(4) + "},")
	}

	sl.Println(strutil.Spaces(4) + "error: (err) => {")
	sl.Println(strutil.Spaces(6) + "console.log(err)")
	sl.Println(strutil.Spaces(6) + "this.snackBar.open(\"Request failed, unknown error\", \"ok\", { duration: 3000 })")
	sl.Println(strutil.Spaces(4) + "}")
	sl.Println(strutil.Spaces(2) + "});")

	sl.LinePrefix = ""
	sl.Printlnf("}\n")

	return sl.String()
}

func genTClientDemo(d httpRouteDoc) (code string) {
	var reqTypeName, respTypeName string = d.JsonRequestDesc.TypeName, d.JsonResponseDesc.TypeName
	sl := new(strutil.SLPinter)

	buildTypeName := func(s string, isPtrSlice, isSlicePtr, isSlice, isPtr bool) string {
		if isPtrSlice {
			s = "*[]" + s
		} else if isSlicePtr || (isSlice && isPtr) {
			s = "[]*" + s
		} else if isSlice {
			s = "[]" + s
		} else if isPtr {
			s = "*" + s
		}
		return s
	}

	respGeneName := respTypeName
	if respGeneName == "" {
		respGeneName = "any"
	} else {
		respGeneName = guessGoTypName(respTypeName)
		if respGeneName == "Resp" {
			for _, n := range d.JsonResponseDesc.Fields {
				if n.GoFieldName == "Data" {

					respGeneName = guessGoTypName(n.TypeNameAlias)
					if n.isMisoPkg() && !n.isMisoDemoPkg() {
						respGeneName = "miso." + respGeneName
						if v := guessGoGenericEleName(n.TypeNameAlias); v != "" {
							respGeneName += "[" + v + "]"
						}
					}
					isPtr := n.isPointer
					isSlice := n.isSliceOrArray
					if n.isSliceOfPointer {
						isPtr = true
						isSlice = true
					}
					respGeneName = buildTypeName(respGeneName, false, false, isSlice, isPtr)
					break
				}
			}
			if respGeneName == "Resp" {
				respGeneName = "any"
				respGeneName = buildTypeName(respGeneName, d.JsonResponseDesc.IsPtrSlice,
					d.JsonResponseDesc.IsSlicePtr, d.JsonResponseDesc.IsSlice, d.JsonResponseDesc.IsPtr)
			}
		} else {
			respGeneName = buildTypeName(respGeneName, d.JsonResponseDesc.IsPtrSlice,
				d.JsonResponseDesc.IsSlicePtr, d.JsonResponseDesc.IsSlice, d.JsonResponseDesc.IsPtr)
		}
	}

	qhp := make([]string, 0, len(d.QueryParams)+len(d.Headers))
	for _, s := range d.QueryParams {
		qhp = append(qhp, fmt.Sprintf("%s string", strutil.CamelCase(s.Name)))
	}
	for _, s := range d.Headers {
		qhp = append(qhp, fmt.Sprintf("%s string", strutil.CamelCase(s.Name)))
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

	{
		desc := strings.TrimSpace(d.Desc)
		if desc != "" {
			sl.Println(strutil.SAddLineIndent(desc, "// "))
		}
	}
	if reqTypeName != "" {
		reqn := buildTypeName(reqTypeName, d.JsonRequestDesc.IsPtrSlice, d.JsonRequestDesc.IsSlicePtr,
			d.JsonRequestDesc.IsSlice, d.JsonRequestDesc.IsPtr)

		if respGeneName == "any" {
			sl.Printlnf("func %s(rail miso.Rail, req %s%s) error {", mn, reqn, qh)
		} else {
			sl.Printlnf("func %s(rail miso.Rail, req %s%s) (%s, error) {", mn, reqn, qh, respGeneName)
		}
	} else {
		if respGeneName == "any" {
			sl.Printlnf("func %s(rail miso.Rail%s) error {", mn, qh)
		} else {
			sl.Printlnf("func %s(rail miso.Rail%s) (%s, error) {", mn, qh, respGeneName)
		}
	}

	sl.LinePrefix = "\t"
	sl.Printlnf("var res miso.GnResp[%s]", respGeneName)
	sl.Printf("\n%serr := miso.NewDynClient(rail, \"%s\", \"%s\")", strutil.Tabs(1), d.Url, GetPropStr(PropAppName))

	for _, q := range d.QueryParams {
		cname := strutil.CamelCase(q.Name)
		sl.Printf(".\n%sAddQuery(\"%s\", %s)", strutil.Tabs(2), cname, cname)
	}

	for _, h := range d.Headers {
		cname := strutil.CamelCase(h.Name)
		sl.Printf(".\n%sAddHeader(\"%s\", %s)", strutil.Tabs(2), cname, cname)
	}

	httpCall := d.Method
	if len(httpCall) > 1 {
		httpCall = strings.ToUpper(string(d.Method[0])) + strings.ToLower(string(d.Method[1:]))
	}
	um := strings.ToUpper(d.Method)
	if reqTypeName != "" {
		if um == "POST" {
			sl.Printf(".\n%sPostJson(req)", strutil.Tabs(2))
		} else if um == "PUT" {
			sl.Printf(".\n%sPutJson(req)", strutil.Tabs(2))
		}
	} else {
		if um == "POST" {
			sl.Printf(".\n%sPost(nil)", strutil.Tabs(2))
		} else if um == "PUT" {
			sl.Printf(".\n%sPut(nil)", strutil.Tabs(2))
		} else {
			sl.Printf(".\n%s%s()", strutil.Tabs(2), httpCall)
		}
	}
	sl.Printf(".\n%sJson(&res)", strutil.Tabs(2))

	sl.Printlnf("if err != nil {")
	sl.Printlnf("%srail.Errorf(\"Request failed, %%v\", err)", strutil.Tabs(1))
	if respGeneName == "any" {
		sl.Printlnf("%sreturn err", strutil.Tabs(1))
	} else {
		if strings.HasPrefix(respGeneName, "*") {
			sl.Printlnf("%sreturn nil, err", strutil.Tabs(1))
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
				sl.Printlnf("%svar dat %s", strutil.Tabs(1), respGeneName)
			}
			sl.Printlnf("%sreturn %s, err", strutil.Tabs(1), dat)
		}
	}
	sl.Printlnf("}")

	if respGeneName == "any" {
		sl.Printlnf("err = res.Err()")
		sl.Printlnf("if err != nil {")
		sl.Printlnf("%srail.Errorf(\"Request failed, %%v\", err)", strutil.Tabs(1))
		sl.Printlnf("}")
		sl.Printlnf("return err")
		sl.Printf("\n}")
		return sl.String()
	}

	sl.Printlnf("dat, err := res.Res()")
	sl.Printlnf("if err != nil {")
	sl.Printlnf("%srail.Errorf(\"Request failed, %%v\", err)", strutil.Tabs(1))
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
	PayloadDesc TypeDesc
}

// Register func to supply PipelineDoc.
func AddGetPipelineDocFunc(f GetPipelineDocFunc) {
	getPipelineDocFuncs = append(getPipelineDocFuncs, f)
}

func guessTsTypeName(d FieldDesc) string {
	if len(d.Fields) > 0 {
		tsTypeName := guessTsItfName(d.TypeNameAlias)
		if strings.HasPrefix(d.TypeNameAlias, "[]") {
			return tsTypeName + "[]"
		}
		return tsTypeName
	} else {
		return d.guessTsPrimiTypeName()
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
					Type: &openapi3.Types{"string"},
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
					Type: &openapi3.Types{"string"},
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

func writeApiDocOpenApiSpec(rail Rail, docs httpRouteDocs) error {
	if oapiFile := GetPropStr(PropServerGenerateEndpointDocOpenApiSpecFile); oapiFile != "" {
		f, err := osutil.OpenRWFile(oapiFile)
		if err != nil {
			rail.Errorf("Failed to openapi spec json file, %v, %v", oapiFile, err)
			return nil // ignore
		}
		f.Truncate(0)
		defer f.Close()

		openApiJson, _ := json.SWriteIndent(docs.openapi)
		openApiJson = json.SIndent(openApiJson)
		f.WriteString(openApiJson)
	}
	return nil
}

func writeApiDocFile(rail Rail, routes []httpRouteDoc, pipelineDoc []PipelineDoc) error {
	outf := GetPropStr(PropServerGenerateEndpointDocFile)
	if outf != "" {
		_ = osutil.MkdirParentAll(outf)
		f, err := osutil.OpenRWFile(outf)
		if err != nil {
			rail.Debugf("Failed to open API doc file, %v, %v", outf, err)
			return nil // ignore
		}
		f.Truncate(0)
		defer f.Close()

		markdown := genMarkDownDoc(routes, pipelineDoc)
		_, err = f.WriteString(markdown)
		return err
	}
	return nil
}

func matchGoFilePathPatternFunc() func(u string) bool {
	goFilePathPatterns := GetPropStrSlice(PropServerApiDocGoPathPatterns)
	goFileExclPathPatterns := GetPropStrSlice(PropServerApiDocGoExclPathPatterns)
	matchPatterns := func(u string) bool {
		if len(goFileExclPathPatterns) > 0 && strutil.MatchPathAny(goFileExclPathPatterns, u) {
			return false
		}
		if len(goFilePathPatterns) < 1 {
			return true
		}
		return strutil.MatchPathAny(goFilePathPatterns, u)
	}
	return matchPatterns
}

func writeApiDocGoFile(rail Rail, goTypeDefs []string, routes []httpRouteDoc) error {
	fp := GetPropStrTrimmed(PropServerApiDocGoFile)
	if fp == "" {
		return nil
	}
	_ = osutil.MkdirParentAll(fp)
	f, err := osutil.OpenRWFile(fp, true)
	if err != nil {
		return err
	}
	defer f.Close()

	matchPatterns := matchGoFilePathPatternFunc()
	b := strings.Builder{}
	b.WriteString("\npackage " + path.Base(path.Dir(fp)))
	b.WriteString("\nimport (")
	b.WriteString("\n\t\"github.com/curtisnewbie/miso/miso\"")
	b.WriteString("\n\t\"github.com/curtisnewbie/miso/util/atom\"")
	b.WriteString("\n\t\"github.com/curtisnewbie/miso/util/hash\"")
	b.WriteString("\n)")

	for _, td := range goTypeDefs {
		b.WriteString("\n" + td + "\n")
	}

	for _, r := range routes {
		if !matchPatterns(r.Url) {
			continue
		}
		if r.MisoTClientWithoutTypes != "" && !GetPropBool(PropServerGenerateEndpointDocFileExclTClientDemo) {
			b.WriteString("\n" + r.MisoTClientWithoutTypes + "\n")
		}
	}

	f.Truncate(0)
	compile := GetPropBool(PropServerApiDocGoCompileFile)
	start := ""
	if !compile {
		start += "//go:build miso_gen_do_not_build\n"
	}
	start += "// auto generated by miso, please do not modify"
	_, err = f.WriteString(start + "\n" + b.String())
	if err != nil {
		return err
	}

	async.PanicSafeRun(func() { tools.RunGoImports(fp) })
	return nil
}

func translateTypeAlias(tname string) (string, bool) {
	typeAlias, typeAliasMatched := ApiDocTypeAlias[tname]
	if typeAliasMatched {
		return typeAlias, true
	}
	return tname, false
}
