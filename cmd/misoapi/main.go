package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

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

	tagHttp          = "http"
	tagDesc          = "desc"
	tagScope         = "scope"
	tagRes           = "resource"
	tagQueryDocV1    = "query-doc"
	tagHeaderDocV1   = "header-doc"
	tagQueryDocV2    = "query"
	tagHeaderDocV2   = "header"
	tagNgTable       = "ngtable"
	tagConfigDesc    = "config-desc"
	tagConfigDefault = "config-default"
	tagConfigSection = "config-section"
)

var (
	refPat = regexp.MustCompile(`ref\(([a-zA-Z0-9 \\-\\_\.]+)\)`)
	flags  = util.NewSet[string]()
)

var (
	Debug       = flag.Bool("debug", false, "Enable debug log")
	ConfigTable = flag.Bool("config-table", false, "Generate config table")
)

func init() {
	flags.Add(tagNgTable)
}

func main() {
	flag.Usage = func() {
		util.Printlnf("\nmisoapi - automatically generate web endpoint in go based on misoapi-* comments\n")
		util.Printlnf("  Supported miso version: %v\n", version.Version)
		util.Printlnf("Usage of %s:", os.Args[0])
		flag.PrintDefaults()
		util.Printlnf("\nFor example:\n")
		util.Printlnf("  misoapi-http: GET /open/api/doc")
		util.Printlnf("  misoapi-desc: open api endpoint to retrieve documents")
		util.Printlnf("  misoapi-query-doc: page: curent page index")
		util.Printlnf("  misoapi-header-doc: Authorization: bearer authorization token")
		util.Printlnf("  misoapi-scope: PROTECTED")
		util.Printlnf("  misoapi-resource: document:read")
		util.Printlnf("  misoapi-ngtable")
		util.Printlnf("")
	}
	flag.Parse()

	files, err := walkDir(".")
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

	pathApiDecls := make(map[string]GroupedApiDecl)
	addApiDecl := func(p string, pkg string, d ApiDecl, imports util.Set[string]) {
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
				Apis:    []ApiDecl{d},
				Imports: imp,
			}
		}
	}

	configDecl := map[string]ConfigSection{}
	var section string
	for _, df := range dstFiles {
		importSepc := map[string]string{}
		dstutil.Apply(df.Dst,
			func(c *dstutil.Cursor) bool {
				// parse api declaration
				ad, imports, ok := parseApiDecl(c, df.Path, importSepc)
				if ok {
					addApiDecl(df.Path, df.Dst.Name.Name, ad, imports)
				}

				// parse config declaration
				if ns := parseConfigDecl(c, df.Path, section, configDecl); ns != "" {
					section = ns
				}
				return true
			},
			func(cursor *dstutil.Cursor) bool {
				return true
			},
		)
	}

	util.DebugPrintlnf(*Debug, "configs: %#v", configDecl)
	if *ConfigTable {
		printConfigTable(configDecl)
	}

	baseIndent := 1
	for dir, v := range pathApiDecls {
		for _, ad := range v.Apis {
			if *Debug {
				util.Printlnf("[DEBUG] %v (%v) => %#v", dir, v.Pkg, ad)
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
				util.Printlnf("Generated code remain the same, skipping %v", outFile)
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
}

func genGoApiRegister(dec []ApiDecl, baseIndent int, imports util.Set[string]) (util.Set[string], string, error) {
	w := util.NewIndentWriter("\t")
	w.SetIndent(baseIndent)
	imports.Add(importMiso)

	for i, d := range dec {
		var custReqType string

		for _, p := range d.FuncParams {
			switch p.Type {
			case typeMisoInboundPtr, typeMisoRail:
				continue
			case typeCommonUser:
				imports.Add(importCommonUser)
				continue
			case typeGormDbPtr, typeMySqlQryPtr:
				imports.Add(importMySQL)
				continue
			default:
				if custReqType == "" {
					custReqType = p.Type
				}
			}
		}

		// TODO: this code is terrible, have to fix it :(
		var custResType string
		var errorOnly bool = false
		var noError bool = true
		for _, p := range d.FuncResults {
			switch p.Type {
			case "error":
				if len(d.FuncResults) == 1 {
					errorOnly = true
				}
				noError = false
				continue
			case typeCommonUser:
				if custResType == "" {
					custResType = p.Type
				}
				imports.Add(importCommonUser)
				continue
			default:
				if custResType == "" {
					custResType = p.Type
				}
			}
		}

		resType := "any"
		if custResType != "" {
			resType = custResType
		}

		mtd := d.Method[:1] + strings.ToLower(d.Method[1:])
		if custReqType != "" {
			w.Writef("miso.I%v(\"%v\",", mtd, d.Url)
			w.IncrIndent()
			w.Writef("func(inb *miso.Inbound, req %v) (%v, error) {", custReqType, resType)
			w.StepIn(func(w *util.IndentWriter) {
				paramTokens := make([]string, 0, len(d.FuncParams))
				for _, p := range d.FuncParams {
					var v string
					switch p.Type {
					case typeMisoInboundPtr:
						v = "inb"
					case typeMisoRail:
						v = "inb.Rail()"
					case typeMySqlQryPtr:
						v = "mysql.NewQuery(mysql.GetMySQL())"
					case typeGormDbPtr:
						v = "mysql.GetMySQL()"
					case typeCommonUser:
						v = "common.GetUser(inb.Rail())"
					case custReqType:
						v = "req"
					}
					paramTokens = append(paramTokens, v)
				}
				if errorOnly { // TODO: refactor this
					w.Writef("return nil, %v(%v)", d.FuncName, strings.Join(paramTokens, ", "))
				} else if len(d.FuncResults) < 1 {
					w.Writef("%v(%v)", d.FuncName, strings.Join(paramTokens, ", "))
					w.Writef("return nil, nil")
				} else if noError {
					w.Writef("return %v(%v), nil", d.FuncName, strings.Join(paramTokens, ", "))
				} else {
					w.Writef("return %v(%v)", d.FuncName, strings.Join(paramTokens, ", "))
				}
			})
			w.NoLbWritef("})")
		} else {
			isRaw := len(d.FuncParams) == 1 && d.FuncParams[0].Type == typeMisoInboundPtr && len(d.FuncResults) < 1
			if isRaw {
				w.NoLbWritef("miso.Raw%v(\"%v\", %v)", mtd, d.Url, d.FuncName)
				if d.Desc != "" || len(d.Header) > 0 || len(d.Query) > 0 {
					w.IncrIndent()
				}
			} else {
				w.Writef("miso.%v(\"%v\",", mtd, d.Url)
				w.IncrIndent()
				w.Writef("func(inb *miso.Inbound) (%v, error) {", resType)
				w.StepIn(func(w *util.IndentWriter) {
					paramTokens := make([]string, 0, len(d.FuncParams))
					for _, p := range d.FuncParams {
						var v string
						switch p.Type {
						case typeMisoInboundPtr:
							v = "inb"
						case typeMisoRail:
							v = "inb.Rail()"
						case typeMySqlQryPtr:
							v = "mysql.NewQuery(mysql.GetMySQL())"
						case typeGormDbPtr:
							v = "mysql.GetMySQL()"
						case typeCommonUser:
							v = "common.GetUser(inb.Rail())"
						}
						paramTokens = append(paramTokens, v)
					}
					if errorOnly { // TODO: refactor this
						w.Writef("return nil, %v(%v)", d.FuncName, strings.Join(paramTokens, ", "))
					} else if len(d.FuncResults) < 1 {
						w.Writef("%v(%v)", d.FuncName, strings.Join(paramTokens, ", "))
						w.Writef("return nil, nil")
					} else if noError {
						w.Writef("return %v(%v), nil", d.FuncName, strings.Join(paramTokens, ", "))
					} else {
						w.Writef("return %v(%v)", d.FuncName, strings.Join(paramTokens, ", "))
					}
				})
				w.NoLbWritef("})")
			}
		}

		if d.FuncName != "" {
			w.NoIndWritef(".\n")
			if d.Flags.Has(tagNgTable) || d.Desc != "" || d.Scope != "" || d.Resource != "" || len(d.Header) > 0 || len(d.Query) > 0 {
				w.NoLbWritef("Extra(miso.ExtraName, \"%s\")", d.FuncName)
			} else {
				w.Writef("Extra(miso.ExtraName, \"%s\")", d.FuncName)
			}
		}

		if d.Flags.Has(tagNgTable) {
			w.NoIndWritef(".\n")
			if d.Desc != "" || d.Scope != "" || d.Resource != "" || len(d.Header) > 0 || len(d.Query) > 0 {
				w.NoLbWritef("Extra(miso.ExtraNgTable, true)")
			} else {
				w.Writef("Extra(miso.ExtraNgTable, true)")
			}
		}
		if d.Desc != "" {
			w.NoIndWritef(".\n")
			if d.Scope != "" || d.Resource != "" || len(d.Header) > 0 || len(d.Query) > 0 {
				w.NoLbWritef("Desc(\"%v\")", d.Desc)
			} else {
				w.Writef("Desc(\"%v\")", d.Desc)
			}
		}
		if d.Scope != "" {
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
			if d.Resource != "" || len(d.Header) > 0 || len(d.Query) > 0 {
				w.NoLbWritef(l)
			} else {
				w.Writef(l)
			}
		}
		if d.Resource != "" {
			w.NoIndWritef(".\n")
			ref, isRef := parseRef(d.Resource)
			var res string
			if !isRef {
				res = "\"" + d.Resource + "\""
			} else {
				res = ref
			}
			if len(d.Header) > 0 || len(d.Query) > 0 {
				w.NoLbWritef("Resource(%v)", res)
			} else {
				w.Writef("Resource(%v)", res)
			}
		}
		for i, dh := range d.Header {
			w.NoIndWritef(".\n")
			if i < len(d.Header)-1 || len(d.Query) > 0 {
				w.NoLbWritef("DocHeader(\"%v\", \"%v\")", dh.K, dh.V)
			} else {
				w.Writef("DocHeader(\"%v\", \"%v\")", dh.K, dh.V)
			}
		}
		for i, dh := range d.Query {
			w.NoIndWritef(".\n")
			if i < len(d.Query)-1 {
				w.NoLbWritef("DocQueryParam(\"%v\", \"%v\")", dh.K, dh.V)
			} else {
				w.Writef("DocQueryParam(\"%v\", \"%v\")", dh.K, dh.V)
			}
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
		Flags: util.NewSet[string](),
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
		case tagNgTable:
			ad.Flags.Add(tagNgTable)
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
	i := strings.Index(m.Body, ":")
	if i < 0 {
		return Pair{}, false
	}
	return Pair{
		K: strings.TrimSpace(m.Body[:i]),
		V: strings.TrimSpace(m.Body[i+1:]),
	}, true
}

func parseMisoApiTag(path string, start dst.Decorations) ([]MisoApiTag, bool) {
	t := []MisoApiTag{}
	currIsDesc := false
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

func walkDir(n string) ([]FsFile, error) {
	entries, err := os.ReadDir(n)
	if err != nil {
		return nil, err
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
			ff, err := walkDir(p)
			if err == nil {
				files = append(files, ff...)
			}
		} else {
			if strings.HasSuffix(fi.Name(), ".go") {
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

type ConfigSection = []ConfigDecl
type ConfigDecl struct {
	Name         string
	Description  string
	DefaultValue string
}

func parseConfigDecl(cursor *dstutil.Cursor, srcPath string, section string, configs map[string]ConfigSection) (newSection string) {

	switch n := cursor.Node().(type) {
	case *dst.GenDecl:
		comment := n.Decs.Start
		tags, ok := parseMisoApiTag(srcPath, comment)
		if !ok {
			return section
		}
		for _, t := range tags {
			if t.Command == tagConfigSection {
				section = t.Body
			}
		}
	case *dst.ValueSpec:
		comment := n.Decs.Start
		tags, ok := parseMisoApiTag(srcPath, comment)
		if !ok {
			return section
		}

		var constName string
		for _, n := range n.Names {
			constName = n.Name
		}

		var found bool = false
		var cd ConfigDecl = ConfigDecl{}
		for _, t := range tags {
			switch t.Command {
			case tagConfigDesc:
				found = true
				cd.Description = t.Body
			case tagConfigDefault:
				found = true
				cd.DefaultValue = t.Body
			}
		}

		if !found {
			return section
		}

		for _, v := range n.Values {
			if bl, ok := v.(*dst.BasicLit); ok && bl.Kind == token.STRING {
				cd.Name = util.UnquoteStr(bl.Value)
			}
		}
		if cd.Name == "" {
			return section
		}
		util.DebugPrintlnf(*Debug, "parseConfigDecl() %v: (%v) %v -> %#v", srcPath, section, constName, cd)
		sec := section
		if sec == "" {
			sec = "General"
		}
		configs[sec] = append(configs[sec], cd)
	}
	return section
}

func printConfigTable(configs map[string][]ConfigDecl) {
	if len(configs) < 1 {
		return
	}
	defer println("")
	util.Printlnf("# Configurations\n")
	for sec, l := range configs {
		if len(l) < 1 {
			continue
		}
		maxNameLen := len("property")
		maxDescLen := len("description")
		maxValLen := len("default value")
		for _, c := range l {
			if len(c.Name) > maxNameLen {
				maxNameLen = len(c.Name)
			}
			if len(c.Description) > maxDescLen {
				maxDescLen = len(c.Description)
			}
			if len(c.DefaultValue) > maxValLen {
				maxValLen = len(c.DefaultValue)
			}
		}

		util.Printlnf("## %v\n", sec)
		println(util.NamedSprintf("| ${Name} | ${Description} | ${DefaultValue} |", map[string]any{
			"Name":         util.PadSpace(-maxNameLen, "property"),
			"Description":  util.PadSpace(-maxDescLen, "description"),
			"DefaultValue": util.PadSpace(-maxValLen, "default value"),
		}))
		println(util.NamedSprintf("| ${Name} | ${Description} | ${DefaultValue} |", map[string]any{
			"Name":         util.PadToken(-maxNameLen, "---", "-"),
			"Description":  util.PadToken(-maxDescLen, "---", "-"),
			"DefaultValue": util.PadToken(-maxValLen, "---", "-"),
		}))
		for _, c := range l {
			c.Name = util.PadSpace(-maxNameLen, c.Name)
			c.Description = util.PadSpace(-maxDescLen, c.Description)
			c.DefaultValue = util.PadSpace(-maxValLen, c.DefaultValue)
			println(util.NamedSprintfv("| ${Name} | ${Description} | ${DefaultValue} |", c))
		}
	}
}
