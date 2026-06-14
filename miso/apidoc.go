package miso

import (
	_ "embed"
	"fmt"
	"reflect"
	"regexp"
	"strings"

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

	apiDocJsoniterConfig = jsoniter.Config{
		EscapeHTML:  true,
		SortMapKeys: true,
	}.Froze()
)

type ApiDocFuzzType struct {
	PkgPath  string
	TypeName string
}

type globalHttpRouteDoc struct {
	GoTypeDef []string
}

type HttpRouteDoc struct {
	Name                    string           // api func name
	SourceFile              string           // source file where the endpoint is registered
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
	JavaClientDemo          string           // java http client demo
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
			case ValidNotEmpty, ValidNotNil:
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

type PipelineDoc struct {
	Name        string
	Desc        string
	Exchange    string
	RoutingKey  string
	Queue       string
	PayloadDesc TypeDesc
}
