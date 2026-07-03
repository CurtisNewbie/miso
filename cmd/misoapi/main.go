package main

import (
	"bytes"
	"encoding/json"
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
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/miso/docgen"
	"github.com/curtisnewbie/miso/tools"
	"github.com/curtisnewbie/miso/util/async"
	"github.com/curtisnewbie/miso/util/atom"
	"github.com/curtisnewbie/miso/util/cli"
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
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/viper"
	goimports "golang.org/x/tools/imports"
)

const (
	typeInvalidMisoInboundVal = "miso.Inbound"
	typeInvalidMisoRailPtr    = "*miso.Rail"
	typeInvalidGormDbVal      = "gorm.DB"
	typeInvalidFlowUserPtr    = "*flow.User"
)

const (
	MisoApiPrefix = "misoapi-"

	typeMisoInboundPtr = "*miso.Inbound"
	typeMisoRail       = "miso.Rail"
	typeGormDbPtr      = "*gorm.DB"
	typeMySqlQryPtr    = "*mysql.Query"
	typeFlowUser       = "flow.User"

	importFlowUser = "github.com/curtisnewbie/miso/flow"
	importMiso     = "github.com/curtisnewbie/miso/miso"
	importGorm     = "gorm.io/gorm"
	importMySQL    = "github.com/curtisnewbie/miso/middleware/mysql"
	importDbQuery  = "github.com/curtisnewbie/miso/middleware/dbquery"
	importAtom     = "github.com/curtisnewbie/miso/util/atom"
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
		typeFlowUser:       importFlowUser,
		typeMySqlQryPtr:    importMySQL,
		typeGormDbPtr:      importDbQuery,
		typeMisoInboundPtr: "",
		typeMisoRail:       "",
		"atom.Time":        importAtom,
		"*atom.Time":       importAtom,
	}
	injectToken = map[string]string{
		typeMisoInboundPtr: "inb",
		typeMisoRail:       "inb.Rail()",
		typeMySqlQryPtr:    "mysql.NewQuery(dbquery.GetDB())",
		typeGormDbPtr:      "dbquery.GetDB()",
		typeFlowUser:       "inb.Rail().User()",
	}
	invalidInjectTokens = map[string]string{
		typeInvalidMisoInboundVal: typeMisoInboundPtr,
		typeInvalidMisoRailPtr:    typeMisoRail,
		typeInvalidGormDbVal:      typeGormDbPtr,
		typeInvalidFlowUserPtr:    typeFlowUser,
	}
)

var (
	log = cli.NewLog(cli.LogWithDebug(Debug), cli.LogWithCaller(func(level string) bool { return level != "INFO" }))

	Debug           = flag.Bool("debug", false, "Enable debug log")
	Perf            = flag.Bool("perf", false, "Enable performance timing logs")
	SkipPkgs        = flag.String("skip-pkgs", "", "Comma-separated list of package paths to skip (e.g., 'internal/web,internal/middleware')")
	Doc             = flags.BoolVal("doc", true, "Generate API docs, true or false (markdown)", false)
	DocFile         = flag.String("file", "doc/api.md", "Output file for API docs")
	DocPort         = flag.String("port", "8080", "Server port for cURL examples in docs")
	DocAppName      = flag.String("appname", "", "Application name for docs (falls back to conf.yml 'app.name' if unset)")
	DocJavaDemo     = flags.BoolVal("java-demo", false, "Generate Java HttpClient demo (OkHttp + Jackson)", false)
	DocGoDemo       = flags.BoolVal("go-demo", true, "Generate miso.TClient demo", false)
	DocNgClientDemo = flags.BoolVal("ngclient-demo", true, "Generate Angular HttpClient demo", false)
	Oas             = flag.Bool("oas", false, "Generate OpenAPI 3.0 JSON spec (for all APIs)")
	OasFile         = flag.String("oas-file", "doc/openapi.json", "Output file for OpenAPI 3.0 JSON spec")
	OasServer       = flag.String("oas-server", "", "Server URL for the generated OpenAPI 3.0 spec")
	PerApiOas       = flags.BoolVal("per-api-oas", false, "Per endpoint OpenAPI JSON", false)
	GoClientFile    = flag.String("go-client-file", "", "Output Go source file with type definitions and TClient demos (e.g., 'internal/client/api_client.go')")
	GoClientApis    = flag.String("go-client-apis", "", "Comma-separated API patterns to include (format: 'METHOD:path' or 'path', e.g., 'POST:/api/user,GET:/api/order/*')")
	GoClientCompile = flags.BoolVal("go-client-compile", false, "Whether the generated Go client file should compile (false adds //go:build miso_gen_do_not_build)", false)
)

