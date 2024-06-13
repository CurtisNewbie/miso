package crypto

import (
	"crypto/aes"
	"encoding/hex"

	"github.com/curtisnewbie/miso/util"
)

// AES/ECB/PKCSPadding encrypt and encoded as hex string.
func AesEcbEncrypt(secret []byte, plain string) (string, error) {
	ci, err := aes.NewCipher(secret)
	if err != nil {
		return "", err
	}

	padded := PKCSPadding([]byte(plain), ci.BlockSize())
	encrypted := make([]byte, len(padded))

	enc := NewECBEncrypter(ci)
	enc.CryptBlocks(encrypted, padded)
	encoded := hex.EncodeToString(encrypted)
	return encoded, nil
}

// AES/ECB/PKCSPadding decode hex string and decrypt.
func AesEcbDecrypt(secret []byte, s string) (string, error) {
	ci, err := aes.NewCipher(secret)
	if err != nil {
		return "", err
	}

	encrypted, err := hex.DecodeString(s)
	if err != nil {
		return "", err
	}

	decrypted := make([]byte, len(encrypted))
	dec := NewECBDecrypter(ci)
	dec.CryptBlocks(decrypted, encrypted)
	decrypted = PKCSTrimming(decrypted)
	return util.UnsafeByt2Str(decrypted), nil
}
