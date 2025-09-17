package main

import (
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/util/cli"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/curtisnewbie/miso/version"
)

const (
	ModFile      = "go.mod"
	ConfFile     = "conf.yml"
	MainFile     = "main.go"
	ConfigDocUrl = "https://github.com/CurtisNewbie/miso/blob/main/doc/config.md"
)

var (
	SchemaDir               = filepath.Join("internal", "schema", "scripts")
	SchemaFile              = filepath.Join(SchemaDir, "schema.sql")
	SchemaMigrateFile       = filepath.Join("internal", "schema", "migrate.go")
	ServerFile              = filepath.Join("internal", "server", "server.go")
	WebFile                 = filepath.Join("internal", "web", "web.go")
	ConfigFile              = filepath.Join("internal", "config", "prop.go")
	VersionFile             = filepath.Join("internal", "server", "version.go")
	StaticFsFile            = filepath.Join("internal", "static", "static.go")
	StaticFsDir             = filepath.Join("internal", "static", "static")
	StaticFsPlaceholderFile = filepath.Join("internal", "static", "static", "miso.html")
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

	// cli style project
	if *CliFlag {
		fmt.Printf("Initializing %s\n", MainFile)
		// main.go
		{
			mainf, err := util.ReadWriteFile(MainFile)
			if err != nil {
				panic(fmt.Errorf("failed to create file %s, %v", MainFile, err))
			}

			iw := strutil.NewIndentWriter("\t")
			iw.Writef("package main")
			iw.Writef("")
			iw.Writef("import (").
				StepIn(func(iw *strutil.IndentWriter) {
					iw.Writef("\"flag\"")
					iw.Writef("")
					iw.Writef("\"github.com/curtisnewbie/miso/miso\"")
					iw.Writef("\"github.com/curtisnewbie/miso/middleware/redis\"")
					iw.Writef("\"github.com/curtisnewbie/miso/middleware/mysql\"")
				}).
				Writef(")").Writef("")

			iw.Writef("var (").
				StepIn(func(iw *strutil.IndentWriter) {
					iw.Writef("DebugFlag = flag.Bool(\"debug\", false, \"Enable debug log\")")
				}).
				Writef(")").Writef("")

			// main func
			iw.Writef("func main() {").
				StepIn(func(iw *strutil.IndentWriter) {
					iw.Writef("rail := miso.EmptyRail()")
					iw.Writef("flag.Parse()")
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

					iw.Writef("res, err := red.Ping().Result()")
					iw.Writef("if err != nil {").
						StepIn(func(iw *strutil.IndentWriter) {
							iw.Writef("panic(err)")
						}).
						Writef("}")
					iw.Writef("rail.Infof(\"Ping result: %%v\", res)")

				}).
				Writef("}")

			if _, err := mainf.WriteString(iw.String()); err != nil {
				panic(err)
			}
		}
		if err := exec.Command("go", "mod", "tidy").Run(); err != nil {
			panic(err)
		}
		return
	}

	// conf.yml
	fmt.Printf("Initializing %s\n", ConfFile)
	if ok, err := util.FileExists(ConfFile); err != nil || !ok {
		if err != nil {
			panic(fmt.Errorf("failed to open file %s, %v", ConfFile, err))
		}

		conf, err := util.ReadWriteFile(ConfFile)
		if err != nil {
			panic(fmt.Errorf("failed to open file %s, %v", ConfFile, err))
		}

		s := `# https://github.com/CurtisNewbie/miso/blob/main/doc/config.md

# To use following middlewares, make sure you have already imported relevant go module

# production mode, must be true in production.
mode.production: false

# http server
server:
  enabled: false
  host: 127.0.0.1
  port: 8080
  request-log:
    enabled: false
  health-check-url: "/health"
  log-routes: false
  graceful-shutdown-time-sec: 30
  auth:
    bearer: "" # bearer token for all api (including pprof)
  pprof:
    enabled: false
    auth:
      bearer: "" # bearer token for pprof api
  api-doc:
    enabled: true
    file: "" # generated api doc file
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

		if _, err := conf.WriteString(s); err != nil {
			panic(err)
		}
	}

	// for svc
	if *SvcFlag {
		fmt.Printf("Initializing %s\n", SchemaFile)
		util.MkdirAll(SchemaDir)
		sf, err := util.ReadWriteFile(SchemaFile)
		if err != nil {
			panic(err)
		}
		sf.WriteString("-- Initialize schema")
		sf.Close()

		fmt.Printf("Initializing %s\n", SchemaMigrateFile)
		mf, err := util.ReadWriteFile(SchemaMigrateFile)
		if err != nil {
			panic(err)
		}

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
		mf.WriteString(sb.String())
		mf.Close()
	}

	// for self-hosted frontend stuff
	if *StaticFlag {
		util.MkdirParentAll(StaticFsFile)
		fmt.Printf("Initializing %s\n", StaticFsFile)
		f, err := util.ReadWriteFile(StaticFsFile)
		if err != nil {
			panic(err)
		}

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
		f.WriteString(sb.String())
		f.Close()

		util.MkdirParentAll(StaticFsPlaceholderFile)
		fmt.Printf("Initializing %s\n", StaticFsPlaceholderFile)
		f, err = util.ReadWriteFile(StaticFsPlaceholderFile)
		if err != nil {
			panic(err)
		}
		f.WriteString(fmt.Sprintf("Powered by miso %s.", version.Version))
		f.Close()
	}

	// version.go
	{
		if err := util.MkdirParentAll(VersionFile); err != nil {
			panic(fmt.Errorf("failed to make parent dir for file %s, %v", VersionFile, err))
		}
		vf, err := util.ReadWriteFile(VersionFile)
		if err != nil {
			panic(fmt.Errorf("failed to create file %s, %v", VersionFile, err))
		}

		sb, writef := strutil.NewIndWritef("\t")
		writef(0, "package server")
		writef(0, "")
		writef(0, "const (")
		writef(1, "Version = \"v0.0.0\"")
		writef(0, ")")
		vf.WriteString(sb.String())
		vf.Close()
	}

	// server.go
	{
		if err := util.MkdirParentAll(ServerFile); err != nil {
			panic(fmt.Errorf("failed to make parent dir for file %s, %v", ServerFile, err))
		}
		serverf, err := util.ReadWriteFile(ServerFile)
		if err != nil {
			panic(fmt.Errorf("failed to create file %s, %v", ServerFile, err))
		}

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
		if _, err := serverf.WriteString(sb.String()); err != nil {
			panic(err)
		}
		serverf.Close()
	}

	// web.go
	if !*DisableWebFlag {
		if err := util.MkdirParentAll(WebFile); err != nil {
			panic(fmt.Errorf("failed to make parent dir for file %s, %v", WebFile, err))
		}
		webf, err := util.ReadWriteFile(WebFile)
		if err != nil {
			panic(fmt.Errorf("failed to create file %s, %v", WebFile, err))
		}
		defer webf.Close()

		sb, writef := strutil.NewIndWritef("\t")
		writef(0, "package web")
		writef(0, "")
		writef(0, "import \"github.com/curtisnewbie/miso/miso\"")
		writef(0, "")
		writef(0, "func PrepareWebServer(rail miso.Rail) error {")
		writef(1, "return nil")
		writef(0, "}")
		if _, err := webf.WriteString(sb.String()); err != nil {
			panic(err)
		}
	}

	// config.go
	{
		if err := util.MkdirParentAll(ConfigFile); err != nil {
			panic(fmt.Errorf("failed to make parent dir for file %s, %v", ConfigFile, err))
		}
		configf, err := util.ReadWriteFile(ConfigFile)
		if err != nil {
			panic(fmt.Errorf("failed to create file %s, %v", ConfigFile, err))
		}
		defer configf.Close()

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

		if _, err := configf.WriteString(sb.String()); err != nil {
			panic(err)
		}
	}

	fmt.Printf("Initializing %s\n", MainFile)
	// main.go
	{
		mainf, err := util.ReadWriteFile(MainFile)
		if err != nil {
			panic(fmt.Errorf("failed to create file %s, %v", MainFile, err))
		}

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
		if _, err := mainf.WriteString(sb.String()); err != nil {
			panic(err)
		}
	}

	if err := exec.Command("go", "mod", "tidy").Run(); err != nil {
		panic(err)
	}

	if err := exec.Command("go", "fmt", "./...").Run(); err != nil {
		cli.Printlnf("failed to fmt source code, %v", err)
	}
}
