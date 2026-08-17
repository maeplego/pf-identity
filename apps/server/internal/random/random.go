// Package random produces unguessable tokens for codes, sessions, and secrets.
// math/rand is never used here: predictability would let an attacker mint sessions.
package random

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Bytes returns n cryptographically random bytes. n must be at least 16.
func Bytes(n int) ([]byte, error) {
	if n < 16 {
		return nil, fmt.Errorf("random byte length %d is below 16", n)
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("crypto/rand: %w", err)
	}
	return b, nil
}

// Token returns a URL-safe string with at least 256 bits of entropy (32 bytes).
func Token() (string, error) {
	b, err := Bytes(32)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
