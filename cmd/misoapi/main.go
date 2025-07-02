package main

import (
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/version"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
)

const (
	MisoApiPrefix = "misoapi-"

	typeMisoInboundPtr = "*miso.Inbound"
	typeMisoRail       = "miso.Rail"
	typeGormDbPtr      = "*gorm.DB"
	typeMySqlQryPtr    = "*mysql.Query"
	typeCommonUser     = "common.User"

	importCommonUser = "github.com/curtisnewbie/miso/middleware/user-vault/common"
	importMiso       = "github.com/curtisnewbie/miso/miso"
	importGorm       = "gorm.io/gorm"
	importMySQL      = "github.com/curtisnewbie/miso/middleware/mysql"
	importDbQuery    = "github.com/curtisnewbie/miso/middleware/dbquery"
)

const (
	tagHttp         = "http"
	tagDesc         = "desc"
	tagScope        = "scope"
	tagRes          = "resource"
	tagQueryDocV1   = "query-doc"
	tagHeaderDocV1  = "header-doc"
	tagQueryDocV2   = "query"
	tagHeaderDocV2  = "header"
	tagJsonRespType = "json-resp-type"
)

const (
	tagNgTable = "ngtable"
	tagRaw     = "raw"
	tagIgnore  = "ignore"
)

var (
	refPat              = regexp.MustCompile(`ref\(([a-zA-Z0-9 \\-\\_\.]+)\)`)
	flagTags            = util.NewSet[string](tagNgTable, tagRaw, tagIgnore)
	injectTokenToImport = map[string]string{
		typeCommonUser:     importCommonUser,
		typeMySqlQryPtr:    importMySQL,
		typeGormDbPtr:      importDbQuery,
		typeMisoInboundPtr: "",
		typeMisoRail:       "",
	}
	injectToken = map[string]string{
		typeMisoInboundPtr: "inb",
		typeMisoRail:       "inb.Rail()",
		typeMySqlQryPtr:    "mysql.NewQuery(dbquery.GetDB())",
		typeGormDbPtr:      "dbquery.GetDB()",
		typeCommonUser:     "common.GetUser(inb.Rail())",
	}
)

var (
	Debug = flag.Bool("debug", false, "Enable debug log")
)

func main() {
	flag.Usage = func() {
		util.Printlnf("\nmisoapi - automatically generate web endpoint in go based on misoapi-* comments\n")
		util.Printlnf("  Supported miso version: %v\n", version.Version)
		util.Printlnf("Usage of %s:", os.Args[0])
		flag.PrintDefaults()
		util.Printlnf("\nFor example:\n")
		util.Printlnf("  misoapi-http: GET /open/api/doc                                     // http method and url")
		util.Printlnf("  misoapi-desc: open api endpoint to retrieve documents               // description")
		util.Printlnf("  misoapi-query-doc: page: curent page index                          // query parameter")
		util.Printlnf("  misoapi-header-doc: Authorization: bearer authorization token       // header parameter")
		util.Printlnf("  misoapi-scope: PROTECTED                                            // access scope")
		util.Printlnf("  misoapi-resource: document:read                                     // resource code")
		util.Printlnf("  misoapi-ngtable                                                     // generate angular table code")
		util.Printlnf("  misoapi-raw                                                         // raw endpoint without auto request/response json handling")
		util.Printlnf("  misoapi-json-resp-type: MyResp                                      // json response type (struct), for raw api only")
		util.Printlnf("  misoapi-ignore                                                      // ignored by misoapi")
		util.Printlnf("")
	}
	flag.Parse()

	files, err := walkDir(".", ".go")
	if err != nil {
		util.Printlnf("[ERROR] walkDir failed, %v", err)
		return
	}
	if err := parseFiles(files); err != nil {
		util.Printlnf("[ERROR] parseFiles failed, %v", err)
	}
}

type GroupedApiDecl struct {
	Dir     string
	Pkg     string
	PkgPath string
	Apis    []ApiDecl
	Imports util.Set[string]
}

type FsFile struct {
	Path string
	File fs.FileInfo
}

