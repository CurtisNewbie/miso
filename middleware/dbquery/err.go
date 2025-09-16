package dbquery

import "github.com/curtisnewbie/miso/util/errs"

const (
	ErrCodeRecordNotFound = "RECORD_NOT_FOUND"
)

var (
	ErrRecordNotFound = errs.NewErrfCode(ErrCodeRecordNotFound, "Record Not Found")
)
