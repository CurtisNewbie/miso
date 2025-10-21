package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/util/cli"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/flags"
	"github.com/curtisnewbie/miso/util/hash"
	"github.com/curtisnewbie/miso/util/osutil"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/curtisnewbie/miso/version"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver/goast"
	"github.com/dave/dst/decorator/resolver/gopackages"
	"github.com/dave/dst/dstutil"
)

const (
	typeInvalidMisoInboundVal = "miso.Inbound"
	typeInvalidMisoRailPtr    = "*miso.Rail"
	typeInvalidGormDbVal      = "gorm.DB"
	typeInvalidCommonUserPtr  = "*common.User"
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
	flagTags            = hash.NewSet(tagNgTable, tagRaw, tagIgnore)
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
	invalidInjectTokens = map[string]string{
		typeInvalidMisoInboundVal: typeMisoInboundPtr,
		typeInvalidMisoRailPtr:    typeMisoRail,
		typeInvalidGormDbVal:      typeGormDbPtr,
		typeInvalidCommonUserPtr:  typeCommonUser,
	}
)

var (
	Debug = flag.Bool("debug", false, "Enable debug log")
	log   = cli.NewLog(cli.LogWithDebug(Debug), cli.LogWithCaller(func(level string) bool { return level != "INFO" }))
)

func main() {
	flags.WithDescriptionBuilder(func(printlnf func(v string, args ...any)) {
		printlnf("misoapi - automatically generate web endpoint in go based on misoapi-* comments\n")
		printlnf("  Supported miso version: %v\n", version.Version)
	})
	flags.WithExtraBuilder(func(printlnf func(v string, args ...any)) {
		printlnf("\nFor example:\n")
		printlnf("  misoapi-http: GET /open/api/doc                                     // http method and url")
		printlnf("  misoapi-desc: open api endpoint to retrieve documents               // description")
		printlnf("  misoapi-query-doc: page: curent page index                          // query parameter")
		printlnf("  misoapi-header-doc: Authorization: bearer authorization token       // header parameter")
		printlnf("  misoapi-scope: PROTECTED                                            // access scope")
		printlnf("  misoapi-resource: document:read                                     // resource code")
		printlnf("  misoapi-ngtable                                                     // generate angular table code")
		printlnf("  misoapi-raw                                                         // raw endpoint without auto request/response json handling")
		printlnf("  misoapi-json-resp-type: MyResp                                      // json response type (struct), for raw api only")
		printlnf("  misoapi-ignore                                                      // ignored by misoapi")
		printlnf("")
		printlnf("Important:\n")
		printlnf(strutil.Spaces(2) + "By default, misoapi looks for `func PrepareWebServer(rail miso.Rail) error` in file './internal/web/web.go'.")
		printlnf(strutil.Spaces(2) + "If file is not found, APIs are registered in init() func, however it's not recommended as it's implicit.")
		printlnf(strutil.Spaces(2) + "If the file is found, APIs are registered explicitly in PrepareWebServer(..) func, and you should")
		printlnf(strutil.Spaces(2) + "makesure the PrepareWebServer(..) is called in miso.PreServerBootstrap(..)")
		printlnf("")
	})
	flags.Parse()

	files, err := walkDir(".", ".go")
	if err != nil {
		log.Errorf("walkDir failed, %v", err)
		return
	}
	if err := parseFiles(files); err != nil {
		log.Errorf("parseFiles failed, %v", err)
	}
}

