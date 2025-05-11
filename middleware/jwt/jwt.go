package jwt

import (
	"crypto/rsa"
	"errors"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/middleware/crypto"
	"github.com/curtisnewbie/miso/miso"
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
)

//lint:ignore U1000 for future use
var module = miso.InitAppModuleFunc(func() *jwtModule {
	return &jwtModule{
		privKeyRwmu: &sync.RWMutex{},
		pubKeyRwmu:  &sync.RWMutex{},
	}
})

type jwtModule struct {
	privKeyRwmu *sync.RWMutex
	privKey     *rsa.PrivateKey

	pubKeyRwmu *sync.RWMutex
	pubKey     *rsa.PublicKey
}

func (m *jwtModule) jwtEncode(claims jwt.MapClaims, exp time.Duration) (string, error) {
	pk, err := m.loadPrivateKey()
	if err != nil {
		return "", err
	}
	issuer := miso.GetPropStr(PropJwtIssue)
	return JwtKeyEncode(pk, claims, exp, issuer)
}

func (m *jwtModule) jwtDecode(token string) (ParsedJwt, error) {
	pubKey, err := m.loadPublicKey()
	if err != nil {
		return ParsedJwt{}, err
	}
	iss := miso.GetPropStr(PropJwtIssue)
	return JwtKeyDecode(pubKey, token, iss)
}

func (m *jwtModule) loadPublicKey() (*rsa.PublicKey, error) {
	m.pubKeyRwmu.RLock()
	if m.pubKey != nil {
		defer m.pubKeyRwmu.RUnlock()
		return m.pubKey, nil
	}
	m.pubKeyRwmu.RUnlock()

	m.pubKeyRwmu.Lock()
	defer m.pubKeyRwmu.Unlock()

	if !miso.HasProp(PropJwtPublicKey) {
		return nil, ErrMissingPublicKey
	}

	k := miso.GetPropStr(PropJwtPublicKey)
	pk, err := crypto.LoadPubKey(k)
	if err != nil {
		miso.Errorf("Failed to load public key, %v", err)
		return nil, err
	}

	m.pubKey = pk
	return m.pubKey, nil
}

func (m *jwtModule) loadPrivateKey() (*rsa.PrivateKey, error) {
	m.privKeyRwmu.RLock()
	if m.privKey != nil {
		defer m.privKeyRwmu.RUnlock()
		return m.privKey, nil
	}
	m.privKeyRwmu.RUnlock()

	m.privKeyRwmu.Lock()
	defer m.privKeyRwmu.Unlock()

	if m.privKey != nil {
		return m.privKey, nil
	}

	if !miso.HasProp(PropJwtPrivateKey) {
		return nil, ErrMissingPublicKey
	}

	k := miso.GetPropStr(PropJwtPrivateKey)
	pk, err := crypto.LoadPrivKey(k)
	if err != nil {
		miso.Errorf("Failed to load private key, %v", err)
		return nil, err
	}

	m.privKey = pk
	return m.privKey, nil
}

// JWT Encode using default configuration in loaded properties.
func JwtEncode(claims jwt.MapClaims, exp time.Duration) (string, error) {
	return module().jwtEncode(claims, exp)
}

// JWT Decode using default configuration in loaded properties.
func JwtDecode(token string) (ParsedJwt, error) {
	return module().jwtDecode(token)
}

// JWT Encode using provided key, claims, exp and iss.
func JwtKeyEncode(pk *rsa.PrivateKey, claims jwt.MapClaims, exp time.Duration, issuer string) (string, error) {
	claims["iss"] = issuer
	claims["exp"] = jwt.NewNumericDate(time.Now().Add(exp))

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(pk)
}

// JWT Decode using provided key and iss.
func JwtKeyDecode(pk *rsa.PublicKey, token string, issuer string) (ParsedJwt, error) {
	validateIssuerFunc := func(p *jwt.Parser) {}
	if issuer != "" {
		validateIssuerFunc = jwt.WithIssuer(issuer)
	}
	pubKeyFunc := func(token *jwt.Token) (interface{}, error) {
		return pk, nil
	}

	parsed, err := jwt.Parse(token, pubKeyFunc, validateIssuerFunc)
	if err != nil {
		return ParsedJwt{Valid: false}, err
	}
	if !parsed.Valid {
		return ParsedJwt{Valid: false}, nil
	}

	if claims, ok := parsed.Claims.(jwt.MapClaims); ok {
		return ParsedJwt{Valid: true, Claims: claims}, nil
	} else {
		return ParsedJwt{Valid: false}, ErrExtractClaimFailed
	}
}
