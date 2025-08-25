package util

import (
	"context"
	"flag"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func TermOpenUrl(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

type StrSliceFlag []string

func (s *StrSliceFlag) String() string {
	return fmt.Sprintf("%v", []string(*s))
}

func (s *StrSliceFlag) Set(t string) error {
	*s = append(*s, t)
	return nil
}

func FlagStrSlice(name string, usage string) *StrSliceFlag {
	p := new(StrSliceFlag)
	flag.Var(p, name, usage)
	return p
}

func Printlnf(pat string, args ...any) {
	fmt.Printf(pat+"\n", args...)
}

func TPrintlnf(pat string, args ...any) {
	t := Now().FormatStdMilli()
	fmt.Printf(t+" "+pat+"\n", args...)
}

func DebugPrintlnf(debug bool, pat string, args ...any) {
	if debug {
		fmt.Printf("[DEBUG] "+pat+"\n", args...)
	}
}

func NamedPrintlnf(pat string, p map[string]any) {
	println(NamedSprintf(pat, p))
}

func DebugNamedPrintlnf(debug bool, pat string, p map[string]any) {
	if debug {
		println(NamedSprintf("[DEBUG] "+pat, p))
	}
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func MustGet[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}

// Run python script.
//
// Python executable must be available beforehand.
func RunPyScript(pyExec string, pyContent string, args []string, opts ...func(*exec.Cmd)) (out []byte, err error) {
	// '-' tells python to read from stdin
	if len(args) < 1 {
		args = append(args, "")
		copy(args[1:], args)
		args[0] = "-"
	} else if args[0] != "-" {
		args = append(args, "-")
	}
	// remove '-' flag in script
	pyContent = "import sys\nsys.argv = sys.argv[1:]\n" + pyContent

	opts = append(opts, func(c *exec.Cmd) {
		c.Stdin = strings.NewReader(pyContent)
	})
	return ExecCmd(pyExec, args, opts...)
}

// CLI runs command.
//
// If err is not nil, out may still contain output from the command.
func ExecCmd(executable string, args []string, opts ...func(*exec.Cmd)) (out []byte, err error) {
	cmd := exec.Command(executable, args...)
	for _, op := range opts {
		op(cmd)
	}

	out, err = cmd.CombinedOutput()
	if err != nil {
		return out, err
	}
	return out, nil
}

// CLI runs command.
//
// If err is not nil, out may still contain output from the command.
func CliRun(rail interface {
	Infof(format string, args ...interface{})
	Context() context.Context
}, executable string, args []string, opts ...func(*exec.Cmd)) (out []byte, err error) {

	cmd := exec.CommandContext(rail.Context(), executable, args...)
	for _, op := range opts {
		op(cmd)
	}
	rail.Infof("Executing Command: %v", cmd)

	out, err = cmd.CombinedOutput()
	if err != nil {
		rail.Infof("Failed to execute command, %s", out)
		return out, err
	}

	return out, nil
}
