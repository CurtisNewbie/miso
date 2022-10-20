package util

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
