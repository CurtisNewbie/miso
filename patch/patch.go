package patch

import "embed"

//go:embed *.patch
var Patches embed.FS
