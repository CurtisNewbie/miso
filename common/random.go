package common

import (
	"math/rand"
	"time"
)

var (
	letters    = []rune("0123456789" + "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	digits     = []rune("0123456789")
	alphabets  = []rune("abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	upperAlpha = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	lowerAlpha = []rune("abcdefghijklmnopqrstuvwxyz")
)

const (
	DEFAULT_LEN       = 35
	INIT_SHUFFLE_TIME = 3
)

func init() {
	rand.Seed(time.Now().UnixNano())
	swap := func(i, j int) { letters[i], letters[j] = letters[j], letters[i] }
	for i := 0; i < INIT_SHUFFLE_TIME; i++ {
		rand.Shuffle(len(letters), swap)
	}
}

// Generate random string with specified length
//
// the generated string will contains [a-zA-Z0-9]
func RandStr(n int) string {
	return doRand(n, letters)
}

// Generate random numeric string with specified length
//
// the generated string will contains [0-9]
func RandNum(n int) string {
	return doRand(n, digits)
}

// Generate random alphabetic string with specified length
//
// the generated string will contains [a-zA-Z]
func RandAlpha(n int) string {
	return doRand(n, alphabets)
}

// Generate random alphabetic, uppercase string with specified length
//
// the generated string will contains [A-Z]
func RandUpperAlpha(n int) string {
	return doRand(n, upperAlpha)
}

// Generate random alphabetic, lowercase string with specified length
//
// the generated string will contains [a-z]
func RandLowerAlpha(n int) string {
	return doRand(n, lowerAlpha)
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
