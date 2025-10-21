package cli

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/src"
	"github.com/curtisnewbie/miso/util/strutil"
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

func Printf(pat string, args ...any) {
	fmt.Printf(pat, args...)
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
	cmd := exec.CommandContext(context.Background(), executable, args...)
	for _, op := range opts {
		op(cmd)
	}

	out, err = cmd.CombinedOutput()
	if err != nil {
		return out, errs.Wrap(err)
	}

	return out, nil
}

func NamedPrintlnf(pat string, p map[string]any) {
	println(strutil.NamedSprintf(pat, p))
}

type cliLogger struct {
	debug          *bool
	timePrefix     bool
	timePrefixCond func(level string) bool
	callerPrefix   bool
	callerCond     func(level string) bool
}

func (l *cliLogger) Infof(pat string, args ...any) {
	b := strings.Builder{}
	l.applyExtra(&b, "INFO")
	b.WriteString(fmt.Sprintf(pat+"\n", args...))
	print(b.String())
}

func (l *cliLogger) Debugf(pat string, args ...any) {
	if *l.debug {
		b := strings.Builder{}
		b.WriteString("[DEBUG] ")
		l.applyExtra(&b, "DEBUG")
		b.WriteString(fmt.Sprintf(pat+"\n", args...))
		print(b.String())
	}
}

func (l *cliLogger) Errorf(pat string, args ...any) {
	b := strings.Builder{}
	b.WriteString("[ERROR] ")
	l.applyExtra(&b, "ERROR")
	b.WriteString(fmt.Sprintf(pat+"\n", args...))
	print(b.String())
}

func (l *cliLogger) applyExtra(b *strings.Builder, level string) {
	if l.timePrefix && l.timePrefixCond(level) {
		b.WriteString(time.Now().Format("2006-01-02 15:04:05.000"))
		b.WriteRune(' ')
	}
	if l.callerPrefix && l.callerCond(level) {
		if l.timePrefix {
			b.WriteRune(' ')
		}
		b.WriteString(strutil.PadSpace(-30, src.GetCallerFnUpN(1)))
		b.WriteString(": ")
	}
}

func NewLog(op ...func(l *cliLogger)) *cliLogger {
	l := &cliLogger{}
	for _, f := range op {
		f(l)
	}
	if l.debug == nil {
		var v bool = false
		l.debug = &v
	}
	return l
}

func LogWithDebug(debug *bool) func(*cliLogger) {
	return func(l *cliLogger) {
		l.debug = debug
	}
}

func LogWithTime(cond ...func(level string) bool) func(*cliLogger) {
	return func(l *cliLogger) {
		l.timePrefixCond = slutil.VarArgAny(cond, func() func(level string) bool {
			return func(level string) bool {
				return true
			}
		})
		l.timePrefix = true
	}
}

func LogWithCaller(cond ...func(level string) bool) func(*cliLogger) {
	return func(l *cliLogger) {
		l.callerCond = slutil.VarArgAny(cond, func() func(level string) bool {
			return func(level string) bool {
				return true
			}
		})
		l.callerPrefix = true
	}
}
