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
	// PKI X.509, public key start
	PubPemBegin = "-----BEGIN PUBLIC KEY-----"

	// PKI X.509, public key end
	pubPemEnd = "-----END PUBLIC KEY-----"

	// PKCS8, private key start
	PrivPemBegin = "-----BEGIN PRIVATE KEY-----"

	// PKCS8, private key end
	PrivPemEnd = "-----END PRIVATE KEY-----"
)

var (
	ErrDecodePemFailed = errors.New("failed to decode public key pem")
	ErrInvalidKey      = errors.New("invalid key")
)

// Load RSA PrivateKey from content.
//
// Content should be PKCS8 compatible format with or without '-----BEGIN PRIVATE KEY-----' prefix and suffix.
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

// Load RSA PublicKey from content.
//
// Content should be X.509 compatible format with or without '-----BEGIN PUBLIC KEY-----' prefix and suffix.
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
