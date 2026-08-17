// Package storetest is the persistence contract both memory and Postgres must satisfy.
package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/id"
)

// Repos exercises single-use codes, unique emails, and refresh-family revoke.
func Repos(t *testing.T, s domain.Repos) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 8, 17, 6, 0, 0, 0, time.UTC)

	t.Run("user email unique and lowercased", func(t *testing.T) {
		u := domain.User{ID: id.New(), Email: "A@Example.COM", Name: "A", PasswordHash: "x", CreatedAt: now}
		if err := s.Create(ctx, u); err != nil {
			t.Fatal(err)
		}
		got, err := s.GetByEmail(ctx, "a@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if got.Email != "a@example.com" {
			t.Fatalf("email stored as %q", got.Email)
		}
		if _, err := s.GetByID(ctx, u.ID); err != nil {
			t.Fatal(err)
		}
		dup := u
		dup.ID = id.New()
		if err := s.Create(ctx, dup); err != domain.ErrConflict {
			t.Fatalf("conflict: %v", err)
		}
	})

	t.Run("auth code single use", func(t *testing.T) {
		c := domain.AuthCode{
			Hash:          id.New(),
			ClientID:      "c",
			UserID:        "u",
			RedirectURI:   "http://127.0.0.1/cb",
			Scopes:        []string{"openid", "email"},
			Nonce:         "n",
			CodeChallenge: "ch",
			ExpiresAt:     now.Add(time.Minute),
		}
		if err := s.PutCode(ctx, c); err != nil {
			t.Fatal(err)
		}
		got, err := s.TakeCode(ctx, c.Hash)
		if err != nil {
			t.Fatal(err)
		}
		if got.Nonce != "n" || len(got.Scopes) != 2 {
			t.Fatalf("taken %+v", got)
		}
		if _, err := s.TakeCode(ctx, c.Hash); err != domain.ErrUsed {
			t.Fatalf("second take: %v", err)
		}
		if _, err := s.TakeCode(ctx, "missing-"+id.New()); err != domain.ErrNotFound {
			t.Fatalf("missing: %v", err)
		}
	})

	t.Run("refresh family revoke", func(t *testing.T) {
		fam := id.New()
		a := domain.RefreshToken{Hash: id.New(), FamilyID: fam, ClientID: "c", UserID: "u", Scopes: []string{"openid"}, ExpiresAt: now.Add(time.Hour)}
		b := domain.RefreshToken{Hash: id.New(), FamilyID: fam, ClientID: "c", UserID: "u", Scopes: []string{"openid"}, ExpiresAt: now.Add(time.Hour)}
		other := domain.RefreshToken{Hash: id.New(), FamilyID: id.New(), ClientID: "c", UserID: "u", Scopes: []string{"openid"}, ExpiresAt: now.Add(time.Hour)}
		if err := s.PutRefresh(ctx, a); err != nil {
			t.Fatal(err)
		}
		if err := s.PutRefresh(ctx, b); err != nil {
			t.Fatal(err)
		}
		if err := s.PutRefresh(ctx, other); err != nil {
			t.Fatal(err)
		}
		live := domain.RefreshToken{Hash: id.New(), FamilyID: fam, ClientID: "c", UserID: "u", Scopes: []string{"openid"}, ExpiresAt: now.Add(time.Hour)}
		if err := s.PutRefresh(ctx, live); err != nil {
			t.Fatal(err)
		}
		taken, err := s.TakeRefresh(ctx, live.Hash)
		if err != nil || taken.FamilyID != fam {
			t.Fatalf("take live: %v %+v", err, taken)
		}
		if _, err := s.TakeRefresh(ctx, live.Hash); err != domain.ErrUsed {
			t.Fatalf("second take: %v", err)
		}
		if err := s.RevokeFamily(ctx, fam); err != nil {
			t.Fatal(err)
		}
		a2, _ := s.GetRefresh(ctx, a.Hash)
		b2, _ := s.GetRefresh(ctx, b.Hash)
		o2, _ := s.GetRefresh(ctx, other.Hash)
		if !a2.Revoked || !b2.Revoked || o2.Revoked {
			t.Fatalf("family a=%v b=%v other=%v", a2.Revoked, b2.Revoked, o2.Revoked)
		}
	})

	t.Run("session client consent access", func(t *testing.T) {
		u := domain.User{ID: id.New(), Email: id.New() + "@example.com", Name: "B", PasswordHash: "x", CreatedAt: now}
		if err := s.Create(ctx, u); err != nil {
			t.Fatal(err)
		}
		sess := domain.Session{TokenHash: id.New(), UserID: u.ID, ExpiresAt: now.Add(time.Hour)}
		if err := s.PutSession(ctx, sess); err != nil {
			t.Fatal(err)
		}
		gotS, err := s.GetSession(ctx, sess.TokenHash)
		if err != nil || gotS.UserID != u.ID {
			t.Fatalf("session %v %+v", err, gotS)
		}
		if err := s.DeleteSession(ctx, sess.TokenHash); err != nil {
			t.Fatal(err)
		}
		if _, err := s.GetSession(ctx, sess.TokenHash); err != domain.ErrNotFound {
			t.Fatalf("deleted session: %v", err)
		}

		c := domain.Client{ID: id.New(), Name: "RP", Type: domain.ClientPublic, RedirectURIs: []string{"http://127.0.0.1/cb", "http://127.0.0.1/cb2"}}
		if err := s.CreateClient(ctx, c); err != nil {
			t.Fatal(err)
		}
		gotC, err := s.GetClient(ctx, c.ID)
		if err != nil || len(gotC.RedirectURIs) != 2 || gotC.Type != domain.ClientPublic {
			t.Fatalf("client %v %+v", err, gotC)
		}
		if err := s.CreateClient(ctx, c); err != domain.ErrConflict {
			t.Fatalf("client conflict: %v", err)
		}

		if err := s.PutConsent(ctx, domain.Consent{UserID: u.ID, ClientID: c.ID, Scopes: []string{"openid"}}); err != nil {
			t.Fatal(err)
		}
		gotN, err := s.GetConsent(ctx, u.ID, c.ID)
		if err != nil || !contains(gotN.Scopes, "openid") {
			t.Fatalf("consent %v %+v", err, gotN)
		}

		at := domain.AccessToken{Hash: id.New(), ClientID: c.ID, UserID: u.ID, Scopes: []string{"openid"}, ExpiresAt: now.Add(time.Hour)}
		if err := s.PutAccess(ctx, at); err != nil {
			t.Fatal(err)
		}
		gotA, err := s.GetAccess(ctx, at.Hash)
		if err != nil || gotA.UserID != u.ID {
			t.Fatalf("access %v %+v", err, gotA)
		}
	})
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
