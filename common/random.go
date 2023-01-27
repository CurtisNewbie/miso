package common

import (
	"math/rand"
	"time"
)

var (
	digits     = ShuffleStr("0123456789", 3)
	upperAlpha = ShuffleStr("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 3)
	lowerAlpha = ShuffleStr("abcdefghijklmnopqrstuvwxyz", 3)
)

func init() {
}

const (
	DEFAULT_LEN = 35
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func ShuffleStr(letters string, times int) string {
	return string(ShuffleRunes([]rune(letters), times))
}

func ShuffleRunes(letters []rune, times int) []rune {
	if letters == nil || len(letters) < 1 {
		return letters
	}

	for i := 0; i < times; i++ {
		rand.Shuffle(len(letters), func(i, j int) { letters[i], letters[j] = letters[j], letters[i] })
	}
	return letters
}

// Generate random string with specified length
//
// the generated string will contains [a-zA-Z0-9]
func RandStr(n int) string {
	return doRand(n, []rune(digits+upperAlpha+lowerAlpha))
}

// Generate random numeric string with specified length
//
// the generated string will contains [0-9]
func RandNum(n int) string {
	return doRand(n, []rune(digits))
}

// Generate random alphabetic string with specified length
//
// the generated string will contains [a-zA-Z]
func RandAlpha(n int) string {
	return doRand(n, []rune(upperAlpha+lowerAlpha))
}

// Generate random alphabetic, uppercase string with specified length
//
// the generated string will contains [A-Z]
func RandUpperAlpha(n int) string {
	return doRand(n, []rune(upperAlpha))
}

// Generate random alphabetic, lowercase string with specified length
//
// the generated string will contains [a-z]
func RandLowerAlpha(n int) string {
	return doRand(n, []rune(lowerAlpha))
}

// Generate random alphabetic, uppercase string with specified length
//
// the generated string will contains [A-Z0-9]
func RandUpperAlphaNumeric(n int) string {
	return doRand(n, []rune(upperAlpha+digits))
}

// Generate random alphabetic, lowercase string with specified length
//
// the generated string will contains [a-z0-9]
func RandLowerAlphaNumeric(n int) string {
	return doRand(n, []rune(lowerAlpha+digits))
}

// generate randon str based on given length and charset
func doRand(n int, set []rune) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = set[rand.Intn(len(set))]
	}
	return string(b)
}

// generate a random sequence number with specified prefix
func GenNo(prefix string) string {
	return GenNoL(prefix, DEFAULT_LEN)
}

// generate a random sequence number with specified prefix
func GenNoL(prefix string, len int) string {
	return prefix + RandStr(len)
}
