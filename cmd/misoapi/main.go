package main

import (
	"flag"
	"go/parser"
	"io/fs"
	"os"
	"strings"

	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/version"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
)

const (
	MisoApiPrefix = "misoapi-"
)

type FsFile struct {
	Path string
	File fs.FileInfo
}

var (
	Debug = flag.Bool("debug", false, "Debug")
)

// For example,
//
// misoapi-http: GET http://localhost:8080/my/api
// misoapi-desc: My api
// misoapi-query-doc: name : my name
// misoapi-header-doc: auth : my auth
// misoapi-scope: PUBLIC
// misoapi-resource: my-api-resource
func main() {
	flag.Parse()
	util.Printlnf("misoapi for miso@%v\n", version.Version)

	files, err := walkDir(".")
	if err != nil {
		util.Printlnf("error - %v", err)
		return
	}
	if err := parseFiles(files); err != nil {
		util.Printlnf("error - %v", err)
	}
}

func parseFiles(files []FsFile) error {
	dstFiles, err := parseFileAst(files)
	if err != nil {
		return err
	}

	if *Debug {
		for _, f := range dstFiles {
			util.Printlnf("Found %v", f.Path)
		}
	}

	// filter and edit the files
	apiDecls := make([]ApiDecl, 0)
	for _, df := range dstFiles {
		dstutil.Apply(df.Dst,
			func(c *dstutil.Cursor) bool {
				ad, ok := parseApiDecl(c, df.Path, df.Dst)
				if ok {
					apiDecls = append(apiDecls, ad)
				}
				return true
			},
			func(cursor *dstutil.Cursor) bool {
				return true
			},
		)
	}

	if *Debug {
		for _, ad := range apiDecls {
			util.Printlnf("=> %#v", ad)
		}
	}

	baseIndent := 0
	code, err := genGoApiRegister(apiDecls, baseIndent)
	if err != nil {
		util.Printlnf("error - %v", err)
	} else {
		util.Printlnf("code:\n\n%v\n\n", code)
	}

	return nil
}

type DstFile struct {
	Dst  *dst.File
	Path string
}

type OutputFile struct {
	Path    string
	Content []string
}

func parseApiDecl(cursor *dstutil.Cursor, path string, file *dst.File) (ApiDecl, bool) {
	switch n := cursor.Node().(type) {
	case *dst.FuncDecl:
		tags, ok := parseMisoApiTag(path, n.Decs.Start)
		if ok {
			if *Debug {
				util.Printlnf("type results: %#v", n.Type.Results)
				util.Printlnf("tags: %+v", tags)
			}
			for _, t := range tags {
				kv, ok := t.BodyKV()
				if *Debug {
					util.Printlnf("tag -> %#v, kv: %#v, ok: %v", t, kv, ok)
				}
			}
			ad, ok := BuildApiDecl(tags)
			if ok {
				ad.TypeParams = n.Decs.TypeParams
				ad.Params = n.Decs.Params
				ad.Results = n.Decs.Results
				ad.FuncParams = parseParamMeta(n.Type.Params)
				ad.FuncResults = parseParamMeta(n.Type.Results)
				ad.FuncName = n.Name.String()
			}
			return ad, ok
		}
	}
	return ApiDecl{}, false
}

