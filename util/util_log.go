package util

var (
	DebugLog func(pat string, args ...any) = func(pat string, args ...any) {}
	ErrorLog func(pat string, args ...any) = func(pat string, args ...any) {
		Printlnf("[Error] "+pat, args...)
	}
)
