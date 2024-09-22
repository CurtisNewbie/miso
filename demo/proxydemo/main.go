package main

import (
	"os"

	"github.com/curtisnewbie/miso/miso"
)

func init() {
	miso.SetProp("app.name", "demo")
}

func main() {

	_ = miso.NewHttpProxy("/proxy/*proxyPath", func(relPath string) (string, error) {
		return "http://localhost:8081" + relPath, nil
	})

	miso.BootstrapServer(os.Args)
}
