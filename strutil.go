package gocommon

import "strings"

// Check if the string is empty
func IsEmpty(s *string) bool {
	if s == nil || strings.TrimSpace(*s) == "" {
		return true
	}
	return false
}

// Check if the string is empty
func IsStrEmpty(s string) bool {
	return s == "" || strings.TrimSpace(s) == ""
}

// Substring (rune)
func Substr(s string, start int, end int) string {
	r := []rune(s)
	l := len(r)

	if start >= l {
		return ""
	}

	if end > l {
		end = l
	}

	return string(r[start:end])
}

// Return len of string (rune)
func StrLen(s string) int {
	return len([]rune(s))
}