func parseParamMeta(l *dst.FieldList) []ParamMeta {
	if l == nil {
		return []ParamMeta{}
	}
	pm := make([]ParamMeta, 0)
	for i, p := range l.List {
		var varName string
		if len(p.Names) > 0 {
			varName = p.Names[0].String()
		}

		var typeName string
		switch v := p.Type.(type) {
		case *dst.SelectorExpr:
			typeName = v.X.(*dst.Ident).Name
			sn := v.Sel.String()
			if sn != "" {
				typeName = typeName + "." + sn
			}
		case *dst.Ident:
			typeName = v.Name
		case *dst.StarExpr:
			xsel := v.X.(*dst.SelectorExpr)
			typeName = xsel.X.(*dst.Ident).Name
			sn := xsel.Sel.String()
			if sn != "" {
				typeName = typeName + "." + sn
			}
		case *dst.ArrayType:
			typeName = "[]" + v.Elt.(*dst.Ident).Name
		default:
			util.Printlnf("error - failed to parse param[%d]: %v %#v", i, p.Names, p.Type)
		}
		if typeName != "" {
			pm = append(pm, ParamMeta{Name: varName, Type: typeName})
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

	TypeParams dst.Decorations
	Params     dst.Decorations
	Results    dst.Decorations

	FuncParams  []ParamMeta
	FuncResults []ParamMeta

	PkgPath  string
	FuncName string
}

func genGoApiRegister(dec []ApiDecl, baseIndent int) (string, error) {
	w := util.NewIndentWriter("\t")
	w.SetIndent(baseIndent)

	for _, d := range dec {
		var custReqType string

		for _, p := range d.FuncParams {
			switch p.Type {
			case "miso.Inbound", "miso.Rail", "gorm.DB", "common.User":
				continue
			default:
				custReqType = p.Type
			}
			if custReqType != "" {
				break
			}
		}

		var custResType string
		for _, p := range d.FuncResults {
			if p.Type != "error" {
				custResType = p.Type
				break
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
					case "miso.Inbound":
						v = "inb"
					case "miso.Rail":
						v = "inb.Rail()"
					case "gorm.DB":
						v = "miso.GetMySQL()"
					case "common.User":
						v = "common.GetUser(inb.Rail())"
					case custReqType:
						v = "req"
					}
					paramTokens = append(paramTokens, v)
				}
				w.Writef("return %v(%v)", d.FuncName, strings.Join(paramTokens, ", "))
			})
			w.NoLbWritef("})")
		} else {
			isRaw := len(d.FuncParams) == 1 && d.FuncParams[0].Type == "miso.Inbound" && len(d.FuncResults) < 1
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
						case "miso.Inbound":
							v = "inb"
						case "miso.Rail":
							v = "inb.Rail()"
						case "gorm.DB":
							v = "miso.GetMySQL()"
						case "common.User":
							v = "common.GetUser(inb.Rail())"
						}
						paramTokens = append(paramTokens, v)
					}
					w.Writef("return %v(%v)", d.FuncName, strings.Join(paramTokens, ", "))
				})
				w.NoLbWritef("})")
			}
		}
		if d.Desc != "" {
			w.NoIndWritef(".\n")
			if len(d.Header) > 0 || len(d.Query) > 0 {
				w.NoLbWritef("Desc(\"%v\")", d.Desc)
			} else {
				w.Writef("Desc(\"%v\")", d.Desc)
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
		w.NoIndWritef("\n")
		w.SetIndent(baseIndent)
	}
	return w.String(), nil
}

func BuildApiDecl(tags []MisoApiTag) (ApiDecl, bool) {
	ad := ApiDecl{}
	for _, t := range tags {
		switch t.Command {
		case "http":
			lr := strings.SplitN(t.Body, " ", 2)
			if len(lr) < 2 {
				return ad, false
			}
			ad.Method = strings.ToUpper(strings.TrimSpace(lr[0]))
			ad.Url = strings.TrimSpace(lr[1])
		case "desc":
			ad.Desc = t.Body
		case "scope":
			ad.Scope = t.Body
		case "resource":
			ad.Resource = t.Body
		case "query-doc":
			kv, ok := t.BodyKV()
			if ok {
				ad.Query = append(ad.Query, kv)
			}
		case "header-doc":
			kv, ok := t.BodyKV()
			if ok {
				ad.Header = append(ad.Header, kv)
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
	for _, s := range start {
		s, _ = strings.CutPrefix(s, "//")
		s = strings.TrimSpace(s)
		if m, ok := strings.CutPrefix(s, MisoApiPrefix); ok {
			if pi := strings.Index(m, ":"); pi > -1 {
				pre := m[:pi]
				m = m[pi+1:]
				if *Debug {
					util.Printlnf("%v -> %v, command: %v, body: %v", path, s, pre, m)
				}
				// return OutputFile{}, false
				t = append(t, MisoApiTag{
					Command: strings.TrimSpace(pre),
					Body:    strings.TrimSpace(m),
				})
			} else {
				continue
			}
		}
	}
	return t, len(t) > 0
}

func parseFileAst(files []FsFile) ([]DstFile, error) {
	parsed := make([]DstFile, 0)
	for _, f := range files {
		p := f.Path
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
			util.Printlnf("error - %v", err)
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
