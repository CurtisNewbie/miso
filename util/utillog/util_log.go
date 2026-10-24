package utillog

import "fmt"

var (
	DebugLog func(pat string, args ...any) = func(pat string, args ...any) {}
	ErrorLog func(pat string, args ...any) = func(pat string, args ...any) {
		fmt.Printf("[Error] "+pat+"\n", args...)
	}
	WarnLog func(pat string, args ...any) = func(pat string, args ...any) {
		fmt.Printf("[Warn] "+pat+"\n", args...)
	}
)
