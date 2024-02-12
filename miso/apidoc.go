package miso

import (
	"fmt"
	"reflect"
	"strings"
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
)

func genEndpointDoc(rail Rail) {
	b := strings.Builder{}
	b.WriteString("# API Endpoints\n")

	hr := GetHttpRoutes()
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
		// if r.Scope != "" {
		// 	b.WriteRune('\n')
		// 	b.WriteString(Spaces(2))
		// 	b.WriteString("- Access Scope: ")
		// 	b.WriteString(r.Scope)
		// }
		// if r.Resource != "" {
		// 	b.WriteRune('\n')
		// 	b.WriteString(Spaces(2))
		// 	b.WriteString("- Resource: \"")
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
		if r.JsonRequestType != nil {
			b.WriteRune('\n')
			b.WriteString(Spaces(2))
			b.WriteString("- JSON Request: ")
			buildJsonPayloadDoc(&b, *r.JsonRequestType, 2)
		}
		if r.JsonResponseType != nil {
			b.WriteRune('\n')
			b.WriteString(Spaces(2))
			b.WriteString("- JSON Response: ")
			buildJsonPayloadDoc(&b, *r.JsonResponseType, 2)
		}
	}
	rail.Infof("Generated API Endpoints Documentation:\n\n%s\n", b.String())
}

func buildJsonPayloadDoc(b *strings.Builder, t reflect.Type, indent int) {
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
		b.WriteString(fmt.Sprintf("\n%s- \"%s\": (%s) %s", Spaces(indent+2), name, typeName, f.Tag.Get(TagApiDocDesc)))

		if !typeAliasMatched {
			if f.Type.Kind() == reflect.Struct {
				buildJsonPayloadDoc(b, f.Type, indent+2)
			} else if f.Type.Kind() == reflect.Slice {
				et := f.Type.Elem()
				if et.Kind() == reflect.Struct {
					buildJsonPayloadDoc(b, et, indent+2)
				}
			}
		}
	}
}
