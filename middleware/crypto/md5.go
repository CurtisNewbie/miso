package crypto

import (
	"crypto/md5"
	"encoding/hex"
)

func MD5Hex(b []byte) string {
	md5 := md5.New()
	md5.Write(b)
	return hex.EncodeToString(md5.Sum(nil))
}
