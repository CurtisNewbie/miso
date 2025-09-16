package util

import (
	"os/exec"

	"github.com/curtisnewbie/miso/util/cli"
	"github.com/curtisnewbie/miso/util/flags"
)

// Deprecated: Migrate to cli pkg. Will be removed in v0.2.19.
var (
	TermOpenUrl   = cli.TermOpenUrl
	Printlnf      = cli.Printlnf
	TPrintlnf     = cli.TPrintlnf
	DebugPrintlnf = cli.DebugPrintlnf
	CliRun        = cli.Run
	Must          = cli.Must
)

// Deprecated: Use [flags.StrSliceFlag] instead. Will be removed in v0.2.19.
type StrSliceFlag = flags.StrSliceFlag

// Deprecated: Use [flags.StrSlice] instead. Will be removed in v0.2.19.
func FlagStrSlice(name string, usage string) *StrSliceFlag {
	return flags.StrSlice(name, usage, false)
}

// Run python script.
//
// Python executable must be available beforehand.
//
// Deprecated: Use [cli.RunPy] instead. Will be removed in v0.2.19.
func RunPyScript(pyExec string, pyContent string, args []string, opts ...func(*exec.Cmd)) (out []byte, err error) {
	return cli.RunPy(nil, pyExec, pyContent, args, opts...)
}

// CLI runs command.
//
// If err is not nil, out may still contain output from the command.
//
// Deprecated: Use [cli.Run] instead. Will be removed in v0.2.19.
func ExecCmd(executable string, args []string, opts ...func(*exec.Cmd)) (out []byte, err error) {
	return cli.Run(nil, executable, args, opts...)
}

// Deprecated: Use [cli.MustGet] instead. Will be removed in v0.2.19.
func MustGet[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
