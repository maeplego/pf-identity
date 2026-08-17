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
		SID:      "sid-1",
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
	sid, _ := tok.Get("sid")
	if sid != "sid-1" {
		t.Fatalf("sid %v", sid)
	}
	if s.JWKS().Len() != 1 {
		t.Fatal("expected one public key")
	}
	_ = jwa.RS256
}

func TestSignLogoutToken(t *testing.T) {
	s, err := GenerateRSA()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	compact, err := s.SignLogoutToken(LogoutTokenInput{
		Issuer:   "http://localhost:8080",
		Subject:  "user-sub",
		Audience: "client-1",
		SID:      "sid-1",
		JTI:      "jti-1",
		Now:      now,
		TTL:      2 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := jwt.Parse([]byte(compact), jwt.WithKeySet(s.JWKS()), jwt.WithValidate(true), jwt.WithAcceptableSkew(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if tok.Subject() != "user-sub" || tok.JwtID() != "jti-1" {
		t.Fatalf("%v %s", tok.Subject(), tok.JwtID())
	}
	events, ok := tok.Get("events")
	if !ok {
		t.Fatal("missing events")
	}
	m, ok := events.(map[string]any)
	if !ok {
		t.Fatalf("events %T", events)
	}
	if _, ok := m[LogoutTokenEvent]; !ok {
		t.Fatalf("events %v", events)
	}
	if _, ok := tok.Get("nonce"); ok {
		t.Fatal("logout_token must not contain nonce")
	}
}

func TestParseIDTokenHintAcceptsExpired(t *testing.T) {
	s, err := GenerateRSA()
	if err != nil {
		t.Fatal(err)
	}
	compact, err := s.SignIDToken(IDTokenInput{
		Issuer:   "http://localhost:8080",
		Subject:  "user-sub",
		Audience: "client-1",
		Nonce:    "n",
		Now:      time.Now().UTC().Add(-2 * time.Hour),
		TTL:      time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jwt.Parse([]byte(compact), jwt.WithKeySet(s.JWKS()), jwt.WithValidate(true)); err == nil {
		t.Fatal("expected expiry to fail full validation")
	}
	hint, err := s.ParseIDTokenHint(compact)
	if err != nil {
		t.Fatal(err)
	}
	if hint.Subject != "user-sub" || hint.Issuer != "http://localhost:8080" || hint.Audience[0] != "client-1" {
		t.Fatalf("%+v", hint)
	}
}

func TestParseIDTokenHintRejectsBadSignature(t *testing.T) {
	s, err := GenerateRSA()
	if err != nil {
		t.Fatal(err)
	}
	other, err := GenerateRSA()
	if err != nil {
		t.Fatal(err)
	}
	compact, err := other.SignIDToken(IDTokenInput{
		Issuer:   "http://localhost:8080",
		Subject:  "user-sub",
		Audience: "client-1",
		Now:      time.Now().UTC(),
		TTL:      time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ParseIDTokenHint(compact); err == nil {
		t.Fatal("expected signature failure")
	}
}