// perfLog logs at INFO level only when the -perf flag is set.
func perfLog(format string, args ...any) {
	if *Perf {
		log.Infof(format, args...)
	}
}

// loadConfYml loads conf.yml using Viper. Returns the Viper instance or nil on failure.
func loadConfYml(configFile string) *viper.Viper {
	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		return nil
	}
	return v
}

// resolveFromViper reads a key from Viper, using the raw key as a fallback
// if nested lookup fails (keys with dots in YAML are stored flat).
func resolveFromViper(v *viper.Viper, key string) string {
	if v.IsSet(key) {
		return v.GetString(key)
	}
	// Dotted keys in flat YAML (e.g., "app.name") may be stored as-is
	// rather than nested. Try all keys.
	for _, k := range v.AllKeys() {
		if k == key {
			return v.GetString(k)
		}
	}
	return ""
}

// matchSkipPkg checks if pkgPath should be skipped based on skipPkgs patterns.
// Patterns can be full import paths (github.com/curtisnewbie/myapp/internal/web)
// or relative paths (internal/web). Two matching modes:
//   - Exact pkg: pkgPath ends with the pattern (e.g., "demo" matches ".../demo")
//   - Subtree: pattern appears as a path segment followed by "/" (e.g., "demo"
//     matches ".../demo/appdemo/api" because "/demo/" appears in the path)
func matchSkipPkg(pkgPath string, skipPkgs []string) bool {
	for _, sp := range skipPkgs {
		if sp == "" {
			continue
		}
		if strings.HasSuffix(pkgPath, sp) || strings.Contains(pkgPath, "/"+sp+"/") {
			return true
		}
	}
	return false
}

func main() {
	start := time.Now()
	defer func() {
		perfLog("misoapi total elapsed: %v", time.Since(start))
	}()

	flags.WithDescriptionBuilder(func(printlnf func(v string, args ...any)) {
		printlnf("misoapi - automatically generate web endpoint in go based on misoapi-* comments\n")
		printlnf("  Supported miso version: %v\n", version.Version)
	})
	flags.WithExtraBuilder(func(printlnf func(v string, args ...any)) {
		printlnf("\nFor example:\n")
		printlnf("  misoapi-http: GET /open/api/doc                                     // http method and url")
		printlnf("  misoapi-desc: open api endpoint to retrieve documents               // description")
		printlnf("  misoapi-query: page: curent page index                              // query parameter")
		printlnf("  misoapi-header: Authorization: bearer authorization token           // header parameter")
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
		printlnf(strutil.Spaces(2) + "Use -oas flag to also generate OpenAPI 3.0 JSON spec.")
		printlnf("")
	})
	flags.Parse()

	// Parse skip packages once (global flag)
	var skipPkgsList []string
	if *SkipPkgs != "" {
		skipPkgsList = strings.Split(*SkipPkgs, ",")
		for i := range skipPkgsList {
			skipPkgsList[i] = strings.TrimSpace(skipPkgsList[i])
		}
	}

	// Check if go.mod exists in the current directory.
	_, goModErr := os.Stat("go.mod")
	if goModErr == nil {
		// Single module mode — current behavior.
		processModule(".", skipPkgsList)
		return
	}

	// Monorepo mode: no go.mod at root; scan subdirectories for Go modules.
	modDirs := findGoModDirs(".")
	if len(modDirs) == 0 {
		log.Errorf("no go.mod found in current directory or subdirectories; misoapi requires a Go module")
		return
	}

	log.Infof("Monorepo detected: %d module(s) found", len(modDirs))

	origDir, err := os.Getwd()
	if err != nil {
		log.Errorf("failed to get current directory: %v", err)
		return
	}

	// Process modules in parallel using async.AwaitFutures
	af := async.NewAwaitFutures[struct{}](nil) // nil pool = one goroutine per module
	for _, modDir := range modDirs {
		md := modDir // capture loop variable
		absModDir := filepath.Join(origDir, md)

		log.Infof("=== Processing module: %s ===", md)
		af.SubmitAsync(func() (struct{}, error) {
			processModule(absModDir, skipPkgsList)
			return struct{}{}, nil
		})
	}
	if err := af.AwaitAnyErr(); err != nil {
		log.Errorf("Module processing failed: %v", err)
	}
}

