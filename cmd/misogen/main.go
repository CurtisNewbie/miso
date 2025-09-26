package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/util/cli"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/curtisnewbie/miso/version"
)

const (
	ModFile  = "go.mod"
	MainFile = "main.go"
)

var (
	ModNameFlag    = flag.String("name", "", "Module name")
	StaticFlag     = flag.Bool("static", false, "Generate code to embed and statically host frontend project")
	SvcFlag        = flag.Bool("svc", false, "Generate code to integrate svc for automatic schema migration")
	DisableWebFlag = flag.Bool("disable-web", false, "Disable web server")
	CliFlag        = flag.Bool("cli", false, "Generate CLI style project")
)

func main() {
	flag.Usage = func() {
		fmt.Printf("\nmisogen, current miso version: %s\n\n", version.Version)
		flag.PrintDefaults()
	}
	flag.Parse()

	var initName string = *ModNameFlag
	if initName != "" {
		initName = strings.TrimSpace(initName)
	}

	if ok, err := util.FileExists(ModFile); err != nil || !ok {
		if err != nil {
			panic(err)
		}
		if !ok {
			if initName == "" {
				fmt.Printf("File %s not found\n\n", ModFile)
				fmt.Println("Create new module yourself (go mod init), or run misogen with your module name\ne.g.,\n\tmisogen -name github.com/curtisnewbie/applejuice")
				return
			}
			out, err := exec.Command("go", "mod", "init", initName).CombinedOutput()
			if err != nil {
				fmt.Println(util.UnsafeByt2Str(out))
				panic(err)
			}
			fmt.Printf("Initialized module '%s'\n", initName)
		}
	}

	ok, err := util.FileExists(MainFile)
	if err != nil {
		panic(fmt.Errorf("failed to open file %s, %v", MainFile, err))
	}
	if ok {
		panic(fmt.Sprintf("%s already exists", MainFile))
	}

	cpltModName := ""
	modName := ""
	modfCtn, err := util.ReadFileAll(ModFile)
	if err != nil {
		panic(fmt.Errorf("failed to read file %v, %v", ModFile, err))
	}

	{
		i := 0
		for j := 0; j < len(modfCtn); j++ {
			if modfCtn[j] == '\n' {
				if j == i || j == i+1 {
					i = j + 1
					continue
				}
				line := util.UnsafeByt2Str(modfCtn[i:j])
				line = strings.TrimSpace(line)
				if n, ok := strings.CutPrefix(line, "module"); ok {
					modName = strings.TrimSpace(n)
					cpltModName = modName
					k := strings.LastIndexByte(modName, '/')
					if k > 0 {
						modName = modName[k+1:]
					}
					break
				}
				i = j + 1
			}
		}
	}

	pkg := fmt.Sprintf("github.com/curtisnewbie/miso/miso@%s", version.Version)
	fmt.Printf("Installing dependency: %s\n", pkg)

	out, err := exec.Command("go", "get", "-x", pkg).CombinedOutput()
	if err != nil {
		fmt.Println(util.UnsafeByt2Str(out))
		panic(fmt.Errorf("failed to install miso, %v", err))
	}

	dirTree := buildDirTree(cpltModName, modName)
	if err := util.MkdirTree(dirTree); err != nil {
		panic(err)
	}

	if err := exec.Command("go", "mod", "tidy").Run(); err != nil {
		panic(err)
	}

	if err := exec.Command("go", "fmt", "./...").Run(); err != nil {
		cli.Printlnf("failed to fmt source code, %v", err)
	}
}

