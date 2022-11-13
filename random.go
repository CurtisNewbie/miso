package gocommon

import (
	"math/rand"
	"time"
)

var (
	letters = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
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

// generate random string with specified length
func RandStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
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
