package semver

import (
	"strings"

	"github.com/spf13/cast"
)

const (
	VerSep = "."
)

// Check if ver1 is eq to ver2.
func VerEq(ver1 string, ver2 string) bool {
	ver1Sp := SplitVer(ver1)
	ver2Sp := SplitVer(ver2)
	ver1Sp, ver2Sp = PadVers(ver1Sp, ver2Sp)
	for i := 0; i < len(ver1Sp); i++ {
		l := cast.ToInt(ver1Sp[i])
		r := cast.ToInt(ver2Sp[i])
		if l != r {
			return false
		}
	}
	return true
}

// Check if ver1 is after or eq to ver2.
func VerAfterEq(ver1 string, ver2 string) bool {
	ver1Sp := SplitVer(ver1)
	ver2Sp := SplitVer(ver2)
	ver1Sp, ver2Sp = PadVers(ver1Sp, ver2Sp)
	for i := 0; i < len(ver1Sp); i++ {
		l := cast.ToInt(ver1Sp[i])
		r := cast.ToInt(ver2Sp[i])
		if l > r {
			return true
		} else if l < r {
			return false
		}
	}
	return true
}

// Check if ver1 is after ver2.
func VerAfter(ver1 string, ver2 string) bool {
	ver1Sp := SplitVer(ver1)
	ver2Sp := SplitVer(ver2)
	ver1Sp, ver2Sp = PadVers(ver1Sp, ver2Sp)
	for i := 0; i < len(ver1Sp); i++ {
		l := cast.ToInt(ver1Sp[i])
		r := cast.ToInt(ver2Sp[i])
		if l > r {
			return true
		} else if l < r {
			return false
		}
	}
	return false
}

func SplitVer(ver string) []string {
	ver = strings.ToLower(ver)
	ver = strings.TrimPrefix(ver, "v")
	return strings.Split(ver, VerSep)
}

// v1, v2.1 => v1.0, v2.1; v2, v1.2 => v2.0, v1.2
func PadVers(ver1 []string, ver2 []string) ([]string, []string) {
	if len(ver1) > len(ver2) {
		return ver1, PadVer(ver2, len(ver1))
	} else if len(ver1) > len(ver2) {
		return PadVer(ver1, len(ver2)), ver2
	}
	return ver1, ver2
}

func PadVer(ver1 []string, s int) []string {
	cp := make([]string, s)
	for i := 0; i < s; i++ {
		if i < len(ver1) {
			cp[i] = ver1[i]
		} else {
			cp[i] = "0"
		}
	}
	return cp
}
