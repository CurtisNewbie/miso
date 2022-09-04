package util

import "strings"

// Check if the string is empty
func IsEmpty(s *string) bool {
	if s == nil || strings.TrimSpace(*s) == "" {
		return true
	}
	return false
}
