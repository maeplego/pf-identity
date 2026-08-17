package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestVerifyS256(t *testing.T) {
	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJK"
	if len(verifier) < 43 {
		t.Fatal("fixture too short")
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if err := VerifyS256(verifier, challenge); err != nil {
		t.Fatal(err)
	}
	if err := VerifyS256(verifier+"x", challenge); err == nil {
		t.Fatal("expected mismatch")
	}
	if err := VerifyS256("short", challenge); err == nil {
		t.Fatal("expected length error")
	}
}