func parseFiles(files []FsFile) error {
	dstFiles, err := parseFileAst(files)
	if err != nil {
		return err
	}

	if *Debug {
		for _, f := range dstFiles {
			util.Printlnf("[DEBUG] Found %v", f.Path)
		}
	}

	modName := ""
	{
		out, err := util.ExecCmd("go", []string{"list", "-m"})
		if err != nil {
			panic(fmt.Errorf("%s, %v", out, err))
		}
		modName = strings.TrimSpace(string(out))
	}

	pathApiDecls := make(map[string]GroupedApiDecl)
	addApiDecl := func(p string, pkg string, pkgPath string, d ApiDecl, imports util.Set[string]) {
		dir, _ := path.Split(p)
		v, ok := pathApiDecls[dir]
		if ok {
			v.Apis = append(v.Apis, d)
			v.Imports.AddAll(imports.CopyKeys())
			pathApiDecls[dir] = v
		} else {
			imp := util.NewSet[string]()
			imp.AddAll(imports.CopyKeys())
			pathApiDecls[dir] = GroupedApiDecl{
				Dir:     dir,
				Pkg:     pkg,
				PkgPath: pkgPath,
				Apis:    []ApiDecl{d},
				Imports: imp,
			}
		}
	}

	for _, df := range dstFiles {
		pkgPath := modName + "/" + path.Dir(path.Dir(path.Clean(df.Path))) + "/" + df.Dst.Name.Name
		importSepc := map[string]string{}
		dstutil.Apply(df.Dst,
			func(c *dstutil.Cursor) bool {
				// parse api declaration
				ad, imports, ok := parseApiDecl(c, df.Path, importSepc)
				if ok {
					addApiDecl(df.Path, df.Dst.Name.Name, pkgPath, ad, imports)
				}

				return true
			},
			func(cursor *dstutil.Cursor) bool {
				return true
			},
		)
	}

	baseIndent := 1
	for dir, v := range pathApiDecls {
		for _, ad := range v.Apis {
			if *Debug {
				util.Printlnf("[DEBUG] %v (%v) => %#v", dir, v.Pkg, ad)
			}
		}
		// check if package is imported in main
		{
			out, err := util.ExecCmd("grep", []string{"-r", v.PkgPath, "--include", "*.go"}, func(c *exec.Cmd) {
				if *Debug {
					util.DebugPrintlnf(*Debug, "cmd: %v", c)
				}
			})
			if err != nil {
				var extErr *exec.ExitError
				if errors.As(err, &extErr) && extErr.ExitCode() == 1 {
					util.Printlnf(util.ANSIRed+"Warning: (1) package '%v' is not imported!"+util.ANSIReset, v.PkgPath)
				} else {
					util.Printlnf("[ERROR] check package import failed, pkg: %v, out: %s, %v", v.PkgPath, out, err)
				}
			} else {
				if strings.TrimSpace(string(out)) == "" {
					util.Printlnf(util.ANSIRed+"Warning: (2) package '%v' is not imported!"+util.ANSIReset, v.PkgPath)
				}
			}
		}

		imports, code, err := genGoApiRegister(v.Apis, baseIndent, v.Imports)
		if err != nil {
			util.Printlnf("[ERROR] generate code failed, %v", err)
			continue
		}

		importSb := strings.Builder{}
		importStrs := imports.CopyKeys()
		sort.Slice(importStrs, func(i, j int) bool { return strings.Compare(importStrs[i], importStrs[j]) < 0 })
		for _, s := range importStrs {
			if importSb.Len() > 0 {
				importSb.WriteString("\n")
			}
			importSb.WriteString(strings.Repeat("\t", baseIndent) + "\"" + s + "\"")
		}

		out := util.NamedSprintf(`// auto generated by misoapi ${misoVersion} at ${nowTimeStr}, please do not modify
package ${package}

import (
${importStr}
)

func init() {
${code}
}
`, map[string]any{
			"misoVersion": version.Version,
			"nowTimeStr":  util.Now().FormatClassicLocale(),
			"package":     v.Pkg,
			"code":        code,
			"importStr":   importSb.String(),
		})

		if *Debug {
			util.Printlnf("[DEBUG] %v (%v) => \n\n%v", dir, v.Pkg, out)
		}
		outFile := fmt.Sprintf("%vmisoapi_generated.go", dir)

		prev, err := os.ReadFile(outFile)
		if err == nil {
			prevs := util.UnsafeByt2Str(prev)
			if i := strings.Index(prevs, "\n"); i > -1 && i+1 < len(prevs) {
				prevs = prevs[i+1:]
			}
			outBody := out[strings.Index(out, "\n")+1:]
			if prevs == outBody {
				util.DebugPrintlnf(*Debug, "Generated code remain the same, skipping %v", outFile)
				continue
			}
		}

		f, err := util.ReadWriteFile(outFile)
		util.Must(err)
		util.Must(f.Truncate(0))
		_, err = f.WriteString(out)
		util.Must(err)
		f.Close()
		util.Printlnf("Generated code written to %v, using pkg: %v, api count: %d", outFile, v.Pkg, len(v.Apis))
	}

	return nil
}

