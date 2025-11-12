package dbquery

import "github.com/curtisnewbie/miso/util/strutil"

// Escape String
//
// Acknowledgement: following code is copied from https://github.com/pingcap/tidb/blob/master/pkg/util/sqlescape/utils.go (Copyright 2021 PingCAP, Inc. Apache License)
func EscapeString(s string) string {
	obuf := make([]byte, 0, len(s))
	sbuf := strutil.UnsafeStr2Byt(s)
	return strutil.UnsafeByt2Str(escapeBytesBackslash(obuf, sbuf))
}

// Acknowledgement: following code is copied from https://github.com/pingcap/tidb/blob/master/pkg/util/sqlescape/utils.go (Copyright 2021 PingCAP, Inc. Apache License)
func escapeBytesBackslash(buf []byte, v []byte) []byte {
	pos := len(buf)
	buf = reserveBuffer(buf, len(v)*2)

	for _, c := range v {
		switch c {
		case '\x00':
			buf[pos] = '\\'
			buf[pos+1] = '0'
			pos += 2
		case '\n':
			buf[pos] = '\\'
			buf[pos+1] = 'n'
			pos += 2
		case '\r':
			buf[pos] = '\\'
			buf[pos+1] = 'r'
			pos += 2
		case '\x1a':
			buf[pos] = '\\'
			buf[pos+1] = 'Z'
			pos += 2
		case '\'':
			buf[pos] = '\\'
			buf[pos+1] = '\''
			pos += 2
		case '"':
			buf[pos] = '\\'
			buf[pos+1] = '"'
			pos += 2
		case '\\':
			buf[pos] = '\\'
			buf[pos+1] = '\\'
			pos += 2
		default:
			buf[pos] = c
			pos++
		}
	}

	return buf[:pos]
}

// Acknowledgement: following code is copied from https://github.com/pingcap/tidb/blob/master/pkg/util/sqlescape/utils.go (Copyright 2021 PingCAP, Inc. Apache License)
func reserveBuffer(buf []byte, appendSize int) []byte {
	newSize := len(buf) + appendSize
	if cap(buf) < newSize {
		newBuf := make([]byte, len(buf)*2+appendSize)
		copy(newBuf, buf)
		buf = newBuf
	}
	return buf[:newSize]
}
