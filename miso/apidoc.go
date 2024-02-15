package miso

import (
	"fmt"
	"html/template"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	TagApiDocDesc = "desc"
)

var (
	ApiDocTypeAlias = map[string]string{
		"ETime":       "int64",
		"*ETime":      "int64",
		"*miso.ETime": "int64",
	}
	apiDocTmpl          *template.Template
	buildApiDocTmplOnce sync.Once
)

func init() {
	SetDefProp(PropServerGenerateEndpointDocEnabled, true)
}

type HttpRouteDoc struct {
	Url              string
	Method           string
	Extra            map[string][]any
	Desc             string     // description of the route (metadata).
	Scope            string     // the documented access scope of the route, it maybe "PUBLIC" or something else (metadata).
	Resource         string     // the documented resource that the route should be bound to (metadata).
	Headers          []ParamDoc // the documented header parameters that will be used by the endpoint (metadata).
	QueryParams      []ParamDoc // the documented query parameters that will used by the endpoint (metadata).
	JsonRequestDesc  []jsonDesc // the documented json request type that is expected by the endpoint (metadata).
	JsonResponseDesc []jsonDesc // the documented json response type that will be returned by the endpoint (metadata).
}

func buildHttpRouteDoc(rail Rail, hr []HttpRoute) []HttpRouteDoc {
	docs := make([]HttpRouteDoc, 0, len(hr))

	for _, r := range hr {
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
		if r.JsonRequestType != nil {
			d.JsonRequestDesc = buildJsonDesc(*r.JsonRequestType)
		}
		if r.JsonResponseType != nil {
			d.JsonResponseDesc = buildJsonDesc(*r.JsonResponseType)
		}
		docs = append(docs, d)
	}
	return docs
}

func genMarkDownDoc(hr []HttpRouteDoc) string {
	b := strings.Builder{}
	b.WriteString("# API Endpoints\n")

	for _, r := range hr {
		b.WriteString("\n- ")
		b.WriteString(r.Method)
		b.WriteString(" ")
		b.WriteString(r.Url)
		if r.Desc != "" {
			b.WriteRune('\n')
			b.WriteString(Spaces(2))
			b.WriteString("- Description: ")
			b.WriteString(r.Desc)
		}
		if r.Scope != "" {
			b.WriteRune('\n')
			b.WriteString(Spaces(2))
			b.WriteString("- Expected Access Scope: ")
			b.WriteString(r.Scope)
		}
		// if r.Resource != "" {
		// 	b.WriteRune('\n')
		// 	b.WriteString(Spaces(2))
		// 	b.WriteString("- Bound to Resource: \"")
		// 	b.WriteString(r.Resource)
		// 	b.WriteRune('"')
		// }
		if len(r.Headers) > 0 {
			for _, h := range r.Headers {
				b.WriteRune('\n')
				b.WriteString(Spaces(2))
				b.WriteString("- Header Parameter: \"")
				b.WriteString(h.Name)
				b.WriteString("\"\n")
				b.WriteString(Spaces(4))
				b.WriteString("- Description: ")
				b.WriteString(h.Desc)
			}
		}
		if len(r.QueryParams) > 0 {
			for _, q := range r.QueryParams {
				b.WriteRune('\n')
				b.WriteString(Spaces(2))
				b.WriteString("- Query Parameter: \"")
				b.WriteString(q.Name)
				b.WriteString("\"\n")
				b.WriteString(Spaces(4))
				b.WriteString("- Description: ")
				b.WriteString(q.Desc)
			}
		}
		if len(r.JsonRequestDesc) > 0 {
			b.WriteRune('\n')
			b.WriteString(Spaces(2))
			b.WriteString("- JSON Request:")
			appendJsonPayloadDoc(&b, r.JsonRequestDesc, 2)
		}
		if len(r.JsonResponseDesc) > 0 {
			b.WriteRune('\n')
			b.WriteString(Spaces(2))
			b.WriteString("- JSON Response:")
			appendJsonPayloadDoc(&b, r.JsonResponseDesc, 2)
		}
	}
	return b.String()
}

func appendJsonPayloadDoc(b *strings.Builder, jds []jsonDesc, indent int) {
	for _, jd := range jds {
		b.WriteString(fmt.Sprintf("\n%s- \"%s\": (%s) %s", Spaces(indent+2), jd.Name, jd.TypeName, jd.Desc))

		if len(jd.Fields) > 0 {
			appendJsonPayloadDoc(b, jd.Fields, indent+2)
		}
	}
}

