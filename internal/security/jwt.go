package security

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UID string `json:"uid"`
	jwt.RegisteredClaims
}

func Sign(secret, uid string, ttl time.Duration) (string, error) {
	claims := Claims{
		UID: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func Parse(secret, token string, out *Claims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(
		token,
		out,
		func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		},
	)
}
