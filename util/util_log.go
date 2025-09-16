package util

import "github.com/curtisnewbie/miso/util/cli"

var (
	DebugLog func(pat string, args ...any) = func(pat string, args ...any) {}
	ErrorLog func(pat string, args ...any) = func(pat string, args ...any) {
		cli.Printlnf("[Error] "+pat, args...)
	}
)
