package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const (
	// ChallengeS256 is the only PKCE method this IdP accepts.
	// "plain" is omitted on purpose: a leaked authorize URL would reveal the verifier.
	ChallengeS256 = "S256"
)

// VerifyS256 checks code_verifier against a stored code_challenge.
func VerifyS256(verifier, challenge string) error {
	if verifier == "" || challenge == "" {
		return fmt.Errorf("pkce verifier and challenge are required")
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		return fmt.Errorf("pkce verifier length out of spec range")
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	if computed != challenge {
		return fmt.Errorf("pkce verification failed")
	}
	return nil
}
