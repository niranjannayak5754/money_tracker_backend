package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// GenerateOpaqueToken creates a random refresh-token value. Only its hash is
// meant to be persisted — the plain value is returned once, to the caller,
// and never stored, so a database read alone can't be replayed as a token.
func GenerateOpaqueToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	return plain, HashToken(plain), nil
}

// HashToken deterministically hashes a plain refresh-token value for storage
// and lookup.
func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
