package core

import (
	"strconv"
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
