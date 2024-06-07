package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/curtisnewbie/miso/util"
)

var (
	PUB_PEM_BEGIN = "-----BEGIN PUBLIC KEY-----"
	PUB_PEM_END   = "-----END PUBLIC KEY-----"

	PRIV_PEM_BEGIN = "-----BEGIN PRIVATE KEY-----"
	PRIV_PEM_END   = "-----END PRIVATE KEY-----"

	ErrDecodePemFailed = errors.New("failed to decode public key pem")
	ErrInvalidKey      = errors.New("invalid key")
)

func LoadPrivKey(content string) (*rsa.PrivateKey, error) {
	if !strings.HasPrefix(content, PRIV_PEM_BEGIN) {
		content = PRIV_PEM_BEGIN + "\n" + content
	}
	if !strings.HasSuffix(content, PRIV_PEM_END) {
		content = content + "\n" + PRIV_PEM_END
	}

	decodedPem, _ := pem.Decode(util.UnsafeStr2Byt(content))
	if decodedPem == nil {
		return nil, ErrDecodePemFailed
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(decodedPem.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key, %w", err)
	}

	var privKey *rsa.PrivateKey
	var ok bool
	if privKey, ok = parsedKey.(*rsa.PrivateKey); !ok {
		return nil, ErrInvalidKey
	}
	return privKey, nil
}

func LoadPubKey(content string) (*rsa.PublicKey, error) {
	if !strings.HasPrefix(content, PUB_PEM_BEGIN) {
		content = PUB_PEM_BEGIN + "\n" + content
	}
	if !strings.HasSuffix(content, PUB_PEM_END) {
		content = content + "\n" + PUB_PEM_END
	}

	decodedPem, _ := pem.Decode(util.UnsafeStr2Byt(content))
	if decodedPem == nil {
		return nil, ErrDecodePemFailed
	}

	parsedPubKey, err := x509.ParsePKIXPublicKey(decodedPem.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key, %w", err)
	}

	var pubKey *rsa.PublicKey
	var ok bool
	if pubKey, ok = parsedPubKey.(*rsa.PublicKey); !ok {
		return nil, ErrInvalidKey
	}
	return pubKey, nil
}
