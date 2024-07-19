package util

import (
	"flag"
	"fmt"
	"os/exec"
	"runtime"
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

func CliRun(ex string, args ...string) ([]byte, error) {
	cmd := exec.Command(ex, args...)
	cmdout, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return cmdout, nil
}

func Printlnf(pat string, args ...any) {
	fmt.Printf(pat+"\n", args...)
}

func NamedPrintlnf(pat string, p map[string]any) {
	println(NamedSprintf(pat, p))
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}
