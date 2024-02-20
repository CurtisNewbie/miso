package miso

import (
	"reflect"
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
// Tricks from https://github.com/valyala/fasthttp.
func UnsafeByt2Str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
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
// Tricks from https://github.com/valyala/fasthttp.
func UnsafeStr2Byt(s string) (b []byte) {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Cap = sh.Len
	bh.Len = sh.Len
	return b
}