type DstFile struct {
	Dst  *dst.File
	Path string
}

func parseApiDecl(cursor *dstutil.Cursor, srcPath string, importSpec map[string]string) (ApiDecl, util.Set[string], bool) {
	switch n := cursor.Node().(type) {
	case *dst.ImportSpec:
		alias := ""
		if n.Name != nil {
			alias = n.Name.String()
		}
		importPath := n.Path.Value
		if len(importPath) > 1 && importPath[:1] == "\"" && importPath[len(importPath)-1:] == "\"" {
			importPath = importPath[1 : len(importPath)-1]
		}
		if alias == "" {
			alias = path.Base(importPath)
		}
		importSpec[alias] = importPath
		if *Debug {
			util.Printlnf("[DEBUG] parseApiDecl() alias: %v, importPath: %v", alias, importPath)
		}
	case *dst.FuncDecl:
		imports := util.NewSet[string]()
		tags, ok := parseMisoApiTag(srcPath, n.Decs.Start)
		if ok {
			if *Debug {
				util.Printlnf("[DEBUG] parseApiDecl() type results: %#v", n.Type.Results)
				util.Printlnf("[DEBUG] parseApiDecl() tags: %+v", tags)
			}
			for _, t := range tags {
				kv, ok := t.BodyKV()
				if *Debug {
					util.Printlnf("[DEBUG] parseApiDecl() tag -> %#v, kv: %#v, ok: %v", t, kv, ok)
				}
			}
			ad, ok := BuildApiDecl(tags)
			if ok {
				ad.FuncName = n.Name.String()
				ad.FuncParams = parseParamMeta(n.Type.Params, srcPath, ad.FuncName, importSpec, imports)
				ad.FuncResults = parseParamMeta(n.Type.Results, srcPath, ad.FuncName, importSpec, imports)
			}
			return ad, imports, ok
		}
	}
	return ApiDecl{}, util.Set[string]{}, false
}

func guessImport(n string, importSpec map[string]string, imports util.Set[string]) {
	if n == "" || importSpec == nil {
		return
	}
	cached, ok := importSpec[n]
	if ok {
		switch cached {
		case importCommonUser, importMiso, importGorm, importMySQL:
			return
		default:
			imports.Add(cached)
		}
	}
}

func parseParamMeta(l *dst.FieldList, path string, funcName string, importSpec map[string]string, imports util.Set[string]) []ParamMeta {
	if l == nil {
		return []ParamMeta{}
	}
	pm := make([]ParamMeta, 0)
	for i, p := range l.List {
		var varName string
		if len(p.Names) > 0 {
			varName = p.Names[0].String()
		}

		if *Debug {
			util.Printlnf("[DEBUG] parseParamMeta() func: %v, param [%v], p: %#v", funcName, i, p.Type)
		}

		typeName := parseParamName(p.Type, importSpec, imports)
		if typeName != "" {
			pm = append(pm, ParamMeta{Name: varName, Type: typeName})
		} else {
			util.Printlnf("[ERROR] failed to parse param[%d]: %v %#v, %v: %v", i, p.Names, p.Type, path, funcName)
		}
	}
	return pm
}