func buildDirTree(cpltModName string, modName string) util.DirTree {
	return util.DirTree{
		Name: ".",
		Childs: []util.DirTree{
			{
				Name:      "main.go",
				IsFile:    true,
				OnCreated: func(f *os.File) error { return CreateMainFile(f, cpltModName) },
			},
			{
				Name:      "conf.yml",
				Skip:      *CliFlag,
				IsFile:    true,
				OnCreated: func(f *os.File) error { return CreateConfFile(f, modName) },
			},
			{
				Name: "doc",
				Childs: []util.DirTree{
					{Name: ".gitkeep", IsFile: true},
				},
			},
			{
				Name: "internal",
				Childs: []util.DirTree{
					{
						Name: "server",
						Skip: *CliFlag,
						Childs: []util.DirTree{
							{
								Name:      "server.go",
								IsFile:    true,
								OnCreated: func(f *os.File) error { return CreateServerFile(f, cpltModName, modName) },
							},
							{
								Name:      "version.go",
								IsFile:    true,
								OnCreated: CreateVersionFile,
							},
						},
					},
					{
						Name: "schema",
						Skip: !*SvcFlag,
						Childs: []util.DirTree{
							{
								Name: "scripts",
								Childs: []util.DirTree{
									{
										Name:      "schema.sql",
										IsFile:    true,
										OnCreated: func(f *os.File) error { return CreateSchemaFile(f) },
									},
								},
							},
							{
								Name:      "migrate.go",
								IsFile:    true,
								OnCreated: func(f *os.File) error { return CreateSchemaMigrateFile(f) },
							},
						},
					},
					{
						Name: "static",
						Skip: !*StaticFlag,
						Childs: []util.DirTree{
							{
								Name:      "static.go",
								IsFile:    true,
								OnCreated: CreateStaticGoFile,
							},
							{
								Name: "static",
								Childs: []util.DirTree{
									{
										Name:      "miso.html",
										IsFile:    true,
										OnCreated: CreateStaticPlaceholderFile,
									},
								},
							},
						},
					},
					{
						Name: "config",
						Skip: *CliFlag,
						Childs: []util.DirTree{
							{
								Name:      "prop.go",
								IsFile:    true,
								OnCreated: CreateConfigPropFile,
							},
						},
					},
					{
						Name:   "repo",
						Skip:   *CliFlag,
						Childs: []util.DirTree{{Name: ".gitkeep", IsFile: true}},
					},
					{
						Name:   "domain",
						Skip:   *CliFlag,
						Childs: []util.DirTree{{Name: ".gitkeep", IsFile: true}},
					},
					{
						Name: "web",
						Skip: *DisableWebFlag || *CliFlag,
						Childs: []util.DirTree{
							{
								Name:      "web.go",
								IsFile:    true,
								OnCreated: CreateWebFile,
							},
						},
					},
					{
						Name:   ".gitkeep",
						IsFile: true,
						Skip:   !*CliFlag,
					},
				},
			},
		},
	}
}

func CreateConfFile(f *os.File, modName string) error {
	fmt.Printf("Initializing %s\n", f.Name())
	s := `# https://github.com/CurtisNewbie/miso/blob/main/doc/config.md

# To use following middlewares, make sure you have already imported relevant go module

# production mode, must be true in production.
mode.production: false

# http server
server:
  enabled: true
  host: 127.0.0.1
  port: 8080
  request-log:
    enabled: true
  health-check-url: "/health"
  log-routes: true
  graceful-shutdown-time-sec: 30
  auth:
    bearer: "" # bearer token for all api (including pprof)
  pprof:
    enabled: false
    auth:
      bearer: "" # bearer token for pprof api
  api-doc:
    enabled: true
    file: "./doc/api.md" # generated api doc file
    web:
      enabled: true
    path-prefix-app: true

# consul for service discovery
consul:
  enabled: false
  consul-address: "localhost:8500"

# redis connection
redis:
  enabled: false
  address: "localhost"
  port: 6379
  username: ""
  password: ""
  database: 0

# mysql connection
mysql:
  enabled: false
  host: "localhost"
  port: 3306
  user: ""
  password: ""
  database: "${dbName}"
  connection:
    - "charset=utf8mb4"
    - "parseTime=true"
    - "loc=UTC"
    - "readTimeout=30s"
    - "writeTimeout=30s"
    - "timeout=5s"
    - "collation=utf8mb4_general_ci"
    - "interpolateParams=false"

# sqlite configuration
sqlite:
  file: ""

# rabbitmq connection
rabbitmq:
  enabled: false
  host: "localhost"
  port: "5672"
  username: "guest"
  password: "guest"
  vhost: ""

# zookeeper connection
zk:
  enabled: false
  hosts:
    - "localhost"
  session-timeout: 5

# kafka connection
kafka:
  enabled: false
  server:
    addr:
      - "localhost:9092"

# nacos config center
nacos:
  enabled: false
  server:
    addr: "localhost"
    namespace: ""
    username: ""
    password: ""

# prometheus metrics
metrics:
  enabled: true
  route: "/metrics"
  auth:
    enabled: false
    bearer: "" # bearer for metrics api

logging:
  level: "info"
  rolling:
    file: "logs/${modName}.log"
`

	dbName := strings.ReplaceAll(strings.ToLower(modName), "-", "_")
	s = strutil.NamedSprintf(s, map[string]any{
		"modName": modName,
		"dbName":  dbName,
	})

	_, err := f.WriteString(s)
	return err
}

