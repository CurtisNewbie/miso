package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/curtisnewbie/miso/miso"
)

const (
	ModFile  = "go.mod"
	ConfFile = "conf.yml"
	MainFile = "main.go"
)

var (
	SchemaDir         = filepath.Join("internal", "schema", "scripts")
	SchemaFile        = filepath.Join(SchemaDir, "schema.sql")
	SchemaMigrateFile = filepath.Join("internal", "schema", "migrate.go")
)

func main() {

	fmt.Printf("misogen, current miso version: %s\n\n", miso.Version)

	var initName string
	args := os.Args
	if len(args) > 1 {
		initName = strings.TrimSpace(args[1])
	}

	if ok, err := miso.FileExists(ModFile); err != nil || !ok {
		if err != nil {
			panic(err)
		}
		if !ok {
			if initName == "" {
				fmt.Printf("File %s not found\n\n", ModFile)
				fmt.Println("Create new module yourself (go mod init), or run misogen with your module name\ne.g.,\n\tmisogen github.com/curtisnewbie/applejuice")
				return
			}
			out, err := exec.Command("go", "mod", "init", initName).CombinedOutput()
			if err != nil {
				fmt.Println(miso.UnsafeByt2Str(out))
				panic(err)
			}
			fmt.Printf("Initialized module '%s'\n", initName)
		}
	}

	modName := ""
	modfCtn, err := miso.ReadFileAll(ModFile)
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
				line := miso.UnsafeByt2Str(modfCtn[i:j])
				line = strings.TrimSpace(line)
				if n, ok := strings.CutPrefix(line, "module"); ok {
					modName = strings.TrimSpace(n)
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

	pkg := fmt.Sprintf("github.com/curtisnewbie/miso/miso@%s", miso.Version)
	fmt.Printf("Installing dependency: %s\n", pkg)

	out, err := exec.Command("go", "get", "-x", pkg).CombinedOutput()
	if err != nil {
		fmt.Println(miso.UnsafeByt2Str(out))
		panic(fmt.Errorf("failed to install miso, %v", err))
	}

	// os.MkdirAll("cmd", 0755)

	fmt.Printf("Initializing %s\n", ConfFile)
	if ok, err := miso.FileExists(ConfFile); err != nil || !ok {
		if err != nil {
			panic(fmt.Errorf("failed to open file %s, %v", ConfFile, err))
		}

		conf, err := miso.ReadWriteFile(ConfFile)
		if err != nil {
			panic(fmt.Errorf("failed to open file %s, %v", ConfFile, err))
		}

		sb, writef := NewWritef("  ")

		writef(0, "mode.production: \"%s\"", miso.GetPropStr(miso.PropProdMode))
		writef(0, "app.name: \"%s\"", modName)

		writef(0, "")
		writef(0, "server: # http server")
		writef(1, "enabled: \"%s\"", miso.GetPropStr(miso.PropServerEnabled))
		writef(1, "host: \"%s\"", miso.GetPropStr(miso.PropServerHost))
		writef(1, "port: \"%s\"", miso.GetPropStr(miso.PropServerPort))

		writef(0, "")
		writef(0, "consul:")
		writef(1, "enabled: \"%s\"", miso.GetPropStr(miso.PropConsulEnabled))
		writef(1, "consulAddress: \"%s\"", miso.GetPropStr(miso.PropConsulAddress))

		writef(0, "")
		writef(0, "redis:")
		writef(1, "enabled: \"%s\"", miso.GetPropStr(miso.PropRedisEnabled))
		writef(1, "address: \"%s\"", miso.GetPropStr(miso.PropRedisAddress))
		writef(1, "port: \"%s\"", miso.GetPropStr(miso.PropRedisPort))
		writef(1, "username: \"%s\"", miso.GetPropStr(miso.PropRedisUsername))
		writef(1, "password: \"%s\"", miso.GetPropStr(miso.PropRedisPassword))
		writef(1, "database: \"%s\"", miso.GetPropStr(miso.PropRedisDatabase))

		writef(0, "")
		writef(0, "mysql:")
		writef(1, "enabled: \"%s\"", miso.GetPropStr(miso.PropMySQLEnabled))
		writef(1, "host: \"%s\"", miso.GetPropStr(miso.PropMySQLHost))
		writef(1, "port: \"%s\"", miso.GetPropStr(miso.PropMySQLPort))
		writef(1, "user: \"%s\"", miso.GetPropStr(miso.PropMySQLUser))
		writef(1, "password: \"%s\"", miso.GetPropStr(miso.PropMySQLPassword))
		writef(1, "database: \"%s\"", miso.GetPropStr(miso.PropMySQLSchema))

		writef(0, "")
		writef(0, "sqlite:")
		writef(1, "file: \"%s\"", miso.GetPropStr(miso.PropSqliteFile))

		writef(0, "")
		writef(0, "rabbitmq:")
		writef(1, "enabled: \"%s\"", miso.GetPropStr(miso.PropRabbitMqEnabled))
		writef(1, "host: \"%s\"", miso.GetPropStr(miso.PropRabbitMqHost))
		writef(1, "port: \"%s\"", miso.GetPropStr(miso.PropRabbitMqPort))
		writef(1, "username: \"%s\"", miso.GetPropStr(miso.PropRabbitMqUsername))
		writef(1, "password: \"%s\"", miso.GetPropStr(miso.PropRabbitMqPassword))
		writef(1, "vhost: \"%s\"", miso.GetPropStr(miso.PropRabbitMqVhost))

		writef(0, "")
		writef(0, "logging:")
		writef(1, "level: \"%s\"", "info")
		writef(1, "# rolling:")
		writef(2, "# file: \"logs/%s.log\"", modName)

		if _, err := conf.WriteString(sb.String()); err != nil {
			panic(err)
		}
	}

	fmt.Printf("Initializing %s\n", MainFile)
	ok, err := miso.FileExists(MainFile)
	if err != nil {
		panic(fmt.Errorf("failed to open file %s, %v", MainFile, err))
	}
	if ok {
		panic(fmt.Sprintf("%s already exists", MainFile))
	}

	mainf, err := miso.ReadWriteFile(MainFile)
	if err != nil {
		panic(fmt.Errorf("failed to create file %s, %v", MainFile, err))
	}

	sb, writef := NewWritef("\t")

	writef(0, "package main")
	writef(0, "")
	writef(0, "import (")
	writef(1, "\"os\"")
	writef(1, "")
	writef(1, "\"github.com/curtisnewbie/miso/miso\"")
	writef(0, ")")
	writef(0, "")
	writef(0, "func main() {")
	writef(1, "miso.PreServerBootstrap(func(rail miso.Rail) error {")
	writef(2, "// TODO: declare http endpoints, jobs/tasks, and other components here")
	writef(2, "return nil")
	writef(1, "})")
	writef(1, "")
	writef(1, "miso.PostServerBootstrapped(func(rail miso.Rail) error {")
	writef(2, "// TODO: do stuff right after server being fully bootstrapped")
	writef(2, "return nil")
	writef(1, "})")
	writef(1, "")
	writef(1, "miso.BootstrapServer(os.Args)")
	writef(0, "}")
	if _, err := mainf.WriteString(sb.String()); err != nil {
		panic(err)
	}

	// for svc
	{
		fmt.Printf("Initializing %s\n", SchemaFile)
		miso.MkdirAll(SchemaDir)
		sf, err := miso.ReadWriteFile(SchemaFile)
		if err != nil {
			panic(err)
		}
		sf.WriteString("-- Initialize schema")
		sf.Close()

		fmt.Printf("Initializing %s\n", SchemaMigrateFile)
		mf, err := miso.ReadWriteFile(SchemaMigrateFile)
		if err != nil {
			panic(err)
		}

		sb, writef := NewWritef("\t")
		writef(0, "// This is for automated MySQL schema migration.")
		writef(0, "//")
		writef(0, "// See https://github.com/CurtisNewbie/svc for more information.")
		writef(0, "package schema")
		writef(0, "")
		writef(0, "import (")
		writef(1, "\"embed\"")
		writef(1, "")
		writef(1, "\"github.com/curtisnewbie/miso/middleware/svc/migrate\"")
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
		writef(0, "func EnableSchemaMigrateOnProd() {")
		writef(1, "migrate.EnableSchemaMigrateOnProd(schemaFs, BaseDir, \"\")")
		writef(0, "}")
		mf.WriteString(sb.String())
		mf.Close()
	}

	if err := exec.Command("go", "mod", "tidy").Run(); err != nil {
		panic(err)
	}
}

func NewWritef(indentc string) (*strings.Builder, func(idt int, pat string, args ...any)) {
	sb := strings.Builder{}
	return &sb, func(idt int, pat string, args ...any) {
		sb.WriteString(strings.Repeat(indentc, idt) + fmt.Sprintf(pat+"\n", args...))
	}
}
