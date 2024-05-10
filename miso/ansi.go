package miso

// https://stackoverflow.com/questions/8357203/is-it-possible-to-display-text-in-a-console-with-a-strike-through-effect
// https://en.wikipedia.org/wiki/ANSI_escape_code
const (
	//lint:ignore U1000 ansi color code, keep it for future use
	ANSIRed   = "\033[1;31m"
	ANSIGreen = "\033[1;32m"
	ANSICyan  = "\033[1;36m"
	ANSIReset = "\033[0m"
)
