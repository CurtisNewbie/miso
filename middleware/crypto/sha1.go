package crypto

import (
	"crypto/sha1"
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
