package cli

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
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

func Printlnf(pat string, args ...any) {
	fmt.Printf(pat+"\n", args...)
}

func TPrintlnf(pat string, args ...any) {
	t := time.Now().Format("2006-01-02 15:04:05.000")
	fmt.Printf(t+" "+pat+"\n", args...)
}

func DebugPrintlnf(debug bool, pat string, args ...any) {
	if debug {
		fmt.Printf("[DEBUG] "+pat+"\n", args...)
	}
}

func ErrorPrintlnf(pat string, args ...any) {
	fmt.Printf("[ERROR] "+pat+"\n", args...)
}

type CliRail interface {
	Infof(format string, args ...interface{})
	Context() context.Context
}

type nilCliRailVal struct {
}

func (n nilCliRailVal) Infof(format string, args ...interface{}) {
	Printlnf(format, args...)
}

func (n nilCliRailVal) Context() context.Context {
	return context.Background()
}

func nilCliRail() CliRail {
	return nilCliRailVal{}
}

// Run python script.
//
// Python executable must be available beforehand.
//
// rail can be nil.
func RunPy(rail CliRail, pyExec string, pyContent string, args []string, opts ...func(*exec.Cmd)) (out []byte, err error) {
	if rail == nil {
		rail = nilCliRail()
	}

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
	return Run(rail, pyExec, args, opts...)
}

// Runs command.
//
// If err is not nil, out may still contain output from the command.
//
// rail can be nil.
func Run(rail CliRail, executable string, args []string, opts ...func(*exec.Cmd)) (out []byte, err error) {
	if rail == nil {
		rail = nilCliRail()
	}

	cmd := exec.CommandContext(rail.Context(), executable, args...)
	for _, op := range opts {
		op(cmd)
	}

	out, err = cmd.CombinedOutput()
	if err != nil {
		rail.Infof("Failed to execute command, %s", out)
		return out, err
	}

	return out, nil
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