func CreateMainFile(f *os.File, cpltModName string) error {

	// cli style
	if *CliFlag {
		fmt.Printf("Initializing %s\n", f.Name())
		iw := strutil.NewIndentWriter("\t")
		iw.Writef("package main")
		iw.Writef("")
		iw.Writef("import (").
			StepIn(func(iw *strutil.IndentWriter) {
				iw.Writef("\"github.com/curtisnewbie/miso/miso\"")
				iw.Writef("\"github.com/curtisnewbie/miso/middleware/redis\"")
				iw.Writef("\"github.com/curtisnewbie/miso/middleware/mysql\"")
				iw.Writef("\"github.com/curtisnewbie/miso/util/flags\"")
			}).
			Writef(")").Writef("")

		iw.Writef("var (").
			StepIn(func(iw *strutil.IndentWriter) {
				iw.Writef("DebugFlag = flags.Bool(\"debug\", false, \"Enable debug log\", false)")
			}).
			Writef(")").Writef("")

		// main func
		iw.Writef("func main() {").
			StepIn(func(iw *strutil.IndentWriter) {
				iw.Writef("rail := miso.EmptyRail()")
				iw.Writef("flags.Parse()")
				iw.Writef("")
				iw.Writef("if *DebugFlag {").
					StepIn(func(iw *strutil.IndentWriter) {
						iw.Writef("miso.SetLogLevel(\"debug\")")
					}).
					Writef("}").Writef("")

				iw.Writef("// for mysql")
				iw.Writef("if err := mysql.InitMySQL(rail, mysql.MySQLConnParam{}); err != nil {").
					StepIn(func(iw *strutil.IndentWriter) {
						iw.Writef("panic(err)")
					}).
					Writef("}")

				iw.Writef("db := mysql.GetMySQL()")
				iw.Writef("if err := db.Exec(`SELECT 1`).Error; err != nil {").
					StepIn(func(iw *strutil.IndentWriter) {
						iw.Writef("panic(err)")
					}).
					Writef("}").Writef("")

				iw.Writef("// for redis")
				iw.Writef("red, err := redis.InitRedis(rail, redis.RedisConnParam{})")
				iw.Writef("if err != nil {").
					StepIn(func(iw *strutil.IndentWriter) {
						iw.Writef("panic(err)")
					}).
					Writef("}")

				iw.Writef("res, err := red.Ping(rail.Context()).Result()")
				iw.Writef("if err != nil {").
					StepIn(func(iw *strutil.IndentWriter) {
						iw.Writef("panic(err)")
					}).
					Writef("}")
				iw.Writef("rail.Infof(\"Ping result: %%v\", res)")

			}).
			Writef("}")

		if _, err := f.WriteString(iw.String()); err != nil {
			panic(err)
		}
		if err := exec.Command("go", "mod", "tidy").Run(); err != nil {
			panic(err)
		}
		return nil
	}

	// generic style
	sb, writef := strutil.NewIndWritef("\t")
	writef(0, "package main")
	writef(0, "")
	writef(0, "import (")
	writef(1, "\"%s/internal/server\"", cpltModName)
	writef(0, ")")
	writef(0, "")
	writef(0, "func main() {")
	writef(1, "server.BootstrapServer()")
	writef(0, "}")
	_, err := f.WriteString(sb.String())
	return err
}

func CreateSchemaFile(sf *os.File) error {
	fmt.Printf("Initializing %s\n", sf.Name())
	_, err := sf.WriteString("-- Initialize schema")
	return err
}

func CreateSchemaMigrateFile(f *os.File) error {
	fmt.Printf("Initializing %s\n", f.Name())
	sb, writef := strutil.NewIndWritef("\t")
	writef(0, "// This is for automated MySQL schema migration.")
	writef(0, "//")
	writef(0, "// See https://github.com/CurtisNewbie/svc for more information.")
	writef(0, "package schema")
	writef(0, "")
	writef(0, "import (")
	writef(1, "\"embed\"")
	writef(1, "")
	writef(1, "\"github.com/curtisnewbie/miso/middleware/svc\"")
	writef(0, ")")
	writef(0, "")
	writef(0, "//go:embed scripts/*.sql")
	writef(0, "var schemaFs embed.FS")
	writef(0, "")
	writef(0, "const (")
	writef(1, "BaseDir = \"scripts\"")
	writef(0, ")")
	writef(0, "")
	writef(0, "// Use miso svc middleware to handle schema migration, only executed on production mode.")
	writef(0, "//")
	writef(0, "// Script files should follow the classic semver, e.g., v0.0.1.sql, v0.0.2.sql, etc.")
	writef(0, "func EnableSchemaMigrate() {")
	writef(1, "svc.ExcludeSchemaFile(\"schema.sql\")")
	writef(1, "svc.EnableSchemaMigrate(schemaFs, BaseDir, \"\")")
	writef(0, "}")
	_, err := f.WriteString(sb.String())
	return err
}

func CreateStaticGoFile(f *os.File) error {
	fmt.Printf("Initializing %s\n", f.Name())
	sb, writef := strutil.NewIndWritef("\t")
	writef(0, "package static")
	writef(0, "")
	writef(0, "import (")
	writef(1, "\"embed\"")
	writef(1, "")
	writef(1, "\"github.com/curtisnewbie/miso/miso\"")
	writef(0, ")")
	writef(0, "")
	writef(0, "//go:embed static")
	writef(0, "var staticFs embed.FS")
	writef(0, "")
	writef(0, "const (")
	writef(1, "staticFsPre = \"/static\"")
	writef(0, ")")
	writef(0, "")
	writef(0, "func PrepareWebStaticFs() {")
	writef(1, "miso.PrepareWebStaticFs(staticFs, staticFsPre)")
	writef(0, "}")
	_, err := f.WriteString(sb.String())
	return err
}

