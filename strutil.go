package gocommon

import "strings"

// Wrapper of Rune
type RuneWrp struct {
	c []rune
}

// Get length in terms of characters
func (rw RuneWrp) Len() int {
	return len(rw.c)
}

// Get string at index
func (rw RuneWrp) StrAt(idx int) string {
	return string(rw.c[idx])
}

// Get substring
func (rw RuneWrp) Substr(start int, end int) string {
	return string(rw.c[start:end])
}

// Check if the string is empty
func IsEmpty(s *string) bool {
	if s == nil || strings.TrimSpace(*s) == "" {
		return true
	}
	return false
}

// Get RuneWrp from string
func GetRuneWRp(s string) RuneWrp {
	return RuneWrp{c: []rune(s)}
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

// Get last char
func LastChar(s string) (length int, lastChar string) {
	rs := []rune(s)
	l := len(rs)

	// empty string
	if l < 1 {
		return l, ""
	}

	return l, string(rs[l-1])
}
