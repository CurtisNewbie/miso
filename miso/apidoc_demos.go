package miso

import (
	"fmt"
	"strings"

	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/strutil"
)

func GenNgTableDemo(d HttpRouteDoc) string {
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
						if f.OriginTypeName == "Time" || f.OriginTypeName == "*Time" || f.OriginTypeName == "atom.Time" || f.OriginTypeName == "*atom.Time" || f.TypeNameAlias == "int64" {
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

func GenNgHttpClientDemo(d HttpRouteDoc, appName string, inclPrefix bool) string {
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
			cleanName := PureGoTypeName(reqTypeName)
			if len(cleanName) > 0 {
				mn = fmt.Sprintf("send%s%s", strings.ToUpper(string(cleanName[0])), string(cleanName[1:]))
			}
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
	if inclPrefix {
		app := appName
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
	if reqTypeName != "" && (d.Method == "POST" || d.Method == "PUT") {
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

func GenTClientDemo(d HttpRouteDoc, appName string) (code string) {
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

					respGeneName = guessGoTypName(PureGoTypeName(n.TypeNameAlias))
					if n.isMisoPkg() && !n.isMisoDemoPkg() {
						respGeneName = "miso." + respGeneName
						if v := guessGoGenericEleName(n.TypeNameAlias); v != "" {
							respGeneName += "[" + v + "]"
						}
					}
					isPtr := n.IsPointer
					isSlice := n.IsSliceOrArray
					if n.IsSliceOfPointer {
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
			cleanName := PureGoTypeName(reqTypeName)
			if len(cleanName) > 0 {
				mn = fmt.Sprintf("Send%s%s", strings.ToUpper(string(cleanName[0])), string(cleanName[1:]))
			}
		}
	}

	{
		desc := strings.TrimSpace(d.Desc)
		if desc != "" {
			sl.Println(strutil.SAddLineIndent(desc, "// "))
		}
	}
	shouldIncludeReq := reqTypeName != "" && (d.Method == "POST" || d.Method == "PUT")
	if shouldIncludeReq {
		reqn := buildTypeName(reqTypeName, d.JsonRequestDesc.IsPtrSlice, d.JsonRequestDesc.IsSlicePtr,
			d.JsonRequestDesc.IsSlice, d.JsonRequestDesc.IsPtr)

		if respGeneName == "any" || respGeneName == "interface{}" {
			sl.Printlnf("func %s(rail miso.Rail, req %s%s) error {", mn, reqn, qh)
		} else {
			sl.Printlnf("func %s(rail miso.Rail, req %s%s) (%s, error) {", mn, reqn, qh, respGeneName)
		}
	} else {
		if respGeneName == "any" || respGeneName == "interface{}" {
			sl.Printlnf("func %s(rail miso.Rail%s) error {", mn, qh)
		} else {
			sl.Printlnf("func %s(rail miso.Rail%s) (%s, error) {", mn, qh, respGeneName)
		}
	}

	sl.LinePrefix = "\t"
	sl.Printlnf("var res miso.GnResp[%s]", respGeneName)
	sl.Printf("\n%serr := miso.NewDynClient(rail, \"%s\", \"%s\")", strutil.Tabs(1), d.Url, appName)

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
	if shouldIncludeReq {
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
	if respGeneName == "any" || respGeneName == "interface{}" {
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

	if respGeneName == "any" || respGeneName == "interface{}" {
		sl.Printlnf("return nil")
		sl.Printf("\n}")
		return sl.String()
	}

	sl.Printlnf("return res.Data, nil")
	sl.Printf("\n}")
	return sl.String()
}
