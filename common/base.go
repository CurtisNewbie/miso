package common

const (
	// record is deleted
	IS_DEL_Y IS_DEL = 1
	// record is not deleted
	IS_DEL_N IS_DEL = 0
)

type IS_DEL int8

// Check if the record is deleted
func IsDeleted(isDel IS_DEL) bool {
	return isDel == IS_DEL_Y
}
