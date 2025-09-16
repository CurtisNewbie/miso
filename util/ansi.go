package util

import "github.com/curtisnewbie/miso/util/cli"

// Deprecated: Use cli pkg instead. Will be removed in v0.2.19.
//
//lint:ignore U1000 ansi color code, keep it for future use
const (
	ANSIRed   = cli.ANSIRed
	ANSIGreen = cli.ANSIGreen
	ANSICyan  = cli.ANSICyan
	ANSIReset = cli.ANSIReset

	ANSIBlinkRed   = cli.ANSIBlinkRed
	ANSIBlinkGreen = cli.ANSIBlinkGreen
	ANSIBlinkCyan  = cli.ANSIBlinkCyan
)
