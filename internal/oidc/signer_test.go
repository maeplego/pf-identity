package oidc

import (
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestSignIDTokenRoundTrip(t *testing.T) {
	s, err := GenerateRSA()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	compact, err := s.SignIDToken(IDTokenInput{
		Issuer:   "http://localhost:8080",
		Subject:  "user-sub",
		Audience: "client-1",
		Nonce:    "n-1",
		Email:    "a@b.co",
		Name:     "A",
		Verified: false,
		Now:      now,
		TTL:      time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := jwt.Parse([]byte(compact), jwt.WithKeySet(s.JWKS()), jwt.WithValidate(true), jwt.WithAcceptableSkew(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if tok.Subject() != "user-sub" {
		t.Fatalf("sub %s", tok.Subject())
	}
	if tok.Audience()[0] != "client-1" {
		t.Fatalf("aud %v", tok.Audience())
	}
	nonce, _ := tok.Get("nonce")
	if nonce != "n-1" {
		t.Fatalf("nonce %v", nonce)
	}
	if s.JWKS().Len() != 1 {
		t.Fatal("expected one public key")
	}
	_ = jwa.RS256
}
