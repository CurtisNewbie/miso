package dbquery

import "github.com/curtisnewbie/miso/miso"

const (
	ErrCodeRecordNotFound = "RECORD_NOT_FOUND"
)

var (
	ErrRecordNotFound = miso.NewErrfCode(ErrCodeRecordNotFound, "Record Not Found")
)
