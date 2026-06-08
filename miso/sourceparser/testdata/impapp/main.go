// Package impapp references constants from impconst for integration testing
// of sourceparser.ParseFileDst with *types.Package.
package impapp

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/miso/sourceparser/testdata/impconst"
)

const LocalURL = "/api/local"

func init() {
	miso.HttpGet(impconst.TestURL, miso.RawHandler(myHandler))
	miso.HttpGet(LocalURL, miso.RawHandler(myHandler2)).
		Desc(impconst.TestDesc).
		Resource(impconst.TestResource)
}

func myHandler(inb *miso.Inbound)  {}
func myHandler2(inb *miso.Inbound) {}