type ParamMeta struct {
	Name string
	Type string
}

type ApiDecl struct {
	Method   string
	Url      string
	Header   []Pair
	Query    []Pair
	Desc     string
	Scope    string
	Resource string

	FuncName    string
	FuncParams  []ParamMeta
	FuncResults []ParamMeta
	Flags       util.Set[string]

	JsonRespType string
	Imports      util.Set[string]
}

func (d ApiDecl) countExtraLines() int {
	n := 0
	if d.Desc != "" {
		n++
	}
	n += len(d.Header)
	n += len(d.Query)
	if d.FuncName != "" {
		n++
	}
	if d.Flags.Has(tagNgTable) {
		n++
	}
	if d.Scope != "" {
		n++
	}
	if d.Resource != "" {
		n++
	}
	return n
}

func (d ApiDecl) parseFuncParams() (reqType string) {
	for _, p := range d.FuncParams {
		if imp, ok := injectTokenToImport[p.Type]; ok {
			if imp != "" {
				d.Imports.Add(imp)
			}
			continue
		}
		if reqType == "" {
			reqType = p.Type
		}
	}
	return reqType
}

func (d ApiDecl) parseFuncResults() (resType string, errorOnly bool, noError bool) {
	noError = true
	errorOnly = len(d.FuncResults) == 1 && d.FuncResults[0].Type == "error"
	for _, p := range d.FuncResults {
		switch p.Type {
		case "error":
			noError = false
			continue
		default:
			if imp, ok := injectTokenToImport[p.Type]; ok {
				if imp != "" {
					d.Imports.Add(imp)
				}
			}
			if resType == "" {
				resType = p.Type
			}
		}
	}
	if resType == "" {
		resType = "any"
	}
	return resType, errorOnly, noError
}

func (d ApiDecl) allParamsInjectable() bool {
	for _, p := range d.FuncParams {
		if _, ok := injectToken[p.Type]; !ok {
			return false
		}
	}
	return true
}

func (d ApiDecl) guessInjectToken(typ string, extra ...func(typ string) (string, bool)) string {
	for _, ex := range extra {
		if v, ok := ex(typ); ok {
			return v
		}
	}
	if injected, ok := injectToken[typ]; ok {
		return injected
	}
	return typ + "{}" // something we don't know :(
}

func (d ApiDecl) injectFuncParams(extra ...func(typ string) (string, bool)) string {
	paramTokens := make([]string, 0, len(d.FuncParams))
	for _, p := range d.FuncParams {
		var v string = d.guessInjectToken(p.Type, extra...)
		paramTokens = append(paramTokens, v)
	}
	return strings.Join(paramTokens, ", ")
}

func (d ApiDecl) printInvokeFunc(extra ...func(typ string) (string, bool)) string {
	params := d.injectFuncParams(extra...)
	return fmt.Sprintf("%v(%v)", d.FuncName, params)
}

