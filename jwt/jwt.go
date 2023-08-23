package jwt

import (
	"crypto/rsa"
	"errors"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/core"
	jwt "github.com/golang-jwt/jwt/v5"
)

type ParsedJwt struct {
	Valid  bool
	Claims jwt.MapClaims
}

var (
	ErrMissingPrivateKey  = errors.New("missing private key")
	ErrMissingPublicKey   = errors.New("missing public key")
	ErrExtractClaimFailed = errors.New("unable to extract claims from token")

	privKeyRwmu sync.RWMutex
	privKey     *rsa.PrivateKey

	pubKeyRwmu sync.RWMutex
	pubKey     *rsa.PublicKey
)

// --------------------------------------------------

func EncodeToken(claims jwt.MapClaims, exp time.Duration) (string, error) {
	pk, err := loadPrivateKey()
	if err != nil {
		return "", err
	}

	claims["iss"] = core.GetPropStr(core.PROP_JWT_ISSUER)
	claims["exp"] = jwt.NewNumericDate(time.Now().Add(exp))

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(pk)
}

func DecodeToken(token string) (ParsedJwt, error) {
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return loadPublicKey()
	}, ValidateIssuer())

	if err != nil {
		return ParsedJwt{}, err
	}

	if !parsed.Valid {
		return ParsedJwt{Valid: false}, nil
	}

	if claims, ok := parsed.Claims.(jwt.MapClaims); ok {
		return ParsedJwt{Valid: true, Claims: claims}, nil
	} else {
		return ParsedJwt{}, ErrExtractClaimFailed
	}
}

func loadPublicKey() (any, error) {
	pubKeyRwmu.RLock()
	if pubKey != nil {
		defer pubKeyRwmu.RUnlock()
		return pubKey, nil
	}
	pubKeyRwmu.RUnlock()

	pubKeyRwmu.Lock()
	defer pubKeyRwmu.Unlock()

	if !core.HasProp(core.PROP_JWT_PUBLIC_KEY) {
		return nil, ErrMissingPublicKey
	}

	k := core.GetPropStr(core.PROP_JWT_PUBLIC_KEY)
	pk, err := core.LoadPubKey(k)
	if err != nil {
		core.EmptyRail().Errorf("Failed to load public key, %v", err)
		return nil, err
	}

	pubKey = pk
	return pubKey, nil
}

func loadPrivateKey() (any, error) {
	privKeyRwmu.RLock()
	if privKey != nil {
		defer privKeyRwmu.RUnlock()
		return privKey, nil
	}
	privKeyRwmu.RUnlock()

	privKeyRwmu.Lock()
	defer privKeyRwmu.Unlock()

	if privKey != nil {
		return privKey, nil
	}

	if !core.HasProp(core.PROP_JWT_PRIVATE_KEY) {
		return nil, ErrMissingPublicKey
	}

	k := core.GetPropStr(core.PROP_JWT_PRIVATE_KEY)
	pk, err := core.LoadPrivKey(k)
	if err != nil {
		core.EmptyRail().Errorf("Failed to load private key, %v", err)
		return nil, err
	}

	privKey = pk
	return privKey, nil
}

func ValidateIssuer() jwt.ParserOption {
	iss := core.GetPropStr(core.PROP_JWT_ISSUER)
	if iss == "" {
		return func(p *jwt.Parser) {}
	}
	return jwt.WithIssuer(iss)
}
