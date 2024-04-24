package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/curtisnewbie/miso/miso"
)

const (
	ModFile  = "go.mod"
	ConfFile = "conf.yml"
	MainFile = "main.go"
)

func main() {
	if ok, err := miso.FileExists(ModFile); err != nil || !ok {
		if err != nil {
			panic(err)
		}
		if !ok {
			panic(fmt.Sprintf("File %s not found", ModFile))
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
						modName = modName[k:]
					}
					break
				}
				i = j + 1
			}
		}
	}

	pkg := "github.com/curtisnewbie/miso/miso@latest"
	fmt.Printf("Installing dependency: %s\n", pkg)

	out, err := exec.Command("go", "get", "-x", pkg).CombinedOutput()
	if err != nil {
		fmt.Println(miso.UnsafeByt2Str(out))
		panic(err)
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

		sb := strings.Builder{}
		writef := func(idt int, pat string, args ...any) {
			sb.WriteString(strings.Repeat("  ", idt) + fmt.Sprintf(pat+"\n", args...))
		}

		writef(0, "mode.production: \"%s\"", miso.GetPropStr(miso.PropProdMode))
		writef(0, "app.name: \"%s\" # TODO extracted from module name, change this if necessary", modName)

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

	sb := strings.Builder{}
	writef := func(idt int, pat string, args ...any) {
		sb.WriteString(strings.Repeat("\t", idt) + fmt.Sprintf(pat+"\n", args...))
	}

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

	if err := exec.Command("go", "mod", "tidy").Run(); err != nil {
		panic(err)
	}
}