func buildJsonDesc(t reflect.Type) []jsonDesc {
	jds := make([]jsonDesc, 0, 5)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if IsVoid(f.Type) {
			continue
		}
		var name string
		if v := f.Tag.Get("json"); v != "" {
			name = v
		} else {
			name = LowercaseNamingStrategy(f.Name)
		}

		var typeName string
		if f.Type.Name() != "" {
			typeName = f.Type.Name()
		} else {
			typeName = f.Type.String()
		}
		typeAlias, typeAliasMatched := ApiDocTypeAlias[typeName]
		if typeAliasMatched {
			typeName = typeAlias
		}

		jd := jsonDesc{
			Name:     name,
			TypeName: typeName,
			Desc:     f.Tag.Get(TagApiDocDesc),
			Fields:   []jsonDesc{},
		}

		if !typeAliasMatched {
			if f.Type.Kind() == reflect.Struct {
				jd.Fields = append(jd.Fields, buildJsonDesc(f.Type)...)
			} else if f.Type.Kind() == reflect.Slice {
				et := f.Type.Elem()
				if et.Kind() == reflect.Struct {
					jd.Fields = append(jd.Fields, buildJsonDesc(et)...)
				}
			}
		}
		jds = append(jds, jd)
	}
	return jds
}

type jsonDesc struct {
	Name     string
	TypeName string
	Desc     string
	Fields   []jsonDesc
}

func serveApiDocTmpl(rail Rail) error {
	var err error
	buildApiDocTmplOnce.Do(func() {
		t, er := template.New("").Parse(`
		{{define "unpackJsonDesc"}}
			<ul>
			{{range . }}
				<li>"{{.Name}}": <i>({{.TypeName}})</i> {{.Desc}}
					{{if .Fields}}
						{{range .Fields}}
							{{template "unpackJsonDesc" .Fields}}
						{{end}}
					{{end}}
				</li>
			{{end}}
			</ul>
		{{end}}

		{{define "apiDocTempl"}}
			<div style="margin:30px;">
				<h1>Generated {{.App}} API Documentation:</h1>
				<h2>1. HTML API DOC:</h2>
				{{range .Doc}}
					<div style="background-color:DBD5D4; margin-top:30px; margin-bottom:30px;
						padding-left:30px; padding-right:30px; padding-top:10px; padding-bottom:10px; border-radius: 20px;">
					<h3>{{.Method}} {{.Url}}</h3>
					{{if .Desc }}
						<p>
						<b><i>Description:</i></b> {{.Desc}}
						</p>
					{{end}}

					{{if .Scope }}
						<p>
						<b><i>Expected Access Scope:</i></b> {{.Scope}}
						</p>
					{{end}}

					{{if .Headers}}
						<p>
						<b><i>Header Parameters:</i></b>
						<ul>
						{{range .Headers}}
							<li>'{{.Name}}': {{.Desc}}</li>
						{{end}}
						</ul>
						</p>
					{{end}}

					{{if .QueryParams}}
						<p>
						<b><i>Query Parameters:</i></b>
						<ul>
						{{range .QueryParams}}
							<li>'{{.Name}}': {{.Desc}}</li>
						{{end}}
						</ul>
						</p>
					{{end}}

					{{if .JsonRequestDesc}}
						<p>
						<b><i>JSON Request:</i></b>
						{{template "unpackJsonDesc" .JsonRequestDesc}}
						</p>
					{{end}}

					{{if .JsonResponseDesc}}
						<p>
						<b><i>JSON Response:</i></b>
						{{template "unpackJsonDesc" .JsonResponseDesc}}
						</p>
					{{end}}
					</div>
				{{end}}

				<h2>2. Markdown API Doc:</h2>
				<pre style="white-space: pre-wrap; background-color:DBD5D4; padding:30px; border-radius: 30px;">{{.Markdown}}</pre>
			</div>
		{{end}}
		`)
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
		func(c *gin.Context, rail Rail) {
			routeDoc := buildHttpRouteDoc(rail, GetHttpRoutes())
			markdown := genMarkDownDoc(routeDoc)

			if err := apiDocTmpl.ExecuteTemplate(c.Writer, "apiDocTempl",
				struct {
					App      string
					Doc      []HttpRouteDoc
					Markdown string
				}{
					App:      GetPropStr(PropAppName),
					Doc:      routeDoc,
					Markdown: markdown,
				}); err != nil {
				rail.Errorf("failed to serve apiDocTmpl, %v", err)
			}
		}).
		Desc("Serve the generated API documentation webpage").
		Public()

	rail.Info("Exposing API Documentation on /doc/api")
	return nil
}
