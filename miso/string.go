package miso

import (
	"strconv"
	"strings"
)

func RepeatStr(tkn string, times int) string {
	var s string
	for i := 0; i < times; i++ {
		s += tkn
	}
	return s
}

func PadNum(n int, digit int) string {
	var cnt int
	var v int = n
	for v > 0 {
		cnt += 1
		v /= 10
	}
	pad := digit - cnt
	num := strconv.Itoa(n)
	if pad > 0 {
		if pad == digit {
			return RepeatStr("0", pad)
		}
		return RepeatStr("0", pad) + num
	}
	return num
}

type RString []rune

// substring
func (r RString) Substr(start int, end int) string {
	return string(r[start:end])
}

// substring starting from start
func (r RString) SubstrStart(start int) string {
	return string(r[start:])
}

// substring before end
func (r RString) SubstrEnd(end int) string {
	return string(r[:end])
}

// count number of runes
func (r RString) RuneLen() int {
	return len(r)
}

// get string at
func (r RString) StrAt(i int) string {
	return string(r[i])
}

// check if r is blank
//
// dont't use it just because you need IsBlank
func (r RString) IsBlank() bool {
	return len(r) < 1 || IsBlankStr(string(r))
}

// if r is blank return defStr else return r
func (r RString) IfBlankThen(defStr string) string {
	if r.IsBlank() {
		return defStr
	}
	return string(r)
}

// Check if the string is blank
func IsBlankStr(s string) bool {
	return strings.TrimSpace(s) == ""
}

// Substring such that len(s) <= max
func MaxLenStr(s string, max int) string {
	ru := []rune(s)
	return string(ru[:MinInt(len(ru), max)])
}
