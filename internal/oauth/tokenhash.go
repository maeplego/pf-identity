package oauth

import (
	"crypto/sha256"
	"encoding/base64"
)

// HashToken SHA-256s a bearer secret before storage.
// The raw value lives only in the cookie, redirect, or token response.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
