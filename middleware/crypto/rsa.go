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

const (
	PubPemBegin = "-----BEGIN PUBLIC KEY-----"
	pubPemEnd   = "-----END PUBLIC KEY-----"

	PrivPemBegin = "-----BEGIN PRIVATE KEY-----"
	PrivPemEnd   = "-----END PRIVATE KEY-----"
)

var (
	ErrDecodePemFailed = errors.New("failed to decode public key pem")
	ErrInvalidKey      = errors.New("invalid key")
)

func LoadPrivKey(content string) (*rsa.PrivateKey, error) {
	if !strings.HasPrefix(content, PrivPemBegin) {
		content = PrivPemBegin + "\n" + content
	}
	if !strings.HasSuffix(content, PrivPemEnd) {
		content = content + "\n" + PrivPemEnd
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
	if !strings.HasPrefix(content, PubPemBegin) {
		content = PubPemBegin + "\n" + content
	}
	if !strings.HasSuffix(content, pubPemEnd) {
		content = content + "\n" + pubPemEnd
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
