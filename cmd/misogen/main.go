package main

import (
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/curtisnewbie/miso/middleware/mysql"
	"github.com/curtisnewbie/miso/middleware/rabbit"
	"github.com/curtisnewbie/miso/middleware/redis"
	"github.com/curtisnewbie/miso/middleware/sqlite"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
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
	flag.Parse()
	fmt.Printf("misogen, current miso version: %s\n\n", version.Version)

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

			iw := util.NewIndentWriter("\t")
			iw.Writef("package main")
			iw.Writef("")
			iw.Writef("import (").
				StepIn(func(iw *util.IndentWriter) {
					iw.Writef("\"flag\"")
					iw.Writef("")
					iw.Writef("\"github.com/curtisnewbie/miso/miso\"")
					iw.Writef("\"github.com/curtisnewbie/miso/middleware/redis\"")
					iw.Writef("\"github.com/curtisnewbie/miso/middleware/mysql\"")
				}).
				Writef(")").Writef("")

			iw.Writef("var (").
				StepIn(func(iw *util.IndentWriter) {
					iw.Writef("DebugFlag = flag.Bool(\"debug\", false, \"Enable debug log\")")
				}).
				Writef(")").Writef("")

			// main func
			iw.Writef("func main() {").
				StepIn(func(iw *util.IndentWriter) {
					iw.Writef("rail := miso.EmptyRail()")
					iw.Writef("flag.Parse()")
					iw.Writef("")
					iw.Writef("if *DebugFlag {").
						StepIn(func(iw *util.IndentWriter) {
							iw.Writef("miso.SetLogLevel(\"debug\")")
						}).
						Writef("}").Writef("")

					iw.Writef("// for mysql")
					iw.Writef("if err := mysql.InitMySQL(rail, mysql.MySQLConnParam{}); err != nil {").
						StepIn(func(iw *util.IndentWriter) {
							iw.Writef("panic(err)")
						}).
						Writef("}")

					iw.Writef("db := mysql.GetMySQL()")
					iw.Writef("if err := db.Exec(`SELECT 1`).Error; err != nil {").
						StepIn(func(iw *util.IndentWriter) {
							iw.Writef("panic(err)")
						}).
						Writef("}").Writef("")

					iw.Writef("// for redis")
					iw.Writef("red, err := redis.InitRedis(rail, redis.RedisConnParam{})")
					iw.Writef("if err != nil {").
						StepIn(func(iw *util.IndentWriter) {
							iw.Writef("panic(err)")
						}).
						Writef("}")

					iw.Writef("res, err := red.Ping().Result()")
					iw.Writef("if err != nil {").
						StepIn(func(iw *util.IndentWriter) {
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

		sb, writef := util.NewIndWritef("  ")

		writef(0, "# %s", ConfigDocUrl)
		writef(0, "")
		writef(0, "mode.production: \"%s\"", "false")
		writef(0, "app.name: \"%s\"", modName)

		writef(0, "")
		writef(0, "server: # http server")
		serverEnabled := miso.GetPropStr(miso.PropServerEnabled)
		if *DisableWebFlag {
			serverEnabled = "false"
		}
		writef(1, "enabled: \"%s\"", serverEnabled)
		writef(1, "host: \"%s\"", miso.GetPropStr(miso.PropServerHost))
		writef(1, "port: \"%s\"", miso.GetPropStr(miso.PropServerPort))

		writef(0, "")
		writef(0, "consul:")
		writef(1, "enabled: \"%s\"", miso.GetPropStr(miso.PropConsulEnabled))
		writef(1, "consulAddress: \"%s\"", miso.GetPropStr(miso.PropConsulAddress))

		writef(0, "")
		writef(0, "redis:")
		writef(1, "enabled: \"%s\"", miso.GetPropStr(redis.PropRedisEnabled))
		writef(1, "address: \"%s\"", miso.GetPropStr(redis.PropRedisAddress))
		writef(1, "port: \"%s\"", miso.GetPropStr(redis.PropRedisPort))
		writef(1, "username: \"%s\"", miso.GetPropStr(redis.PropRedisUsername))
		writef(1, "password: \"%s\"", miso.GetPropStr(redis.PropRedisPassword))
		writef(1, "database: \"%s\"", miso.GetPropStr(redis.PropRedisDatabase))

		writef(0, "")
		writef(0, "mysql:")
		writef(1, "enabled: \"%s\"", miso.GetPropStr(mysql.PropMySQLEnabled))
		writef(1, "host: \"%s\"", miso.GetPropStr(mysql.PropMySQLHost))
		writef(1, "port: \"%s\"", miso.GetPropStr(mysql.PropMySQLPort))
		writef(1, "user: \"%s\"", miso.GetPropStr(mysql.PropMySQLUser))
		writef(1, "password: \"%s\"", miso.GetPropStr(mysql.PropMySQLPassword))
		writef(1, "database: \"%s\"", guessSchemaName(modName))
		writef(1, "connection:")
		writef(2, "parameters:")
		for _, s := range []string{
			"charset=utf8mb4",
			"parseTime=True",
			"loc=Local",
			"readTimeout=30s",
			"writeTimeout=30s",
			"timeout=3s",
		} {
			writef(3, "- \"%s\"", s)
		}

		writef(0, "")
		writef(0, "sqlite:")
		writef(1, "file: \"%s\"", miso.GetPropStr(sqlite.PropSqliteFile))

		writef(0, "")
		writef(0, "rabbitmq:")
		writef(1, "enabled: \"%s\"", miso.GetPropStr(rabbit.PropRabbitMqEnabled))
		writef(1, "host: \"%s\"", miso.GetPropStr(rabbit.PropRabbitMqHost))
		writef(1, "port: \"%s\"", miso.GetPropStr(rabbit.PropRabbitMqPort))
		writef(1, "username: \"%s\"", miso.GetPropStr(rabbit.PropRabbitMqUsername))
		writef(1, "password: \"%s\"", miso.GetPropStr(rabbit.PropRabbitMqPassword))
		writef(1, "vhost: \"%s\"", miso.GetPropStr(rabbit.PropRabbitMqVhost))

		writef(0, "")
		writef(0, "logging:")
		writef(1, "level: \"%s\"", "info")
		writef(1, "# rolling:")
		writef(2, "# file: \"logs/%s.log\"", modName)

		if _, err := conf.WriteString(sb.String()); err != nil {
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

		sb, writef := util.NewIndWritef("\t")
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

		sb, writef := util.NewIndWritef("\t")
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

		sb, writef := util.NewIndWritef("\t")
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

		sb, writef := util.NewIndWritef("\t")
		writef(0, "package server")
		writef(0, "")
		writef(0, "import (")
		writef(1, "\"os\"")
		writef(1, "")
		if *StaticFlag {
			writef(1, "\"%s/internal/static\"", cpltModName)
		}
		if *SvcFlag {
			writef(1, "\"%s/internal/schema\"", cpltModName)
		}
		writef(1, "\"github.com/curtisnewbie/miso/miso\"")
		writef(0, ")")
		writef(0, "func init() {")
		writef(1, "miso.PreServerBootstrap(func(rail miso.Rail) error {")
		writef(2, "rail.Infof(\"%v version: %%v\", Version)", modName)
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
		writef(1, "miso.PreServerBootstrap(PreServerBootstrap)")
		writef(1, "miso.PostServerBootstrap(PostServerBootstrap)")
		writef(1, "miso.BootstrapServer(os.Args)")
		writef(0, "}")
		writef(0, "")
		writef(0, "func PreServerBootstrap(rail miso.Rail) error {")
		writef(1, "// declare http endpoints, jobs/tasks, and other components here")
		writef(1, "return nil")
		writef(0, "}")
		writef(0, "")
		writef(0, "func PostServerBootstrap(rail miso.Rail) error {")
		writef(1, "// do stuff right after server being fully bootstrapped")
		writef(1, "return nil")
		writef(0, "}")
		if _, err := serverf.WriteString(sb.String()); err != nil {
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

		sb, writef := util.NewIndWritef("\t")
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
}

func guessSchemaName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ToLower(name)
	return name
}