func CreateStaticPlaceholderFile(f *os.File) error {
	fmt.Printf("Initializing %s\n", f.Name())
	_, err := f.WriteString(fmt.Sprintf("Powered by miso %s.", version.Version))
	return err
}

func CreateConfigPropFile(f *os.File) error {
	fmt.Printf("Initializing %s\n", f.Name())
	sb, writef := strutil.NewIndWritef("\t")
	writef(0, "package config")
	writef(0, "")
	writef(0, "// misoconfig-section: General Configuration")
	writef(0, "const (")
	writef(1, "// misoconfig-prop:")
	writef(1, "// PropMyProp = \"123\"")
	writef(0, ")")
	writef(0, "")
	writef(0, "// misoconfig-default-start")
	writef(0, "// misoconfig-default-end")

	_, err := f.WriteString(sb.String())
	return err
}

func CreateVersionFile(f *os.File) error {
	fmt.Printf("Initializing %s\n", f.Name())
	sb, writef := strutil.NewIndWritef("\t")
	writef(0, "package server")
	writef(0, "")
	writef(0, "const (")
	writef(1, "Version = \"v0.0.0\"")
	writef(0, ")")
	_, err := f.WriteString(sb.String())
	return err
}

func CreateWebFile(f *os.File) error {
	fmt.Printf("Initializing %s\n", f.Name())
	sb, writef := strutil.NewIndWritef("\t")
	writef(0, "package web")
	writef(0, "")
	writef(0, "import \"github.com/curtisnewbie/miso/miso\"")
	writef(0, "")
	writef(0, "func PrepareWebServer(rail miso.Rail) error {")
	writef(1, "// do not remove this if misoapi is used, make sure misoapi generated code is imported")
	writef(1, "return nil")
	writef(0, "}")
	_, err := f.WriteString(sb.String())
	return err
}

func CreateServerFile(f *os.File, cpltModName string, modName string) error {
	fmt.Printf("Initializing %s\n", f.Name())
	sb, writef := strutil.NewIndWritef("\t")
	writef(0, "package server")
	writef(0, "")
	writef(0, "import (")
	writef(1, "\"os\"")
	if !*DisableWebFlag {
		writef(1, "\"%s/internal/web\"", cpltModName)
	}
	writef(1, "")
	if *StaticFlag {
		writef(1, "\"%s/internal/static\"", cpltModName)
	}
	if *SvcFlag {
		writef(1, "\"%s/internal/schema\"", cpltModName)
	}
	writef(1, "\"github.com/curtisnewbie/miso/miso\"")
	writef(0, ")")
	writef(0, "")
	writef(0, "func init() {")
	writef(1, "")
	writef(1, "// default name of the app")
	writef(1, "miso.SetDefProp(miso.PropAppName, \"%v\")", modName)
	writef(1, "")
	writef(1, "// log app's name on startup")
	writef(1, "miso.PreServerBootstrap(func(rail miso.Rail) error {")
	writef(2, "rail.Infof(\"%%v version: %%v\", miso.GetPropStr(miso.PropAppName), Version)")
	writef(2, "return nil")
	writef(1, "})")
	writef(0, "}")
	writef(0, "")
	writef(0, "func BootstrapServer() {")
	if *SvcFlag {
		writef(1, "// automatic MySQL schema migration using svc")
		writef(1, "schema.EnableSchemaMigrate()")
	}
	if *StaticFlag {
		if *SvcFlag {
			writef(1, "")
		}
		writef(1, "// host static files, try 'http://%s:%s/static/miso.html'",
			miso.GetPropStr(miso.PropServerHost), miso.GetPropStr(miso.PropServerPort))
		writef(1, "static.PrepareWebStaticFs()")
		writef(1, "")
	}
	writef(1, "")
	writef(1, "// declare http endpoints, jobs/tasks, and other components here")
	if !*DisableWebFlag {
		writef(1, "miso.PreServerBootstrap(web.PrepareWebServer)")
	} else {
		writef(1, "miso.PreServerBootstrap()")
	}

	writef(1, "")
	writef(1, "// do stuff right after the server has been fully bootstrapped")
	writef(1, "miso.PostServerBootstrap()")
	writef(1, "")
	writef(1, "// boostrap server")
	writef(1, "miso.BootstrapServer(os.Args)")
	writef(0, "}")
	writef(0, "")
	_, err := f.WriteString(sb.String())
	return err
}
