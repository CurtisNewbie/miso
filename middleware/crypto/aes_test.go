package crypto

import (
	"crypto/aes"
	"strings"
	"testing"

	"github.com/curtisnewbie/miso/util"
)

func TestAesEcbPkc(t *testing.T) {
	s := []byte(util.RandAlpha(aes.BlockSize))

	plain := strings.Repeat("we are banana!", 10)
	t.Logf("plain: %v", plain)

	encrypted, err := AesEcbEncrypt(s, plain)
	if err != nil {
		panic(err)
	}
	decrypted, err := AesEcbDecrypt(s, encrypted)
	if err != nil {
		panic(err)
	}

	if plain != decrypted {
		t.Fatalf("es != ds, es: %v, ds: %v", plain, decrypted)
	}
	t.Log(decrypted)
}
