package flags

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/curtisnewbie/miso/util/strutil"
)

var (
	requiredFlags = map[string]struct{}{}
	description   string
	extra         string
)

type FlagInfo struct {
	Ptr      any
	Name     string
	Required bool
	Set      bool
}

func updateUsage(usage string, required bool) string {
	if required {
		usage = strings.TrimSpace(usage)
		if usage != "" {
			usage = usage + ". "
		}
		usage = usage + "Required."
	}
	return usage
}

func Float64(name string, value float64, usage string, required bool) *float64 {
	p := flag.Float64(name, value, updateUsage(usage, required))
	if required {
		requiredFlags[name] = struct{}{}
	}
	return p
}

func Int(name string, value int, usage string, required bool) *int {
	p := flag.Int(name, value, updateUsage(usage, required))
	if required {
		requiredFlags[name] = struct{}{}
	}
	return p
}

func Duration(name string, value time.Duration, usage string, required bool) *time.Duration {
	p := flag.Duration(name, value, updateUsage(usage, required))
	if required {
		requiredFlags[name] = struct{}{}
	}
	return p
}

func Bool(name string, value bool, usage string, required bool) *bool {
	p := flag.Bool(name, value, updateUsage(usage, required))
	if required {
		requiredFlags[name] = struct{}{}
	}
	return p
}

// boolVal is a flag.Value that requires an explicit "true" or "false" argument
// (e.g. -flag true, -flag=false) rather than the standard Go bool flag behavior
// where -flag alone implicitly sets the value to true.
type boolVal struct{ b *bool }

func (v *boolVal) String() string {
	if v.b == nil {
		return "false"
	}
	return strconv.FormatBool(*v.b)
}
func (v *boolVal) Set(s string) error {
	t, err := strconv.ParseBool(s)
	if err != nil {
		return fmt.Errorf("invalid bool value %q for flag, must be true or false", s)
	}
	*v.b = t
	return nil
}
func (v *boolVal) IsBoolFlag() bool { return false }

// BoolVal returns a *bool that requires an explicit "true"/"false" argument.
// -flag alone is an error; use -flag true or -flag=false instead.
func BoolVal(name string, value bool, usage string, required bool) *bool {
	b := &boolVal{b: &value}
	flag.Var(b, name, updateUsage(usage, required))
	if required {
		requiredFlags[name] = struct{}{}
	}
	return &value
}

func String(name string, value string, usage string, required bool) *string {
	p := flag.String(name, value, updateUsage(usage, required))
	if required {
		requiredFlags[name] = struct{}{}
	}
	return p
}

type StrSliceFlag []string

func (s *StrSliceFlag) String() string {
	return fmt.Sprintf("%v", []string(*s))
}

func (s *StrSliceFlag) Set(t string) error {
	*s = append(*s, t)
	return nil
}

func StrSlice(name string, usage string, required bool) *StrSliceFlag {
	p := new(StrSliceFlag)
	flag.Var(p, name, usage)
	if required {
		requiredFlags[name] = struct{}{}
	}
	return p
}

func visited() map[string]struct{} {
	m := map[string]struct{}{}
	flag.Visit(func(f *flag.Flag) {
		m[f.Name] = struct{}{}
	})
	return m
}

func WithDescriptionBuilder(f func(printlnf func(v string, args ...any))) {
	b := &strings.Builder{}
	f(func(v string, args ...any) {
		b.WriteString(fmt.Sprintf(v+"\n", args...))
	})
	WithDescription(b.String())
}

func WithExtraBuilder(f func(printlnf func(v string, args ...any))) {
	b := &strings.Builder{}
	f(func(v string, args ...any) {
		b.WriteString(fmt.Sprintf(v+"\n", args...))
	})
	WithExtra(b.String())
}

func WithDescription(s string) {
	description = s
}

func WithExtra(s string) {
	extra = s
}

func Parse() {
	if description != "" || extra != "" {
		flag.Usage = func() {
			if description != "" {
				fmt.Printf("\n%s\n", description)
			}
			fmt.Printf("Usage of %s:\n", os.Args[0])
			flag.PrintDefaults()
			if extra != "" {
				fmt.Printf("\n%s", strutil.TrimSpaceRight(extra))
			}
			fmt.Print("\n\n")
		}
	}

	flag.Parse()
	m := visited()
	for name := range requiredFlags {
		if _, ok := m[name]; !ok {
			fmt.Printf("Arg '%v' is required \n\n", name)
			flag.Usage()
			os.Exit(2)
		}
	}
}
