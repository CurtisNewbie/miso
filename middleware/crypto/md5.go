package crypto

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/curtisnewbie/miso/util/strutil"
)

func MD5Hex(b []byte) string {
	md5 := md5.New()
	md5.Write(b)
	return hex.EncodeToString(md5.Sum(nil))
}

func MD5HexStr(b string) string {
	return MD5Hex(strutil.UnsafeStr2Byt(b))
}
