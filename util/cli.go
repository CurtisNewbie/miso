package util

import (
	"os/exec"

	"github.com/curtisnewbie/miso/util/cli"
	"github.com/curtisnewbie/miso/util/flags"
)

// Deprecated: Since v0.2.17, migrate to cli pkg.
var (
	TermOpenUrl   = cli.TermOpenUrl
	Printlnf      = cli.Printlnf
	TPrintlnf     = cli.TPrintlnf
	DebugPrintlnf = cli.DebugPrintlnf
	CliRun        = cli.Run
	Must          = cli.Must
)

// Deprecated: Since v0.2.17, migrate to [flags.StrSliceFlag].
type StrSliceFlag = flags.StrSliceFlag

// Deprecated: Since v0.2.17, migrate to [flags.StrSlice].
func FlagStrSlice(name string, usage string) *StrSliceFlag {
	return flags.StrSlice(name, usage, false)
}

// Run python script.
//
// Python executable must be available beforehand.
//
// Deprecated: Since v0.2.17, migrate to [cli.RunPy].
func RunPyScript(pyExec string, pyContent string, args []string, opts ...func(*exec.Cmd)) (out []byte, err error) {
	return cli.RunPy(nil, pyExec, pyContent, args, opts...)
}

// CLI runs command.
//
// If err is not nil, out may still contain output from the command.
//
// Deprecated: Since v0.2.17, migrate to [cli.Run].
func ExecCmd(executable string, args []string, opts ...func(*exec.Cmd)) (out []byte, err error) {
	return cli.Run(nil, executable, args, opts...)
}

// Deprecated: Since v0.2.17, migrate to [cli.MustGet].
func MustGet[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
