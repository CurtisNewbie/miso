package jwt

import (
	"testing"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/golang-jwt/jwt/v5"
)

const (
	publicKey  = "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCUZIXyb43pYp6xr7nrnBWF23U/LyXu/Mgy6D34EW5N+fPV4hnYMCUVULjG8WZwN/kddIBDaab15y4L1WLBWiGarTP3O0LhvA2uJ4PcABi6AeqbTI5FeimByUMhypEHpELhpZIef9q5WpIj4C04tOE1FSaaWHlzdXQa9lR7JmjJDQIDAQAB"
	privateKey = "MIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAJRkhfJvjelinrGvueucFYXbdT8vJe78yDLoPfgRbk3589XiGdgwJRVQuMbxZnA3+R10gENppvXnLgvVYsFaIZqtM/c7QuG8Da4ng9wAGLoB6ptMjkV6KYHJQyHKkQekQuGlkh5/2rlakiPgLTi04TUVJppYeXN1dBr2VHsmaMkNAgMBAAECgYBxouU8eZb4MZCLS6GZvwZwYlXQE//9mtCIw3apIFgTGKVUlffqqTvMretCVhx3NTXtC4kplp/H0cheQYOFw8rU6G84GJnLmiq1Mq2kxzF2YA0agTe3YJpB0W5MpReoHZ0ryTaEdvyyT9KkWRD+oyO/QLQBM5fyDWnkD6gcJ5mVtQJBAM4wShYNtzCTG0XEqoyECWP4Cxf3wN8f3anSETJiIo5XKAG8+eXJkrAPzw7mruFwoKVDNFxz2nGzmqng6M+qttMCQQC4PdmDmxy4tlL4a9d+ESzOeFuP8HMGtbVYWiAmeM0S/xtLkI6/2+Ftt2+nqRRbKcROkqVqnourNy1DVdGkjFSfAkAYFW3h65I1O0mZOaKOLTIHmkZ5czf1F/zFREM79liA9c83fMJXw9a9d+tAm1NcA9LP2uy3y9R9KXRsWVf4QcF/AkEAkGoalyf8SWTQgFy3mt+HiYeZ7aeB4h6IOOrcDIvf4yYHlSGIYybM+p0wbfEAPbztXNFhy8Leo6QqXH9mRl6g7QJAJK544BDd0PyZFJpVE1t4YhcNS8H/3MP6iu2oUOn3LVvCiAATT9vzkJ298z+bQEjaLDv/KHU0IhSYnW14pr0E1w=="
	issuer     = "jwt_test"
)

func TestEncodeToken(t *testing.T) {
	common.SetProp(common.PROP_JWT_PRIVATE_KEY, privateKey)
	common.SetProp(common.PROP_JWT_PUBLIC_KEY, publicKey)
	common.SetProp(common.PROP_JWT_ISSUER, issuer)
	tkn, err := EncodeToken(jwt.MapClaims{
		"username": "zhuangyongj",
	}, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if tkn == "" {
		t.Fatal("token is empty")
	}
	t.Log(tkn)
}

func TestDecodeToken(t *testing.T) {
	common.SetProp(common.PROP_JWT_PRIVATE_KEY, privateKey)
	common.SetProp(common.PROP_JWT_PUBLIC_KEY, publicKey)
	common.SetProp(common.PROP_JWT_ISSUER, issuer)

	jwt := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTA2MjM1MTQsImlzcyI6Imp3dF90ZXN0IiwidXNlcm5hbWUiOiJ6aHVhbmd5b25naiJ9.Q_2CR6MyZ9jzmI0HswtfMHZ7-T9BZK62XXEDz_GwmmO_YRYRF5NmSdYng3-jHVjUsNIsfvu5VSUxOSrUgGzsURwYvxG-I8BtXAJf3dtV4n---iz9zvm8TzItiVeplEgW3N_QGx5RibKQM8lwt3-VZL2tU70AmsxNGVlKp63We_A"
	tkn, err := DecodeToken(jwt)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tkn.Claims)
}