func genGoApiRegister(dec []ApiDecl, baseIndent int, imports util.Set[string]) (util.Set[string], string, error) {
	w := util.NewIndentWriter("\t")
	w.SetIndent(baseIndent)
	imports.Add(importMiso)

	for i, d := range dec {

		if d.Flags.Has(tagIgnore) {
			continue
		}

		custReqType := d.parseFuncParams()
		custResType, errorOnly, noError := d.parseFuncResults()
		imports.AddAll(d.Imports.CopyKeys())
		extraLines := d.countExtraLines()

		httpMethod := d.Method[:1] + strings.ToLower(d.Method[1:])
		if custReqType != "" {
			if d.Flags.Has(tagRaw) {
				w.Writef("miso.Http%v(\"%v\", miso.RawHandler(", httpMethod, d.Url)
				w.IncrIndent()
				w.Writef("func(inb *miso.Inbound) {")
				w.StepIn(func(w *util.IndentWriter) {
					invokeFunc := d.printInvokeFunc(func(typ string) (string, bool) {
						if typ == custReqType {
							return "req", true
						}
						return "", false
					})
					w.Writef("var req %v", custReqType)
					w.Writef("inb.MustBind(&req)")
					w.Writef(invokeFunc)
				})
				w.Writef("})).")
				if d.JsonRespType != "" {
					w.Writef("DocJsonReq(%v{}).", custReqType)
					w.NoLbWritef("DocJsonResp(%v{})", d.JsonRespType)
				} else {
					w.NoLbWritef("DocJsonReq(%v{})", custReqType)
				}
			} else {
				w.Writef("miso.Http%v(\"%v\", miso.AutoHandler(", httpMethod, d.Url)
				w.IncrIndent()
				w.Writef("func(inb *miso.Inbound, req %v) (%v, error) {", custReqType, custResType)
				w.StepIn(func(w *util.IndentWriter) {
					invokeFunc := d.printInvokeFunc(func(typ string) (string, bool) {
						if typ == custReqType {
							return "req", true
						}
						return "", false
					})
					if errorOnly {
						w.Writef("return nil, %v", invokeFunc)
					} else if len(d.FuncResults) < 1 {
						w.Writef(invokeFunc)
						w.Writef("return nil, nil")
					} else if noError {
						w.Writef("return %v, nil", invokeFunc)
					} else {
						w.Writef("return %v", invokeFunc)
					}
				})
				w.NoLbWritef("}))")
			}
		} else {
			isRaw := d.Flags.Has(tagRaw) && len(d.FuncResults) < 1 && d.allParamsInjectable()
			if isRaw {
				if len(d.FuncParams) == 1 && d.FuncParams[0].Type == typeMisoInboundPtr {
					if d.JsonRespType != "" {
						w.Writef("miso.Http%v(\"%v\", miso.RawHandler(%v)).", httpMethod, d.Url, d.FuncName)
						w.StepIn(func(iw *util.IndentWriter) {
							iw.NoLbWritef("DocJsonResp(%v{})", d.JsonRespType)
						})
					} else {
						w.NoLbWritef("miso.Http%v(\"%v\", miso.RawHandler(%v))", httpMethod, d.Url, d.FuncName)
					}
					if extraLines > 0 {
						w.IncrIndent()
					}
				} else {
					w.Writef("miso.Http%v(\"%v\", miso.RawHandler(", httpMethod, d.Url)
					w.IncrIndent()
					w.Writef("func(inb *miso.Inbound) {")
					w.StepIn(func(w *util.IndentWriter) {
						w.Writef(d.printInvokeFunc())
					})
					if d.JsonRespType != "" {
						w.Writef("})).")
						w.NoLbWritef("DocJsonResp(%v{})", d.JsonRespType)
					} else {
						w.NoLbWritef("}))")
					}
				}

			} else {
				w.Writef("miso.Http%v(\"%v\", miso.ResHandler(", httpMethod, d.Url)
				w.IncrIndent()
				w.Writef("func(inb *miso.Inbound) (%v, error) {", custResType)
				w.StepIn(func(w *util.IndentWriter) {
					invokeFunc := d.printInvokeFunc()
					if errorOnly {
						w.Writef("return nil, %v", invokeFunc)
					} else if len(d.FuncResults) < 1 {
						w.Writef(invokeFunc)
						w.Writef("return nil, nil")
					} else if noError {
						w.Writef("return %v, nil", invokeFunc)
					} else {
						w.Writef("return %v", invokeFunc)
					}
				})
				w.NoLbWritef("}))")
			}
		}

		if d.FuncName != "" {
			extraLines--
			w.NoIndWritef(".\n")
			w.NoLbWritefWhen(extraLines > 0, "Extra(miso.ExtraName, \"%s\")", d.FuncName)
		}

		if d.Flags.Has(tagNgTable) {
			extraLines--
			w.NoIndWritef(".\n")
			w.NoLbWritefWhen(extraLines > 0, "Extra(miso.ExtraNgTable, true)")
		}
		if d.Desc != "" {
			extraLines--
			w.NoIndWritef(".\n")
			w.NoLbWritefWhen(extraLines > 0, "Desc(`%v`)", d.Desc)
		}
		if d.Scope != "" {
			extraLines--
			w.NoIndWritef(".\n")
			var l string
			switch d.Scope {
			case "PROTECTED":
				l = "Protected()"
			case "PUBLIC":
				l = "Public()"
			default:
				l = fmt.Sprintf("Scope(\"%v\")", d.Scope)
			}
			w.NoLbWritefWhen(extraLines > 0, l)
		}
		if d.Resource != "" {
			extraLines--
			w.NoIndWritef(".\n")
			ref, isRef := parseRef(d.Resource)
			var res string
			if !isRef {
				res = "\"" + d.Resource + "\""
			} else {
				res = ref
			}
			w.NoLbWritefWhen(extraLines > 0, "Resource(%v)", res)
		}
		for _, dh := range d.Header {
			extraLines--
			w.NoIndWritef(".\n")
			w.NoLbWritefWhen(extraLines > 0, "DocHeader(\"%v\", \"%v\")", dh.K, dh.V)
		}
		for _, dh := range d.Query {
			extraLines--
			w.NoIndWritef(".\n")
			w.NoLbWritefWhen(extraLines > 0, "DocQueryParam(\"%v\", \"%v\")", dh.K, dh.V)
		}
		if i < len(dec)-1 {
			w.NoIndWritef("\n")
		}
		w.SetIndent(baseIndent)
	}
	return imports, w.String(), nil
}

