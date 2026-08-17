// Package oidc signs ID Tokens and serves JWKS. Signing uses jwx; we do not
// implement RSA-PKCS1 ourselves.
package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Signer holds the current RSA key and its kid.
type Signer struct {
	key jwk.Key
	pub jwk.Set
	kid string
}

// LoadRSA reads a PKCS#8 PEM file.
func LoadRSA(path string) (*Signer, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("no PEM block in %s", path)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse pkcs8: %w", err)
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}
	return fromRSA(rsaKey)
}

// GenerateRSA makes an ephemeral 2048-bit key for local development.
// Tokens become invalid after restart, which is acceptable for IDENTITY_DEV_GENERATE_KEYS.
func GenerateRSA() (*Signer, error) {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return fromRSA(k)
}

func fromRSA(priv *rsa.PrivateKey) (*Signer, error) {
	key, err := jwk.FromRaw(priv)
	if err != nil {
		return nil, err
	}
	kid := "dev-1"
	if err := key.Set(jwk.KeyIDKey, kid); err != nil {
		return nil, err
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		return nil, err
	}
	pub, err := key.PublicKey()
	if err != nil {
		return nil, err
	}
	set := jwk.NewSet()
	if err := set.AddKey(pub); err != nil {
		return nil, err
	}
	return &Signer{key: key, pub: set, kid: kid}, nil
}

// JWKS returns the public key set for /.well-known/jwks.json.
func (s *Signer) JWKS() jwk.Set { return s.pub }

// IDTokenInput is the minimum OIDC ID Token claims this IdP issues.
type IDTokenInput struct {
	Issuer   string
	Subject  string
	Audience string
	Nonce    string
	Email    string
	Name     string
	Verified bool
	SID      string
	Now      time.Time
	TTL      time.Duration
}

// SignIDToken builds a compact RS256 JWT.
func (s *Signer) SignIDToken(in IDTokenInput) (string, error) {
	b := jwt.NewBuilder().
		Issuer(in.Issuer).
		Subject(in.Subject).
		Audience([]string{in.Audience}).
		IssuedAt(in.Now).
		Expiration(in.Now.Add(in.TTL)).
		Claim("nonce", in.Nonce).
		Claim("email", in.Email).
		Claim("email_verified", in.Verified).
		Claim("name", in.Name)
	if in.SID != "" {
		b = b.Claim("sid", in.SID)
	}
	tok, err := b.Build()
	if err != nil {
		return "", err
	}
	raw, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, s.key))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// LogoutTokenEvent is the Back-Channel Logout event type in the events claim.
const LogoutTokenEvent = "http://schemas.openid.net/event/backchannel-logout"

// LogoutTokenInput is the claims for a back-channel logout_token. nonce must never be set.
type LogoutTokenInput struct {
	Issuer   string
	Subject  string
	Audience string
	SID      string
	JTI      string
	Now      time.Time
	TTL      time.Duration
}

// SignLogoutToken builds a compact RS256 JWT for Back-Channel Logout.
func (s *Signer) SignLogoutToken(in LogoutTokenInput) (string, error) {
	if in.Subject == "" && in.SID == "" {
		return "", fmt.Errorf("logout_token needs sub or sid")
	}
	b := jwt.NewBuilder().
		Issuer(in.Issuer).
		Audience([]string{in.Audience}).
		IssuedAt(in.Now).
		Expiration(in.Now.Add(in.TTL)).
		JwtID(in.JTI).
		Claim("events", map[string]any{LogoutTokenEvent: map[string]any{}})
	if in.Subject != "" {
		b = b.Subject(in.Subject)
	}
	if in.SID != "" {
		b = b.Claim("sid", in.SID)
	}
	tok, err := b.Build()
	if err != nil {
		return "", err
	}
	raw, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, s.key))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// IDTokenHint is the subset of an ID Token used by RP-Initiated Logout.
// The spec allows expired hints, so expiry is not checked.
type IDTokenHint struct {
	Issuer   string
	Subject  string
	Audience []string
}

// ParseIDTokenHint verifies the signature and returns claims. Expiry is ignored on purpose.
func (s *Signer) ParseIDTokenHint(compact string) (IDTokenHint, error) {
	if compact == "" {
		return IDTokenHint{}, fmt.Errorf("empty id_token_hint")
	}
	tok, err := jwt.Parse([]byte(compact), jwt.WithKeySet(s.JWKS()), jwt.WithValidate(false))
	if err != nil {
		return IDTokenHint{}, err
	}
	return IDTokenHint{
		Issuer:   tok.Issuer(),
		Subject:  tok.Subject(),
		Audience: append([]string{}, tok.Audience()...),
	}, nil
}