type GroupedApiDecl struct {
	Dir     string
	Pkg     string
	PkgPath string
	Apis    []ApiDecl
	Imports hash.Set[string]
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

	modName, err := guessModName()
	if err != nil {
		return err
	}

	pathApiDecls := make(map[string]GroupedApiDecl)
	addApiDecl := func(p string, pkg string, pkgPath string, d ApiDecl, imports hash.Set[string]) {
		dir, _ := path.Split(p)
		v, ok := pathApiDecls[dir]
		if ok {
			v.Apis = append(v.Apis, d)
			v.Imports.AddAll(imports.CopyKeys())
			pathApiDecls[dir] = v
		} else {
			imp := hash.NewSet[string]()
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
		dir := path.Dir(path.Dir(path.Clean(df.Path)))
		var pkgPath string
		if dir == "." {
			pkgPath = modName + "/" + df.Dst.Name.Name
		} else {
			pkgPath = modName + "/" + dir + "/" + df.Dst.Name.Name
		}
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

	webGoPath := "." + string(os.PathSeparator) + path.Join("internal", "web", "web.go")
	regApiDstFiles := slutil.Transform(dstFiles,
		slutil.MapFunc(func(f DstFile) string { return f.Path }),
		slutil.FilterFunc(func(p string) bool {
			return webGoPath == p
		}),
	)

	// if internal/web/web.go is found, we call the registere func explicitly, if not, we register in init()
	doInsertRegisterApiFunc := len(regApiDstFiles) > 0
	misoapiFnName := "init"
	if doInsertRegisterApiFunc {
		misoapiFnName = "RegisterApi"
	} else {
		log.Debugf("file %v not found, register misoapi generated code in init()", webGoPath)
	}

	genPkgs := hash.NewSet[string]() // misoapi_generated.go pkg paths
	baseIndent := 1
	for dir, v := range pathApiDecls {
		for _, ad := range v.Apis {
			log.Debugf("%v (%v) => %#v", dir, v.Pkg, ad)
		}

		imports, code, err := genGoApiRegister(v.Apis, baseIndent, v.Imports)
		if err != nil {
			log.Errorf("Generate code failed, %v", err)
			continue
		}
		if code == "" {
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

		out := strutil.NamedSprintf(`// auto generated by misoapi ${misoVersion} at ${nowTimeStr}, please do not modify
package ${package}

import (
${importStr}
)

func ${misoapiFnName}() {
${code}
}
`, map[string]any{
			"misoapiFnName": misoapiFnName,
			"misoVersion":   version.Version,
			"nowTimeStr":    util.Now().FormatClassicLocale(),
			"package":       v.Pkg,
			"code":          code,
			"importStr":     importSb.String(),
		})
		genPkgs.Add(v.PkgPath)

		log.Debugf("%v (%v) => \n\n%v", dir, v.Pkg, out)
		outFile := fmt.Sprintf("%vmisoapi_generated.go", dir)

		// if generated file already existed, check if the content is still the same
		prev, err := os.ReadFile(outFile)
		if err == nil {
			prevs := util.UnsafeByt2Str(prev)
			if i := strings.Index(prevs, "\n"); i > -1 && i+1 < len(prevs) {
				prevs = prevs[i+1:]
			}
			outBody := out[strings.Index(out, "\n")+1:]
			if prevs == outBody {
				log.Debugf("generated code remain the same, skipping %v", outFile)
				continue
			}
		}

		// flush generated code
		f, err := osutil.OpenRWFile(outFile)
		if err != nil {
			return err
		}
		_ = f.Truncate(0)
		_, err = f.WriteString(out)
		f.Close()
		if err != nil {
			return err
		}
		log.Infof("Generated code written to %v, using pkg: %v, api count: %d", outFile, v.Pkg, len(v.Apis))
	}

	// insert func calls to register apis
	if doInsertRegisterApiFunc {
		sort.SliceStable(regApiDstFiles, func(i, j int) bool { return regApiDstFiles[i] < regApiDstFiles[j] })
		if err := insertMisoApiRegisterFunc(regApiDstFiles[0], "web", modName, genPkgs.CopyKeys()); err != nil {
			log.Errorf("Insert misoapi register func in %v failed, %v", regApiDstFiles[0], err)
			return err
		}
	}

	return nil
}

type DstFile struct {
	Dst  *dst.File
	Path string
}

func parseApiDecl(cursor *dstutil.Cursor, srcPath string, importSpec map[string]string) (ApiDecl, hash.Set[string], bool) {
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
		log.Debugf("alias: %v, importPath: %v", alias, importPath)
	case *dst.FuncDecl:
		imports := hash.NewSet[string]()
		tags, ok := parseMisoApiTag(srcPath, n.Decs.Start)
		if ok {
			log.Debugf("type results: %#v", n.Type.Results)
			log.Debugf("tags: %+v", tags)
			for _, t := range tags {
				kv, ok := t.BodyKV()
				log.Debugf("tag -> %#v, kv: %#v, ok: %v", t, kv, ok)
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
	return ApiDecl{}, hash.Set[string]{}, false
}

func guessImport(n string, importSpec map[string]string, imports hash.Set[string]) {
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

func parseParamMeta(l *dst.FieldList, path string, funcName string, importSpec map[string]string, imports hash.Set[string]) []ParamMeta {
	if l == nil {
		return []ParamMeta{}
	}
	pm := make([]ParamMeta, 0)
	for i, p := range l.List {
		var varName string
		if len(p.Names) > 0 {
			varName = p.Names[0].String()
		}

		log.Debugf("func: %v, param [%v], p: %#v", funcName, i, p.Type)

		typeName := parseParamName(p.Type, importSpec, imports)
		if typeName != "" {
			pm = append(pm, ParamMeta{Name: varName, Type: typeName})
		} else {
			log.Errorf("failed to parse param[%d]: %v %#v, %v: %v", i, p.Names, p.Type, path, funcName)
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
	Flags       hash.Set[string]

	JsonRespType string
	Imports      hash.Set[string]
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
		if alt, ok := d.invalidInjectToken(p.Type); !ok {
			panic(errs.NewErrf("Found invalid func arg type: '%v' in '%v(..)', maybe you mean '%v'?", p.Type, d.FuncName, alt))
		}
		var v string = d.guessInjectToken(p.Type, extra...)
		paramTokens = append(paramTokens, v)
	}
	return strings.Join(paramTokens, ", ")
}

func (d ApiDecl) invalidInjectToken(t string) (string, bool) {
	alt, ok := invalidInjectTokens[t]
	return alt, !ok
}

func (d ApiDecl) printInvokeFunc(extra ...func(typ string) (string, bool)) string {
	params := d.injectFuncParams(extra...)
	return fmt.Sprintf("%v(%v)", d.FuncName, params)
}

func genGoApiRegister(dec []ApiDecl, baseIndent int, imports hash.Set[string]) (hash.Set[string], string, error) {
	w := strutil.NewIndentWriter("\t")
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
				w.StepIn(func(w *strutil.IndentWriter) {
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
				w.StepIn(func(w *strutil.IndentWriter) {
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
						w.StepIn(func(iw *strutil.IndentWriter) {
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
					w.StepIn(func(w *strutil.IndentWriter) {
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
				w.StepIn(func(w *strutil.IndentWriter) {
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
			w.NoLbWritefWhen(extraLines > 0, "%s", l)
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
		Flags:   hash.NewSet[string](),
		Imports: hash.NewSet[string](),
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
	return ad, !strutil.IsBlankStr(ad.Method) && !strutil.IsBlankStr(ad.Url)
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
				log.Debugf("%v -> %v, command: %v, body: %v", path, s, pre, m)
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
				if descTmp != "" {
					descTmp += " " + s
				} else {
					descTmp += s
				}
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
		return nil, errs.Wrap(err)
	}
	files := make([]FsFile, 0, len(entries))
	for _, et := range entries {
		fi, err := et.Info()
		if err != nil {
			log.Errorf("%v", err)
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

func parseParamName(t dst.Expr, importSpec map[string]string, imports hash.Set[string]) string {
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
	case *dst.MapType:
		var kType string = parseParamName(v.Key, importSpec, imports)
		if kType == "" {
			return ""
		}
		var vType string = parseParamName(v.Value, importSpec, imports)
		if vType == "" {
			return ""
		}
		return fmt.Sprintf("map[%v]%v", kType, vType)
	default:
		miso.Warnf("dst.Expr: %#v", t)
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

// Add _ imports
//
// e.g.,
//
//	var restDstFiles []string = slutil.Transform(dstFiles,
//			slutil.MapFunc(func(f DstFile) string { return f.Path }),
//			slutil.FilterFunc(func(p string) bool {
//				return path.Base(p) == "main.go"
//			}),
//		)
//
//	for _, m := range restDstFiles {
//			if err := addBlankImports(m, "web", modName, genPkgs.CopyKeys()); err != nil {
//				log.ErrorPrintlnf("Add imports in main.go failed, %v", err)
//				panic(err)
//			}
//	}
func addBlankImports(filePath string, pkgName string, modName string, importPath []string) error {
	dir := filepath.Dir(filePath)
	log.Debugf("dir: %v, filePath: %v, modName: %v", dir, filePath, modName)

	fset := token.NewFileSet()
	dec := decorator.NewDecoratorWithImports(fset, pkgName, goast.New())

	fc, err := osutil.ReadFileAll(filePath)
	if err != nil {
		return err
	}

	f, err := dec.Parse(fc)
	if err != nil {
		return errs.Wrap(err)
	}

	var importDecl *dst.GenDecl
	for _, decl := range f.Decls {
		if genDecl, ok := decl.(*dst.GenDecl); ok && genDecl.Tok == token.IMPORT {
			importDecl = genDecl
			break
		}
	}

	if importDecl == nil {
		importDecl = &dst.GenDecl{
			Tok:   token.IMPORT,
			Specs: []dst.Spec{},
		}
		newDecls := make([]dst.Decl, 0, len(f.Decls)+1)
		newDecls = append(newDecls, importDecl)
		newDecls = append(newDecls, f.Decls...)
		f.Decls = newDecls
	}

	// filter already included imports
	importPath = slutil.Filter(importPath, func(s string) bool {
		for _, imp := range f.Imports {
			if imp.Path.Value == fmt.Sprintf("%q", s) {
				return false
			}
		}
		return true
	})
	if len(importPath) < 1 {
		return nil
	}

	// insert import specs
	for _, imp := range importPath {
		newImport := &dst.ImportSpec{
			Name: &dst.Ident{
				Name: "_",
			},
			Path: &dst.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("%q", imp),
			},
		}
		importDecl.Specs = append(importDecl.Specs, newImport)
	}

	// restore to *ast.File
	r := decorator.NewRestorerWithImports(modName, gopackages.New(dir))
	fr := r.FileRestorer()
	rstf, err := fr.RestoreFile(f)
	if err != nil {
		return errs.Wrap(err)
	}

	// write file
	buf := &bytes.Buffer{}
	if err := format.Node(buf, fr.Fset, rstf); err != nil {
		return errs.Wrap(err)
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return errs.Wrap(err)
	}

	log.Infof("Added import %q to %s\n", importPath, filePath)
	return nil
}

func insertMisoApiRegisterFunc(filePath string, pkgName string, modName string, importPath []string) error {
	if len(importPath) < 1 {
		return nil
	}

	dir := filepath.Dir(filePath)
	log.Debugf("dir: %v, filePath: %v, modName: %v, importPath: %+v", dir, filePath, modName, importPath)

	f, err := parseDstFile(filePath, pkgName)
	if err != nil {
		return err
	}

	// filter already included imports
	var importDecl *dst.GenDecl
	for _, decl := range f.Decls {
		if genDecl, ok := decl.(*dst.GenDecl); ok && genDecl.Tok == token.IMPORT {
			importDecl = genDecl
			break
		}
	}

	if importDecl == nil {
		importDecl = &dst.GenDecl{
			Tok:   token.IMPORT,
			Specs: []dst.Spec{},
		}
		newDecls := make([]dst.Decl, 0, len(f.Decls)+1)
		newDecls = append(newDecls, importDecl)
		newDecls = append(newDecls, f.Decls...)
		f.Decls = newDecls
	}

	// insert import specs
	for _, imp := range slutil.Filter(append(importPath, importMiso), func(s string) bool {
		for _, imp := range f.Imports {
			if imp.Path.Value == fmt.Sprintf("%q", s) {
				return false
			}
		}
		return true
	}) {
		newImport := &dst.ImportSpec{
			Path: &dst.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("%q", imp),
			},
		}
		importDecl.Specs = append(importDecl.Specs, newImport)
	}

	var targetFuncName = "PrepareWebServer"
	var fdecl *dst.FuncDecl
	for _, decl := range f.Decls {
		funcDecl, ok := decl.(*dst.FuncDecl)
		if !ok {
			continue
		}
		if funcDecl.Name.Name != targetFuncName {
			continue
		}
		if funcDecl.Recv != nil { // not function
			continue
		}
		fdecl = funcDecl
		break
	}

	if fdecl == nil {
		log.Infof("Func %v(..) missing in %v, writing new func declaration", targetFuncName, filePath)
		fdecl = &dst.FuncDecl{
			Name: &dst.Ident{
				Name: targetFuncName,
			},
			Type: &dst.FuncType{
				Params: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{
								Name: "rail",
							}},
							Type: &dst.Ident{
								Name: "Rail",
								Path: importMiso,
							},
						},
					},
				},
				Results: &dst.FieldList{
					List: []*dst.Field{
						{
							Type: &dst.Ident{
								Name: "error",
							},
						},
					},
				},
			},
			Body: &dst.BlockStmt{
				List: []dst.Stmt{
					&dst.ReturnStmt{
						Results: []dst.Expr{
							&dst.Ident{Name: "nil"},
						},
					},
				},
			},
		}
		fdecl.Decs.Before = dst.EmptyLine
		fdecl.Decs.Start.Append("// Do not modify this if misoapi is used, misoapi may modify func body")
		f.Decls = append(f.Decls, fdecl)
	}

	callFuncs := slutil.Transform(importPath,
		slutil.MapFunc[string, Pair](func(imp string) Pair {
			var p string
			if !strings.HasSuffix(imp, path.Dir(filePath)) {
				p = imp
			}
			return Pair{K: p, V: "RegisterApi"}
		}),
		slutil.FilterFunc[Pair](func(p Pair) bool {
			for _, st := range fdecl.Body.List {
				exprStmt, ok := st.(*dst.ExprStmt)
				if !ok {
					continue
				}
				callExpr, ok := exprStmt.X.(*dst.CallExpr)
				if !ok {
					continue
				}
				ident, ok := callExpr.Fun.(*dst.Ident)
				if ok && ident.Name == p.V && ident.Path == p.K {
					return false
				}
			}
			return true
		}))

	if len(callFuncs) > 0 {

		// Insert the new call at the beginning of the function body
		// Create a new list with our call followed by existing statements
		newBody := make([]dst.Stmt, 0, len(fdecl.Body.List)+len(callFuncs))
		for _, cf := range callFuncs {
			fun := &dst.Ident{Path: cf.K, Name: cf.V}
			st := &dst.ExprStmt{X: &dst.CallExpr{Fun: fun}}
			st.Decs.After = dst.NewLine
			newBody = append(newBody, st)

			pb := cf.K
			if pb != "" {
				pb = path.Base(pb) + "."
			}
			log.Infof("Inserting %v%v() call in %v.%v(..)", pb, cf.V, pkgName, targetFuncName)
		}
		newBody = append(newBody, fdecl.Body.List...)
		fdecl.Body.List = newBody
		log.Infof("Inserted misoapi register func in %s", filePath)
	}

	if err := restoreDstFile(filePath, modName, f); err != nil {
		return err
	}

	return nil
}

func restoreDstFile(filePath string, modName string, f *dst.File) error {
	// restore to *ast.File
	r := decorator.NewRestorerWithImports(modName, gopackages.New(path.Dir(filePath)))
	fr := r.FileRestorer()
	rstf, err := fr.RestoreFile(f)
	if err != nil {
		return errs.Wrap(err)
	}

	// write file
	buf := &bytes.Buffer{}
	if err := format.Node(buf, fr.Fset, rstf); err != nil {
		return errs.Wrap(err)
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return errs.Wrap(err)
	}
	return nil
}

// check if package is imported in main
func checkImported(pkgPath string) {
	out, err := cli.Run(nil, "grep", []string{"-r", pkgPath, "--include", "*.go"})
	if err != nil {
		var extErr *exec.ExitError
		if errors.As(err, &extErr) && extErr.ExitCode() == 1 {
			log.Infof(cli.ANSIRed+"Warning: (1) package '%v' is not imported!"+cli.ANSIReset, pkgPath)
		} else {
			log.Errorf("check package import failed, pkg: %v, out: %s, %v", pkgPath, out, err)
		}
	} else {
		if strings.TrimSpace(string(out)) == "" {
			log.Infof(cli.ANSIRed+"Warning: (2) package '%v' is not imported!"+cli.ANSIReset, pkgPath)
		}
	}
}

func guessModName() (string, error) {
	modName := ""
	out, err := cli.Run(nil, "go", []string{"list", "-m"})
	if err != nil {
		return "", errs.Wrap(fmt.Errorf("%s, %v", out, err))
	}
	modName = strings.TrimSpace(string(out))
	return modName, nil
}

func parseDstFile(fpath string, shortPkgName string) (*dst.File, error) {
	fset := token.NewFileSet()
	dec := decorator.NewDecoratorWithImports(fset, shortPkgName, goast.New())

	fc, err := osutil.ReadFileAll(fpath)
	if err != nil {
		return nil, err
	}

	f, err := dec.Parse(fc)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return f, nil
}