func BuildApiDecl(tags []MisoApiTag) (ApiDecl, bool) {
	ad := ApiDecl{
		Flags:   util.NewSet[string](),
		Imports: util.NewSet[string](),
	}
	for _, t := range tags {
		switch t.Command {
		case tagHttp:
			lr := strings.SplitN(t.Body, " ", 2)
			if len(lr) < 2 {
				return ad, false
			}
			ad.Method = strings.ToUpper(strings.TrimSpace(lr[0]))
			ad.Url = strings.TrimSpace(lr[1])
		case tagDesc:
			ad.Desc = t.Body
		case tagScope:
			ad.Scope = t.Body
		case tagRes:
			ad.Resource = t.Body
		case tagQueryDocV1, tagQueryDocV2:
			kv, ok := t.BodyKV()
			if ok {
				ad.Query = append(ad.Query, kv)
			}
		case tagHeaderDocV1, tagHeaderDocV2:
			kv, ok := t.BodyKV()
			if ok {
				ad.Header = append(ad.Header, kv)
			}
		case tagJsonRespType:
			ad.JsonRespType = t.Body
		default:
			if flagTags.Has(t.Command) {
				ad.Flags.Add(t.Command)
			}
		}
	}
	return ad, !util.IsBlankStr(ad.Method) && !util.IsBlankStr(ad.Url)
}

type Pair struct {
	K string
	V string
}

type MisoApiTag struct {
	Command string
	Body    string
}

func (m *MisoApiTag) BodyKV() (Pair, bool) {
	return m.BodyKVTok(":")
}

func (m *MisoApiTag) BodyKVTok(tok string) (Pair, bool) {
	i := strings.Index(m.Body, tok)
	if i < 0 {
		return Pair{K: m.Body}, false
	}
	return Pair{
		K: strings.TrimSpace(m.Body[:i]),
		V: strings.TrimSpace(m.Body[i+1:]),
	}, true
}

