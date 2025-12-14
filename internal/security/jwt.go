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
	now := time.Now().UTC()

	claims := Claims{
		UID: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
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
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		},
	)
}
