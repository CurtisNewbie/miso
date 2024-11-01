package util

import (
	cr "crypto/rand"
	"encoding/base64"
	"fmt"
	rand "math/rand/v2"
	"sync"
)

var (
	digits     = ShuffleStr("0123456789", 3)
	upperAlpha = ShuffleStr("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 3)
	lowerAlpha = ShuffleStr("abcdefghijklmnopqrstuvwxyz", 3)

	lowerAlphaDigitsRune = []rune(lowerAlpha + digits)

	rune16Pool = sync.Pool{
		New: func() any {
			v := make([]rune, 16)
			return &v
		},
	}
)

func init() {
}

const (
	DEFAULT_LEN = 35
)

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
//
// ERand() is preferred for higher entrophy
func RandStr(n int) string {
	return RandRune(n, []rune(digits+upperAlpha+lowerAlpha))
}

// Generate random numeric string with specified length
//
// the generated string will contains [0-9]
func RandNum(n int) string {
	return RandRune(n, []rune(digits))
}

// Generate random alphabetic string with specified length
//
// the generated string will contains [a-zA-Z]
func RandAlpha(n int) string {
	return RandRune(n, []rune(upperAlpha+lowerAlpha))
}

// Generate random alphabetic, uppercase string with specified length
//
// the generated string will contains [A-Z]
func RandUpperAlpha(n int) string {
	return RandRune(n, []rune(upperAlpha))
}

// Generate random alphabetic, lowercase string with specified length
//
// the generated string will contains [a-z]
func RandLowerAlpha(n int) string {
	return RandRune(n, []rune(lowerAlpha))
}

// Generate random alphabetic, uppercase string with specified length
//
// the generated string will contains [A-Z0-9]
func RandUpperAlphaNumeric(n int) string {
	return RandRune(n, []rune(upperAlpha+digits))
}

// Generate random alphabetic, lowercase string with specified length
//
// the generated string will contains [a-z0-9]
func RandLowerAlphaNumeric(n int) string {
	return RandRune(n, lowerAlphaDigitsRune)
}

// Same as RandLowerAlphaNumeric(16) but with less allocation.
func RandLowerAlphaNumeric16() string {
	return doRand16(lowerAlphaDigitsRune)
}

// generate randon str based on fixed length 16 and the given charset
func doRand16(set []rune) string {
	b := rune16Pool.Get().(*[]rune)
	for i := 0; i < 16; i++ {
		(*b)[i] = set[rand.IntN(16)]
	}
	s := string(*b)
	rune16Pool.Put(b)
	return s
}

// generate randon str based on given length and given charset
func RandRune(n int, set []rune) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = set[rand.IntN(len(set))]
	}
	return string(b)
}

// invoke one of the funcs randomly
func RandOp(ops ...func()) {
	n := len(ops)
	if n < 1 {
		return
	}
	ops[rand.IntN(n)]()
}

// pick random rune from the slice
func Pick(set []rune) rune {
	return set[rand.IntN(len(set))]
}

// generate a random sequence number with specified prefix
func GenNo(prefix string) string {
	return GenNoL(prefix, DEFAULT_LEN)
}

// generate a random sequence number with specified prefix
func GenNoL(prefix string, len int) string {
	return prefix + RandStr(len)
}

// Generate random string with high entrophy
func ERand(len int) string {
	if len < 1 {
		return ""
	}

	// each base64 character represent 6 bits of data
	c := len * 3 / 4 // wihtout padding
	b := make([]byte, c)
	_, err := cr.Read(b)

	// not a real io operation, we don't really need to handle the err, it will always be nil
	if err != nil {
		panic(fmt.Errorf("cr.Read(..) returns error, shouldn't happen, %v", err))
	}
	return base64.RawStdEncoding.EncodeToString(b)
}

func RandPick[T any](s []T) T {
	return s[rand.IntN(len(s))]
}

func WeightedRandPick[T interface{ GetWeight() float64 }](s []T) T {
	var totalWeight float64 = 0
	for _, v := range s {
		totalWeight += v.GetWeight()
	}
	i := 0
	r := rand.Float64() * totalWeight
	for ; i < len(s)-1; i++ {
		r -= s[i].GetWeight()
		if r <= 0.0 {
			break
		}
	}
	return s[i]
}

type WeightedItem[T any] struct {
	Value  T
	Weight float64
}

func (w WeightedItem[T]) GetWeight() float64 {
	return w.Weight
}
