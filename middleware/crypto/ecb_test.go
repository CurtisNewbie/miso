package crypto

import (
	"crypto/aes"
	"testing"

	"github.com/curtisnewbie/miso/util"
)

func TestEcb(t *testing.T) {
	s := []byte(util.RandAlpha(aes.BlockSize))
	t.Logf("s: %v, len(s): %v", s, len(s))
	ci, err := aes.NewCipher(s)
	if err != nil {
		panic(err)
	}
	plain := "we are banana!"
	padded := PKCSPadding([]byte(plain), ci.BlockSize())
	t.Logf("padded: %v", padded)

	encrypted := make([]byte, len(padded))
	decrypted := make([]byte, len(padded))

	enc := NewECBEncrypter(ci)
	enc.CryptBlocks(encrypted, padded)
	t.Logf("encrypted: %v", encrypted)

	dec := NewECBDecrypter(ci)
	dec.CryptBlocks(decrypted, encrypted)
	t.Logf("decrypted: %v", decrypted)

	decrypted = PKCSTrimming(decrypted)
	t.Logf("trimmed: %v", decrypted)

	if plain != string(decrypted) {
		t.Fatalf("es != ds, es: %v, ds: %v", plain, string(decrypted))
	}
	t.Log(string(decrypted))
}
