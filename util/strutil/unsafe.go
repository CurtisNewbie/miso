package strutil

import (
	"unsafe"
)

// Convert []byte to string without alloc.
//
// Both the []byte and the string share the same memory.
//
// Any modification on the original []byte is reflected on the returned string.
//
//	byt = []byte("abc")
//	s = UnsafeByt2Str(byt) // "abc" using the same memory
//	byt[0] = 'd' // modified in place at 0, also reflected on s ("dbc")
//
// Tricks from https://github.com/valyala/fasthttp and https://go101.org/article/unsafe.html
//
// See: https://github.com/golang/go/issues/53003
func UnsafeByt2Str(b []byte) string {
	if len(b) < 1 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// Convert string to []byte without alloc.
//
// Both the []byte and the string share the same memory.
//
// The resulting []byte is not modifiable, program will panic if modified.
//
//	s := "abc"
//	byt := UnsafeStr2Byt(s) // "abc" but in []byte
//	byt[0] = 'd' // will panic
//
// Tricks from https://github.com/valyala/fasthttp and https://go101.org/article/unsafe.html
//
// See: https://github.com/golang/go/issues/53003
func UnsafeStr2Byt(s string) (b []byte) {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// Get string's length in bytes
func StrByteLen(s string) int {
	return len(UnsafeStr2Byt(s))
}
