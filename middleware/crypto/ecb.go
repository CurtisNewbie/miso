package crypto

import "crypto/cipher"

// ECB Mode
type ecb struct {
	b         cipher.Block
	blockSize int
}

func newEcb(b cipher.Block) *ecb {
	return &ecb{b: b, blockSize: b.BlockSize()}
}

// ECB block mode for encryption.
func NewECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ECBEncrypter)(newEcb(b))
}

// ECB block mode for decryption.
func NewECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ECBDecrypter)(newEcb(b))
}

type ECBEncrypter ecb

func (ec *ECBEncrypter) BlockSize() int {
	return ec.blockSize
}
func (ec *ECBEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%ec.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}

	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}

	for len(src) > 0 {
		ec.b.Encrypt(dst[:ec.blockSize], src[:ec.blockSize])
		src = src[ec.blockSize:]
		dst = dst[ec.blockSize:]
	}
}

type ECBDecrypter ecb

func (ec *ECBDecrypter) BlockSize() int {
	return ec.blockSize
}
func (ec *ECBDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%ec.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	if len(src) == 0 {
		return
	}

	for len(src) > 0 {
		ec.b.Decrypt(dst[:ec.blockSize], src[:ec.blockSize])
		src = src[ec.blockSize:]
		dst = dst[ec.blockSize:]
	}
}
