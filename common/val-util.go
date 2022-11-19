package common

import "strings"

const (
	BOOL_STR_TRUE  = "true"
	BOOL_STR_FALSE = "false"
)

func IsTrue(boolStr string) bool {
	return strings.ToLower(boolStr) == "true"
}

func IsBool(boolStr string) bool {
	ls := strings.ToLower(boolStr)
	return ls == "true" || ls == "false"
}
