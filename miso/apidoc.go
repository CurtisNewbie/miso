package miso

import (
	"fmt"
	"html/template"
	"reflect"
	"strings"
	"sync"
	"time"
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
	ignoredJsonDocTag   = []string{"form", "header"}
)

func init() {
	SetDefProp(PropServerGenerateEndpointDocEnabled, true)
}

type ParamDoc struct {
	Name string
	Desc string
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
		if r.JsonRequestVal != nil {
			d.JsonRequestDesc = buildJsonDesc(reflect.ValueOf(r.JsonRequestVal))
		}
		if r.JsonResponseVal != nil {
			d.JsonResponseDesc = buildJsonDesc(reflect.ValueOf(r.JsonResponseVal))
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
			b.WriteRune('\n')
			b.WriteString(Spaces(2))
			b.WriteString("- Header Parameter:")
			for _, h := range r.Headers {
				b.WriteRune('\n')
				b.WriteString(Spaces(4))
				b.WriteString("- \"")
				b.WriteString(h.Name)
				b.WriteString("\": ")
				b.WriteString(h.Desc)
			}
		}
		if len(r.QueryParams) > 0 {
			b.WriteRune('\n')
			b.WriteString(Spaces(2))
			b.WriteString("- Query Parameter:")
			for _, q := range r.QueryParams {
				b.WriteRune('\n')
				b.WriteString(Spaces(4))
				b.WriteString("- \"")
				b.WriteString(q.Name)
				b.WriteString("\": ")
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

func buildJsonDesc(v reflect.Value) []jsonDesc {
	t := v.Type()
	jds := make([]jsonDesc, 0, 5)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if IsVoid(f.Type) {
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
			name = LowercaseNamingStrategy(f.Name)
		}

		typeName := TypeName(f.Type)
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
					jd.TypeName = TypeName(et)
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

func reflectAppendJsonDesc(t reflect.Type, v reflect.Value, fields []jsonDesc) []jsonDesc {
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
				<li>'{{.Name}}': <i>({{.TypeName}})</i> {{.Desc}}
					{{if .Fields}}
						{{template "unpackJsonDesc" .Fields}}
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
							<div style="text-indent:8px;border-left: 4px solid #757575;">
								<b><i>Description:</i></b>
							</div>
						</p>
						<p>&nbsp;&nbsp;&nbsp;&nbsp;{{.Desc}}</p>
					{{end}}

					{{if .Scope }}
						<p>
							<div style="text-indent:8px;border-left: 4px solid #757575;">
								<b><i>Expected Access Scope:</i></b>
							</div>
						</p>
						<p>&nbsp;&nbsp;&nbsp;&nbsp;{{.Scope}}</p>
					{{end}}

					{{if .Headers}}
						<p>
							<div style="text-indent:8px;border-left: 4px solid #757575;">
								<b><i>Header Parameters:</i></b>
							</div>
							<ul>
							{{range .Headers}}
								<li>'{{.Name}}': {{.Desc}}</li>
							{{end}}
							</ul>
						</p>
					{{end}}

					{{if .QueryParams}}
						<p>
							<div style="text-indent:8px;border-left: 4px solid #757575;">
								<b><i>Query Parameters:</i></b>
							</div>
							<ul>
								{{range .QueryParams}}
									<li>'{{.Name}}': {{.Desc}}</li>
								{{end}}
							</ul>
						</p>
					{{end}}

					{{if .JsonRequestDesc}}
						<p>
							<div style="text-indent:8px;border-left: 4px solid #757575;">
								<b><i>JSON Request:</i></b>
							</div>
							{{template "unpackJsonDesc" .JsonRequestDesc}}
						</p>
					{{end}}

					{{if .JsonResponseDesc}}
						<p>
							<div style="text-indent:8px;border-left: 4px solid #757575;">
								<b><i>JSON Response:</i></b>
							</div>
							{{template "unpackJsonDesc" .JsonResponseDesc}}
						</p>
					{{end}}
					</div>
				{{end}}

				<h2>2. Markdown API Doc:</h2>

				<div style="text-align: right;">
					<button style="padding:10px; box-shadow: 3px 3px 10px lightgrey;" onclick="copyMarkdown()">Copy Generated Markdown</button>
				</div>
				<pre id="markdownPre" style="white-space: pre-wrap; background-color:DBD5D4; padding:30px; border-radius: 30px;">{{.Markdown}}</pre>

				<script>
					function copyMarkdown() {
						var preContent = document.getElementById("markdownPre").textContent;
						const textArea = document.createElement('textarea');
						textArea.textContent = preContent;
						document.body.append(textArea);
						textArea.select();
						document.execCommand("copy");
						textArea.remove();
					}
				</script>
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
		func(inb *Inbound) {
			defer DebugTimeOp(rail, time.Now(), "gen api doc")

			routeDoc := buildHttpRouteDoc(rail, GetHttpRoutes())
			markdown := genMarkDownDoc(routeDoc)

			w, _ := inb.Unwrap()
			if err := apiDocTmpl.ExecuteTemplate(w, "apiDocTempl",
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

	rail.Infof("Exposing API Documentation on http://localhost:%v/doc/api", GetPropInt(PropServerPort))
	return nil
}

func parseQueryDoc(t reflect.Type) []ParamDoc {
	if t == nil {
		return nil
	}
	pds := []ParamDoc{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if IsVoid(f.Type) {
			continue
		}

		query := f.Tag.Get(TagQueryParam)
		if query == "" {
			continue
		}
		desc := f.Tag.Get(TagApiDocDesc)
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
	pds := []ParamDoc{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if IsVoid(f.Type) {
			continue
		}

		header := f.Tag.Get(TagHeaderParam)
		if header == "" {
			continue
		}
		header = strings.ToLower(header)
		desc := f.Tag.Get(TagApiDocDesc)
		pds = append(pds, ParamDoc{
			Name: header,
			Desc: desc,
		})
	}
	return pds
}
