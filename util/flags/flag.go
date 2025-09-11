package flags

import (
	"flag"
	"fmt"
	"os"
	"time"
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

func Float64(name string, value float64, usage string, required bool) *float64 {
	p := flag.Float64(name, value, usage)
	if required {
		requiredFlags[name] = struct{}{}
	}
	return p
}

func Int(name string, value int, usage string, required bool) *int {
	p := flag.Int(name, value, usage)
	if required {
		requiredFlags[name] = struct{}{}
	}
	return p
}

func Duration(name string, value time.Duration, usage string, required bool) *time.Duration {
	p := flag.Duration(name, value, usage)
	if required {
		requiredFlags[name] = struct{}{}
	}
	return p
}

func Bool(name string, value bool, usage string, required bool) *bool {
	p := flag.Bool(name, value, usage)
	if required {
		requiredFlags[name] = struct{}{}
	}
	return p
}

func String(name string, value string, usage string, required bool) *string {
	p := flag.String(name, value, usage)
	if required {
		requiredFlags[name] = struct{}{}
	}
	return p
}

type strSlice []string

func (s *strSlice) String() string {
	return fmt.Sprintf("%v", []string(*s))
}

func (s *strSlice) Set(t string) error {
	*s = append(*s, t)
	return nil
}

func StrSlice(name string, usage string, required bool) *strSlice {
	p := new(strSlice)
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
				fmt.Printf("\n%s\n", extra)
			}
		}
	}

	flag.Parse()
	m := visited()
	for name := range requiredFlags {
		if _, ok := m[name]; !ok {
			fmt.Printf("Arg '%v' is required \n", name)
			flag.Usage()
			os.Exit(2)
		}
	}
}