// findGoModDirs recursively finds directories containing go.mod under root.
// Nested modules are skipped — only the outermost module per subtree is returned.
// Vendor, .git, node_modules, and testdata directories are skipped.
func findGoModDirs(root string) []string {
	var dirs []string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if name == "vendor" || name == ".git" || name == "node_modules" || name == "testdata" {
			return filepath.SkipDir
		}
		if path == root {
			return nil // only interested in subdirectories
		}
		if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
			dirs = append(dirs, path)
			return filepath.SkipDir // don't recurse into module directories
		}
		return nil
	})
	return dirs
}

// processModule runs the core misoapi logic in the given module directory.
func processModule(dir string, skipPkgsList []string) {
	appName := *DocAppName
	port := *DocPort

	// If conf.yml exists and flags were not explicitly set, load defaults from it.
	if fi, err := os.Stat(filepath.Join(dir, "conf.yml")); err == nil && !fi.IsDir() {
		if v := loadConfYml(filepath.Join(dir, "conf.yml")); v != nil {
			d := map[string]bool{}
			flag.Visit(func(f *flag.Flag) { d[f.Name] = true })
			if !d["appname"] {
				if s := resolveFromViper(v, "app.name"); s != "" {
					appName = s
				}
			}
			if !d["port"] {
				if s := resolveFromViper(v, "server.port"); s != "" {
					port = s
				}
			}
		}
	}

	files, err := walkDir(dir, ".go")
	if err != nil {
		log.Errorf("walkDir failed, %v", err)
		return
	}
	_, err = parseFiles(dir, files, true, skipPkgsList)
	if err != nil {
		log.Errorf("parseFiles failed, %v", err)
	}

	if *Doc {
		if err := generateDocs(dir, skipPkgsList, appName, port); err != nil {
			log.Errorf("generateDocs failed, %v", err)
		}
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

func parseFiles(dir string, files []FsFile, generateCode bool, skipPkgs []string) (map[string]GroupedApiDecl, error) {
	start := time.Now()
	defer func() {
		perfLog("parseFiles total elapsed: %v", time.Since(start))
	}()

	astStart := time.Now()
	dstFiles, err := parseFileAst(files)
	perfLog("parseFileAst elapsed: %v, %d files", time.Since(astStart), len(dstFiles))
	if err != nil {
		return nil, err
	}

	modName, err := guessModName(dir)
	if err != nil {
		return nil, err
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

	extractStart := time.Now()
	for _, df := range dstFiles {
		fileDir := path.Dir(path.Dir(path.Clean(df.Path)))
		// Strip module root prefix to get module-relative dir for pkgPath
		if dir != "." {
			fileDir = strings.TrimPrefix(fileDir, dir+string(os.PathSeparator))
		}
		var pkgPath string
		if fileDir == "." {
			pkgPath = modName + "/" + df.Dst.Name.Name
		} else {
			pkgPath = modName + "/" + fileDir + "/" + df.Dst.Name.Name
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
	perfLog("API extraction (AST walk) elapsed: %v, %d groups", time.Since(extractStart), len(pathApiDecls))

	webGoPath := path.Join(dir, "internal", "web", "web.go")
	regApiDstFiles := slutil.Transform(dstFiles,
		slutil.MapFunc(func(f DstFile) string { return f.Path }),
		slutil.FilterFunc(func(p string) bool {
			return path.Clean(webGoPath) == path.Clean(p)
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

	if generateCode {
		genStart := time.Now()
		genPkgs := hash.NewSet[string]() // misoapi_generated.go pkg paths
		baseIndent := 1
		for dir, v := range pathApiDecls {
			if matchSkipPkg(v.PkgPath, skipPkgs) {
				log.Infof("Skipping package %v (matched by -skip-pkgs)", v.PkgPath)
				continue
			}
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

			rawOut := strutil.NamedSprintf(`// auto generated by misoapi ${misoVersion} at ${nowTimeStr}, please do not modify
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
				"nowTimeStr":    atom.Now().FormatClassicLocale(),
				"package":       v.Pkg,
				"code":          code,
				"importStr":     importSb.String(),
			})
			genPkgs.Add(v.PkgPath)

			log.Debugf("%v (%v) => \n\n%v", dir, v.Pkg, rawOut)
			outFile := fmt.Sprintf("%vmisoapi_generated.go", dir)

			// Run goimports on the generated output so its import formatting
			// matches what the existing file already has (from previous runs).
			// This ensures byte-level comparison is accurate regardless of
			// import ordering/grouping differences.
			outBytes, err := goimports.Process(outFile, []byte(rawOut), &goimports.Options{
				TabWidth:   8,
				TabIndent:  true,
				Comments:   true,
				Fragment:   true,
				FormatOnly: false,
			})
			if err != nil {
				log.Errorf("goimports processing failed for %v, using raw output: %v", outFile, err)
				outBytes = []byte(rawOut)
			}
			out := strutil.UnsafeByt2Str(outBytes)

			// if generated file already existed, check if the content is still the same
			prev, err := os.ReadFile(outFile)
			if err == nil {
				prevs := strutil.UnsafeByt2Str(prev)
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
				return pathApiDecls, err
			}
			_ = f.Truncate(0)
			_, err = f.WriteString(out)
			f.Close()
			if err != nil {
				return pathApiDecls, err
			}
			log.Infof("Generated code written to %v, using pkg: %v, api count: %d", outFile, v.Pkg, len(v.Apis))
		}

		// insert func calls to register apis
		if doInsertRegisterApiFunc {
			sort.SliceStable(regApiDstFiles, func(i, j int) bool { return regApiDstFiles[i] < regApiDstFiles[j] })
			if err := insertMisoApiRegisterFunc(dir, regApiDstFiles[0], "web", modName, genPkgs.CopyKeys()); err != nil {
				log.Errorf("Insert misoapi register func in %v failed, %v", regApiDstFiles[0], err)
				return pathApiDecls, err
			}
		}
		perfLog("Code generation + write elapsed: %v", time.Since(genStart))
	}

	return pathApiDecls, nil
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
		case importFlowUser, importMiso, importGorm, importMySQL:
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
			kv, _ := t.BodyKV()
			ad.Query = append(ad.Query, kv)
		case tagHeaderDocV1, tagHeaderDocV2:
			kv, _ := t.BodyKV()
			ad.Header = append(ad.Header, kv)
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
		return Pair{K: m.Body, V: m.Body}, false
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
		// Normalize to canonical package name (e.g., alias "at" -> canonical "atom")
		// to ensure generated code uses the correct package name matching the import.
		if p, ok := importSpec[n]; ok {
			n = path.Base(p)
		}
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

func insertMisoApiRegisterFunc(dir string, filePath string, pkgName string, modName string, importPath []string) error {
	if len(importPath) < 1 {
		return nil
	}

	// Compute the Go import path of the package containing the web.go file
	// (e.g., github.com/curtisnewbie/user-vault/internal/web).
	webRelDir := path.Dir(filePath)
	if dir != "." {
		webRelDir = strings.TrimPrefix(webRelDir, dir+string(os.PathSeparator))
	}
	webImportPath := modName + "/" + webRelDir

	fileDir := filepath.Dir(filePath)
	log.Debugf("fileDir: %v, filePath: %v, modName: %v, webImportPath: %v, importPath: %+v", fileDir, filePath, modName, webImportPath, importPath)

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

	// insert import specs (skip the web package's own import path to avoid self-import)
	importPaths := slutil.Filter(importPath, func(imp string) bool { return imp != webImportPath })
	for _, imp := range slutil.Filter(append(importPaths, importMiso), func(s string) bool {
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
			// If this is the web package itself, use bare call (no import prefix)
			if imp != webImportPath {
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

// writeGoApiFile generates a standalone Go source file with type definitions
// and TClient demos for APIs matching the configured patterns.
func writeGoApiFile(dir string, allDocs []miso.HttpRouteDoc) error {
	if *GoClientFile == "" {
		return nil
	}
	fp := filepath.Join(dir, *GoClientFile)

	// Build match function from include/exclude patterns
	match := buildApiPatternMatcher(*GoClientApis)

	// Collect type defs (deduplicated) and client functions for matching routes
	seenContent := hash.NewSet[string]()
	var typeDefs []string
	var clientFuncs []string
	for _, d := range allDocs {
		if !match(d) {
			continue
		}
		if d.JsonReqGoDef != "" && seenContent.Add(d.JsonReqGoDef) {
			typeDefs = append(typeDefs, d.JsonReqGoDef)
		}
		if d.JsonRespGoDef != "" && seenContent.Add(d.JsonRespGoDef) {
			typeDefs = append(typeDefs, d.JsonRespGoDef)
		}
		if d.MisoTClientWithoutTypes != "" {
			clientFuncs = append(clientFuncs, d.MisoTClientWithoutTypes)
		}
	}

	if len(clientFuncs) == 0 {
		log.Infof("No APIs matched for Go client file generation, skipping %s", fp)
		return nil
	}

	// Collect import paths from field descriptions so the generated file
	// has correct imports (e.g., atom.Time -> github.com/curtisnewbie/miso/util/atom)
	// instead of relying on goimports which may resolve to a wrong package.
	importPaths := collectTypeImports(allDocs, match)

	// Build file content
	var b strings.Builder

	// Optional build tag for non-compilable mode
	if !*GoClientCompile {
		b.WriteString("//go:build miso_gen_do_not_build\n\n")
	}
	b.WriteString("// auto generated by miso, please do not modify\n")

	// Package declaration derived from output path directory
	pkgName := path.Base(path.Dir(fp))
	if pkgName == "." || pkgName == "/" {
		pkgName = "main"
	}
	b.WriteString("\npackage " + pkgName + "\n")

	b.WriteString("\nimport (\n")
	for _, imp := range importPaths {
		b.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}
	b.WriteString(")\n")

	// Write type definitions
	for _, def := range typeDefs {
		b.WriteString("\n" + def + "\n")
	}

	// Write TClient functions
	for _, fn := range clientFuncs {
		b.WriteString("\n" + fn + "\n")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(fp), 0755); err != nil {
		return errs.Wrapf(err, "failed to create output directory for %s", fp)
	}
	if err := os.WriteFile(fp, []byte(b.String()), 0644); err != nil {
		return errs.Wrapf(err, "failed to write Go client file %s", fp)
	}

	// Fix imports with goimports
	tools.RunGoImports(fp)

	log.Infof("Go client file written to %s (%d APIs, %d type defs)", fp, len(clientFuncs), len(typeDefs))
	return nil
}

// collectTypeImports collects unique import paths from the field descriptions
// of matching API docs. This ensures the generated Go client file includes
// correct imports for external types (e.g., atom.Time -> github.com/curtisnewbie/miso/util/atom)
// instead of relying on goimports which may resolve to a wrong package.
func collectTypeImports(allDocs []miso.HttpRouteDoc, match func(miso.HttpRouteDoc) bool) []string {
	pkgs := hash.NewSet[string]()
	pkgs.Add("github.com/curtisnewbie/miso/miso")
	for _, d := range allDocs {
		if !match(d) {
			continue
		}
		collectFieldsPkgs(d.JsonRequestDesc.Fields, pkgs)
		collectFieldsPkgs(d.JsonResponseDesc.Fields, pkgs)
	}
	keys := pkgs.CopyKeys()
	sort.Strings(keys)
	return keys
}

func collectFieldsPkgs(fields []miso.FieldDesc, pkgs hash.Set[string]) {
	for _, f := range fields {
		if f.TypePkg != "" {
			pkgs.Add(f.TypePkg)
		}
		if len(f.Fields) > 0 {
			collectFieldsPkgs(f.Fields, pkgs)
		}
	}
}

// buildApiPatternMatcher returns a function that tests whether an HttpRouteDoc
// matches the comma-separated pattern string. Patterns use format "METHOD:path"
// or "path". "*" in path acts as a glob wildcard.
func buildApiPatternMatcher(patternStr string) func(d miso.HttpRouteDoc) bool {
	patterns := strutil.SplitStr(patternStr, ",")
	if len(patterns) == 0 {
		return func(d miso.HttpRouteDoc) bool { return true }
	}
	return func(d miso.HttpRouteDoc) bool {
		return strutil.MatchApiPatterns(d.Method, d.Url, patterns)
	}
}

func guessModName(dir string) (string, error) {
	modName := ""
	out, err := cli.Run("go", []string{"list", "-m"}, func(cmd *exec.Cmd) { cmd.Dir = dir })
	if err != nil {
		return "", errs.Wrap(fmt.Errorf("%s, %v", out, err))
	}
	modName = strings.TrimSpace(string(out))
	// In go.work setups, go list -m may return multiple modules (one per line).
	// We only want the module containing the current directory (the first line).
	if i := strings.IndexByte(modName, '\n'); i > -1 {
		modName = modName[:i]
	}
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

// generateDocs generates static API documentation from source code parsing.
func generateDocs(dir string, skipPkgs []string, appName string, port string) error {
	start := time.Now()
	defer func() {
		perfLog("generateDocs total elapsed: %v", time.Since(start))
	}()

	modName, err := guessModName(dir)
	if err != nil {
		return err
	}

	// If -appname not set, default to the last segment of the module name
	// (e.g., "github.com/curtisnewbie/xxx" → "xxx").
	if appName == "" {
		appName = path.Base(modName)
	}

	manualFiles, err := walkDir(dir, ".go")
	if err != nil {
		return fmt.Errorf("walkDir for docs failed: %v", err)
	}

	// Filter out files from skipped packages
	if len(skipPkgs) > 0 {
		manualFiles = slutil.Filter(manualFiles, func(f FsFile) bool {
			relDir := path.Dir(f.Path)
			if dir != "." {
				relDir = strings.TrimPrefix(relDir, dir+string(os.PathSeparator))
			}
			pkgPath := modName + "/" + relDir
			return !matchSkipPkg(pkgPath, skipPkgs)
		})
	}

	// Parse all files once; both endpoint and pipeline extraction share the ASTs
	// embedded in each SourceFile.
	parseStart := time.Now()
	convertedFiles := make([]docgen.SourceFile, 0, len(manualFiles))
	for _, f := range manualFiles {
		parsed, err := decorator.ParseFile(nil, f.Path, nil, parser.ParseComments)
		if err != nil {
			return errs.Wrapf(err, "failed to parse %s", f.Path)
		}
		convertedFiles = append(convertedFiles, docgen.SourceFile{Path: f.Path, Ast: parsed})
	}
	perfLog("Parse all files: %v, %d files", time.Since(parseStart), len(convertedFiles))

	buildStart := time.Now()
	docgen.LogPerf = *Perf
	preloaded, misoPkg, err := docgen.LoadPackagesAt(dir)
	if err != nil {
		return errs.Wrapf(err, "LoadPackagesAt failed")
	}
	perfLog("LoadPackagesAt elapsed: %v, %d dirs", time.Since(buildStart), len(preloaded))

	allDocs := docgen.BuildManualRouteDocs(convertedFiles, modName, log, preloaded, misoPkg)
	perfLog("BuildManualRouteDocs elapsed: %v, %d endpoints", time.Since(buildStart), len(allDocs))

	pipelineDocs := docgen.BuildManualPipelineDocs(convertedFiles, modName, log, preloaded, misoPkg)
	log.Infof("Built %d pipeline docs", len(pipelineDocs))

	log.Infof("Built %d route docs", len(allDocs))
	for i, d := range allDocs {
		log.Debugf("  Doc[%d] %s %-40s (%s) reqFields=%d, respFields=%d", i, d.Method, d.Url, d.SourceFile, len(d.JsonRequestDesc.Fields), len(d.JsonResponseDesc.Fields))
	}

	// Fill generated fields for each doc
	renderStart := time.Now()
	seenGoTypes := hash.NewSet[string]()
	for i := range allDocs {
		d := &allDocs[i]
		d.Curl = miso.GenRouteCurl(*d, port)

		// Generate Go/Ts definitions for request types.
		// Only POST/PUT have request bodies — empty structs (e.g. EmptyReq{}) are
		// valid for these methods. Other methods never have a body.
		// GenGoDef returns "" for primitives/non-struct types; returns
		// "type  struct { }" for zero-value TypeDesc (TypeName="").
		jsonReqGoDef, jsonReqGoDefTypeName := miso.GenGoDef(d.JsonRequestDesc, hash.NewSet[string]())
		if jsonReqGoDef != "" && d.JsonRequestDesc.TypeName != "" &&
			(d.Method == "POST" || d.Method == "PUT") {
			d.JsonReqTsDef = miso.GenTsDef(d.JsonRequestDesc)
			d.JsonTsDef = d.JsonReqTsDef
			d.JsonReqGoDef = jsonReqGoDef
			d.JsonReqGoDefTypeName = jsonReqGoDefTypeName
		}
		if len(d.JsonResponseDesc.Fields) > 0 {
			d.JsonRespTsDef = miso.GenTsDef(d.JsonResponseDesc)
			if d.JsonTsDef != "" {
				d.JsonTsDef += "\n"
			}
			d.JsonTsDef += d.JsonRespTsDef
			d.JsonRespGoDef, d.JsonRespGoDefTypeName = miso.GenGoDef(d.JsonResponseDesc, hash.NewSet[string]())
		}

		// Miso HTTP Client demo
		rawTClient := miso.GenTClientDemo(*d, appName)

		d.MisoTClientDemo = rawTClient
		if d.JsonRespGoDef != "" {
			d.MisoTClientDemo = d.JsonRespGoDef + "\n" + d.MisoTClientDemo
		}
		if d.JsonReqGoDef != "" {
			d.MisoTClientDemo = d.JsonReqGoDef + "\n" + d.MisoTClientDemo
		}
		d.MisoTClientWithoutTypes = rawTClient

		// Angular HttpClient demo (exclude OpenAPI to keep simpler)
		d.NgHttpClientDemo = miso.GenNgHttpClientDemo(*d, appName, true)

		// Java HttpClient demo
		if *DocJavaDemo {
			d.JavaClientDemo = miso.GenJavaHttpClientDemo(*d, appName)
		}

		_ = seenGoTypes // used for global Go type defs in future
	}
	perfLog("Per-doc rendering elapsed: %v, %d docs", time.Since(renderStart), len(allDocs))

	// Generate standalone Go client file if requested
	if *GoClientFile != "" {
		if err := writeGoApiFile(dir, allDocs); err != nil {
			return err
		}
	}

	// Generate aggregate OpenAPI 3.0 spec
	if *Oas {
		oasStart := time.Now()
		oasPath := filepath.Join(dir, *OasFile)
		rootSpec := &openapi3.T{
			OpenAPI: "3.0.0",
			Info: &openapi3.Info{
				Title:   appName,
				Version: "1.0.0",
			},
		}
		if port != "" {
			rootSpec.AddServer(&openapi3.Server{URL: "http://localhost:" + port})
		}

		for _, d := range allDocs {
			miso.GenOpenApiDoc(d, rootSpec, *OasServer)
		}

		oasJson, err := json.MarshalIndent(rootSpec, "", "  ")
		if err != nil {
			return errs.Wrapf(err, "failed to marshal OpenAPI spec")
		}

		if err := os.MkdirAll(filepath.Dir(oasPath), 0755); err != nil {
			return errs.Wrapf(err, "failed to create output directory for %s", oasPath)
		}
		if err := os.WriteFile(oasPath, oasJson, 0644); err != nil {
			return errs.Wrapf(err, "failed to write OpenAPI spec file %s", oasPath)
		}
		perfLog("OpenAPI spec generation + write elapsed: %v", time.Since(oasStart))
		log.Infof("OpenAPI spec written to %s", oasPath)
	}

	mdStart := time.Now()
	markdown := miso.GenMarkDownDoc(allDocs, pipelineDocs, miso.MarkdownOpt{
		ExclTClientDemo:  !*DocGoDemo,
		ExclNgClientDemo: !*DocNgClientDemo,
		ExclOpenApi:      !*PerApiOas,
	})

	docFilePath := filepath.Join(dir, *DocFile)
	if err := os.MkdirAll(filepath.Dir(docFilePath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory for %s: %v", docFilePath, err)
	}
	if err := os.WriteFile(docFilePath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("failed to write doc file %s: %v", docFilePath, err)
	}
	perfLog("GenMarkDownDoc + write elapsed: %v", time.Since(mdStart))

	log.Infof("API docs written to %s", docFilePath)
	return nil
}