func parseMisoApiTag(path string, start dst.Decorations) ([]MisoApiTag, bool) {
	t := []MisoApiTag{}
	currIsDesc := false
	var descTmp string
	for _, s := range start {
		s = strings.TrimSpace(s)
		s, _ = strings.CutPrefix(s, "//")
		s = strings.TrimSpace(s)
		s, _ = strings.CutPrefix(s, "-")
		s = strings.TrimSpace(s)
		if m, ok := strings.CutPrefix(s, MisoApiPrefix); ok { // e.g., misoapi-http
			if pi := strings.Index(m, ":"); pi > -1 { // e.g., "misoapi-http: POST /api/doc"
				pre := m[:pi]
				m = m[pi+1:]
				if *Debug {
					util.Printlnf("[DEBUG] parseMisoApiTag() %v -> %v, command: %v, body: %v", path, s, pre, m)
				}
				pre = strings.TrimSpace(pre)
				currIsDesc = pre == tagDesc
				t = append(t, MisoApiTag{
					Command: pre,
					Body:    strings.TrimSpace(m),
				})
			} else { // e.g., "misoapi-ngtable"
				currIsDesc = false
				trimmed := strings.TrimSpace(m)
				t = append(t, MisoApiTag{
					Command: trimmed,
					Body:    trimmed,
				})
				continue
			}
		} else { // not related to misoapi.
			if s == "" {
				currIsDesc = false
				continue
			}

			if len(t) < 1 {
				descTmp += s
				continue
			}

			// multi-lines misoapi-desc
			if currIsDesc && len(t) > 0 && t[len(t)-1].Command == tagDesc {
				last := t[len(t)-1]
				s, cut := strings.CutPrefix(s, "\\")
				if cut {
					s = strings.TrimSpace(s)
				}

				last.Body += " " + s
				t[len(t)-1] = last
			}
		}
	}
	if descTmp != "" {
		anyDesc := false
		for _, v := range t {
			if v.Command == tagDesc {
				anyDesc = true
			}
		}
		if !anyDesc {
			t = append(t, MisoApiTag{Command: tagDesc, Body: strings.TrimSpace(descTmp)})
		}
	}

	return t, len(t) > 0
}

func parseFileAst(files []FsFile) ([]DstFile, error) {
	parsed := make([]DstFile, 0)
	for _, f := range files {
		p := f.Path
		if path.Base(p) == "misoapi_generated.go" {
			continue
		}
		d, err := decorator.ParseFile(nil, p, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, DstFile{
			Dst:  d,
			Path: p,
		})
	}
	return parsed, nil
}

func walkDir(n string, suffix string) ([]FsFile, error) {
	entries, err := os.ReadDir(n)
	if err != nil {
		return nil, miso.WrapErr(err)
	}
	files := make([]FsFile, 0, len(entries))
	for _, et := range entries {
		fi, err := et.Info()
		if err != nil {
			util.Printlnf("[ERROR] %v", err)
			continue
		}
		p := n + "/" + fi.Name()
		if et.IsDir() {
			ff, err := walkDir(p, suffix)
			if err == nil {
				files = append(files, ff...)
			}
		} else {
			if strings.HasSuffix(fi.Name(), suffix) {
				files = append(files, FsFile{File: fi, Path: p})
			}
		}
	}
	return files, nil
}

func parseParamName(t dst.Expr, importSpec map[string]string, imports util.Set[string]) string {
	switch v := t.(type) {
	case *dst.Ident:
		p := v.Path
		if p != "" {
			guessImport(p, importSpec, imports)
			p += "." + v.Name
		} else {
			p = v.Name
		}
		return p
	case *dst.SelectorExpr:
		n := parseParamName(v.X, importSpec, imports)
		guessImport(n, importSpec, imports)
		sn := v.Sel.String()
		if sn != "" {
			n += "." + sn
		}
		return n
	case *dst.StarExpr:
		return "*" + parseParamName(v.X, importSpec, imports)
	case *dst.ArrayType:
		return "[]" + parseParamName(v.Elt, importSpec, imports)
	case *dst.IndexExpr:
		n := parseParamName(v.X, importSpec, imports)
		return n + "[" + parseParamName(v.Index, importSpec, imports) + "]"
	default:
		return ""
	}
}

func parseRef(r string) (string, bool) {
	s := refPat.FindStringSubmatch(r)
	if s == nil {
		return "", false
	}
	return strings.TrimSpace(s[1]), true
}
