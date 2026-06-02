package miso

import (
	_ "embed"
	"fmt"
	"html/template"
	"path"
	"reflect"
	"regexp"
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

	ApiDocGoSkipParsingTypes = []ApiDocFuzzType{
		{"github.com/curtisnewbie/miso/middleware/user-vault/common", "User"},
		{"github.com/curtisnewbie/miso/util/hash", "Set"},
		{"github.com/curtisnewbie/miso/util/hash", "SyncSet"},
		{"github.com/curtisnewbie/miso/middleware/money", "Amt"},
	}

	ApiDocTsSkipParsingTypes = []ApiDocFuzzType{
		{"github.com/curtisnewbie/miso/util/hash", "Set"},
		{"github.com/curtisnewbie/miso/util/hash", "SyncSet"},
		{"github.com/curtisnewbie/miso/middleware/money", "Amt"},
	}

	ApiDocNotInclTypes = []ApiDocFuzzType{
		{"github.com/curtisnewbie/miso/miso", "PageRes"},
		{"github.com/curtisnewbie/miso/miso", "Paging"},
	}

	ApiDocTypeAlias = map[string]string{
		// Time is atom.Time (util/atom). Both short and full-module-path forms
		// are covered: runtime uses reflect.Type.Name() → "Time", static misoapi
		// uses types.TypeString(…, RelativeTo(…)) → full module path.
		"github.com/curtisnewbie/miso/util/atom.Time":  "int64",
		"*github.com/curtisnewbie/miso/util/atom.Time": "int64",

		// Set[T] is util/hash.Set[T], ~map[T]struct{}
		"github.com/curtisnewbie/miso/util/hash.Set[any]":      "[]any",
		"*github.com/curtisnewbie/miso/util/hash.Set[any]":     "[]any",
		"github.com/curtisnewbie/miso/util/hash.Set[string]":   "[]string",
		"*github.com/curtisnewbie/miso/util/hash.Set[string]":  "[]string",
		"github.com/curtisnewbie/miso/util/hash.Set[int]":      "[]int",
		"*github.com/curtisnewbie/miso/util/hash.Set[int]":     "[]int",
		"github.com/curtisnewbie/miso/util/hash.Set[int32]":    "[]int32",
		"*github.com/curtisnewbie/miso/util/hash.Set[int32]":   "[]int32",
		"github.com/curtisnewbie/miso/util/hash.Set[int64]":    "[]int64",
		"*github.com/curtisnewbie/miso/util/hash.Set[int64]":   "[]int64",
		"github.com/curtisnewbie/miso/util/hash.Set[float32]":  "[]float32",
		"*github.com/curtisnewbie/miso/util/hash.Set[float32]": "[]float32",
		"github.com/curtisnewbie/miso/util/hash.Set[float64]":  "[]float64",
		"*github.com/curtisnewbie/miso/util/hash.Set[float64]": "[]float64",

		// Amt is middleware/money.Amt, type Amt string
		"github.com/curtisnewbie/miso/middleware/money.Amt":  "string",
		"*github.com/curtisnewbie/miso/middleware/money.Amt": "string",
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

		// misoapi now generate doc statically
		//
		// err := writeApiDocFile(rail, docs.routeDocs, pipelineDoc)
		// if err != nil {
		// 	rail.Errorf("Failed to write api-doc markdown file: %v", err)
		// 	return err
		// }

		// TODO: deprecate, migrate to misoapi?
		err := writeApiDocOpenApiSpec(rail, docs)
		if err != nil {
			rail.Errorf("Failed to write api-doc open-api 3.0 spec file: %v", err)
			return err
		}

		// TODO: deprecate, migrate to misoapi?
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

type HttpRouteDoc struct {
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
	IsSliceOrArray        bool        // slice or array []T
	IsSliceOfPointer      bool        // slice of pointer []*T
	IsMap                 bool        // map
	IsPointer             bool        // *T
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
	if f.IsMap {
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
	return f.IsMap
}

func (f FieldDesc) pureGoTypeName() string {
	n := f.OriginTypeName
	if f.IsMap {
		return n
	}
	return PureGoTypeName(n)
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
	if f.IsSliceOfPointer {
		return "[]*" + ptn
	}
	if f.IsSliceOrArray {
		return "[]" + ptn
	}
	if f.IsPointer {
		return "*" + ptn
	}
	return f.OriginTypeName
}

// TODO: Support map as TypeDesc

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
	if f.IsSliceOrArray {
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
	return PureGoTypeName(n)
}

func (f TypeDesc) isBuiltInType() bool {
	return false
}

func PureGoTypeName(n string) string {
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

type HttpRouteDocs struct {
	routeDocs []HttpRouteDoc
	globalDoc globalHttpRouteDoc
	openapi   *openapi3.T
}

func buildHttpRouteDoc(hr []HttpRoute) HttpRouteDocs {
	docs := make([]HttpRouteDoc, 0, len(hr))
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

		d := HttpRouteDoc{
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

			d.JsonReqTsDef = GenTsDef(d.JsonRequestDesc)
			d.JsonReqGoDef, d.JsonReqGoDefTypeName = GenGoDef(d.JsonRequestDesc, hash.NewSet[string]())

			if addGlobalGoTypeDef {
				td, _ := GenGoDef(d.JsonRequestDesc, seenGlobalGoTypeDef)
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

			d.JsonRespTsDef = GenTsDef(d.JsonResponseDesc)
			d.JsonRespGoDef, d.JsonRespGoDefTypeName = GenGoDef(d.JsonResponseDesc, hash.NewSet[string]())

			if addGlobalGoTypeDef {
				td, _ := GenGoDef(d.JsonResponseDesc, seenGlobalGoTypeDef)
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
		d.Curl = GenRouteCurl(d, GetPropStr(PropServerPort))

		// ng http client
		d.NgHttpClientDemo = GenNgHttpClientDemo(d, GetPropStr(PropAppName), GetPropBool(PropServerGenerateEndpointDocInclPrefix))

		// ng table demo
		if _, ok := r.Extra[ExtraNgTable]; ok {
			d.NgTableDemo = GenNgTableDemo(d)
		}

		// miso http TClient
		d.MisoTClientDemo = GenTClientDemo(d, GetPropStr(PropAppName))
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
			d.OpenApiDoc = GenOpenApiDoc(d, rootSpec)
		} else {
			d.OpenApiDoc = GenOpenApiDoc(d, nil)
		}

		docs = append(docs, d)
	}
	return HttpRouteDocs{
		routeDocs: docs,
		globalDoc: globalHttpRouteDoc{
			GoTypeDef: globalGoTypeDef,
		},
		openapi: rootSpec,
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

		if !f.IsExported() {
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
						jd.IsSliceOrArray = true
						if et.Elem().Kind() == reflect.Pointer {
							jd.IsSliceOfPointer = true
						}
					case reflect.Map:
						jd.IsMap = true
					case reflect.Pointer:
						jd.IsPointer = true
					}
					if !seen.Has(et) {
						jd.Fields = reflectAppendFieldDesc(et, ele, jd.Fields, seen)
					}
				}
			} else {
				appendable = false // e.g., the any field in GnResp[any]{}
				Tracef("reflect.Value is zero or nil, not displayed in api doc, type: %v, field: %v", t.Name(), jd.JsonName)
			}
		} else {
			switch fv.Kind() {
			case reflect.Slice, reflect.Array:
				jd.IsSliceOrArray = true
				if fv.Type().Elem().Kind() == reflect.Pointer {
					jd.IsSliceOfPointer = true
				}
			case reflect.Map:
				jd.IsMap = true
			case reflect.Pointer:
				jd.IsPointer = true
			}
			if !seen.Has(f.Type) {
				jd.Fields = reflectAppendFieldDesc(f.Type, fv, jd.Fields, seen)
			}
		}

		if appendable {
			jds = append(jds, jd)
		}
	}
	return jds
}

func reflectAppendFieldDesc(t reflect.Type, v reflect.Value, fields []FieldDesc, seen *hash.Set[reflect.Type]) []FieldDesc {
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
				fields = reflectAppendFieldDesc(et, ele, fields, seen)
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
			markdown := GenMarkDownDoc(docs.routeDocs, pipelineDoc)

			w, _ := inb.Unwrap()
			if err := apiDocTmpl.ExecuteTemplate(w, "apiDocTempl",
				struct {
					App         string
					HttpDoc     []HttpRouteDoc
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

func writeApiDocOpenApiSpec(rail Rail, docs HttpRouteDocs) error {
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

func writeApiDocFile(rail Rail, routes []HttpRouteDoc, pipelineDoc []PipelineDoc) error {
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

		markdown := GenMarkDownDoc(routes, pipelineDoc)
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

func writeApiDocGoFile(rail Rail, goTypeDefs []string, routes []HttpRouteDoc) error {
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
