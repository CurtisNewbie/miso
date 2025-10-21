package main

import (
	_ "embed"
	"os"

	"github.com/curtisnewbie/miso/patch"
	"github.com/curtisnewbie/miso/util/cli"
	"github.com/curtisnewbie/miso/util/flags"
	"github.com/curtisnewbie/miso/util/osutil"
	"github.com/curtisnewbie/miso/version"
)

var (
	Debug = flags.Bool("debug", false, "enable debug log", false)
	log   = cli.NewLog(cli.LogWithDebug(Debug), cli.LogWithCaller(func(level string) bool { return level == "DEBUG" }))
)

func main() {
	flags.WithDescriptionBuilder(func(printlnf func(v string, args ...any)) {
		printlnf("misopatch - automatically apply gopatch on current working directory\n")
		printlnf("  miso build version: %v\n", version.Version)
	})
	flags.Parse()

	if err := checkGopatch(); err != nil {
		panic(err)
	}
	if err := applyPatches(); err != nil {
		panic(err)
	}
}

func checkGopatch() error {
	_, err := cli.Run(nil, "command", []string{"-v", "gopatch"})
	if err != nil {
		log.Infof("gopatch not found, installing")
		out, err := cli.Run(nil, "go", []string{"install", "github.com/uber-go/gopatch@latest"})
		if err != nil {
			log.Errorf("Install gopatch failed, output: '%s', %v", out, err)
			return err
		}
	}
	return nil
}

func runGopatch(path string) error {
	out, err := cli.Run(nil, "gopatch", []string{"-p", path, "./..."})
	if err != nil {
		log.Errorf("gopatch failed, %s, %v", out, err)
		return err
	}
	return nil
}

func applyPatches() error {
	entries, err := patch.Patches.ReadDir(".")
	if err != nil {
		return err
	}
	log.Infof("Found %v patches", len(entries))
	for _, et := range entries {
		f, err := osutil.NewTmpFileWith("patch")
		if err != nil {
			return err
		}
		defer os.Remove(f.Name())

		dat, err := patch.Patches.ReadFile(et.Name())
		if err != nil {
			return err
		}
		if _, err = f.Write(dat); err != nil {
			return err
		}
		if err := runGopatch(f.Name()); err != nil {
			return err
		}

		log.Infof("Applied patch: %v", et.Name())
	}
	return nil
}
