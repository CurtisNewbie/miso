package crypto

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"

	"github.com/curtisnewbie/miso/util/strutil"
)

func SHA1Hex(b []byte) string {
	c := sha1.New()
	c.Write(b)
	return hex.EncodeToString(c.Sum(nil))
}

func SHA1HexStr(b string) string {
	return SHA1Hex(strutil.UnsafeStr2Byt(b))
}

func SHA256Hex(b []byte) string {
	c := sha256.New()
	c.Write(b)
	return hex.EncodeToString(c.Sum(nil))
}

func SHA256HexStr(b string) string {
	return SHA256Hex(strutil.UnsafeStr2Byt(b))
}
