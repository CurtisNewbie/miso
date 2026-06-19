package miso

import (
	"github.com/curtisnewbie/miso/util/json"
	"github.com/getkin/kin-openapi/openapi3"
)

func GenOpenApiDoc(d HttpRouteDoc, root *openapi3.T, server string) string {
	title := d.Desc
	if title == "" {
		title = d.Method + " " + d.Url
	}

	servers := openapi3.Servers{}
	if server != "" {
		servers = openapi3.Servers{
			&openapi3.Server{URL: server},
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
