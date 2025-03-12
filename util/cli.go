package util

import (
	"flag"
	"fmt"
	"os/exec"
	"runtime"
)

var CliErrLog func(pat string, args ...any) = Printlnf

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

// CLI runs command.
//
// If err is not nil, out may still contain output from the command.
func CliRun(ex string, args ...string) (out []byte, err error) {
	cmd := exec.Command(ex, args...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return out, err
	}
	return out, nil
}

// CLI runs command v2.
//
// If err is not nil, out may still contain output from the command.
func CliRunV2(dir string, env []string, ex string, args ...string) (out []byte, err error) {
	cmd := exec.Command(ex, args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Env, env...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return out, err
	}
	return out, nil
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
