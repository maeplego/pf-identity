package memory

import (
	"context"
	"testing"

	"github.com/portfolio/pf-identity-server/internal/domain"
)

func TestUserCreateAndGet(t *testing.T) {
	s := NewStore()
	u := domain.User{ID: "01USER", Email: "A@Example.COM", Name: "A", PasswordHash: "x"}
	if err := s.Create(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetByEmail(context.Background(), "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "a@example.com" {
		t.Fatalf("email stored as %q", got.Email)
	}
	if err := s.Create(context.Background(), u); err != domain.ErrConflict {
		t.Fatalf("conflict: %v", err)
	}
}

func TestAuthCodeSingleUse(t *testing.T) {
	s := NewStore()
	c := domain.AuthCode{Hash: "h", ClientID: "c", UserID: "u"}
	if err := s.PutCode(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TakeCode(context.Background(), "h"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TakeCode(context.Background(), "h"); err != domain.ErrUsed {
		t.Fatalf("second take: %v", err)
	}
}

func TestRefreshFamilyRevoke(t *testing.T) {
	s := NewStore()
	_ = s.PutRefresh(context.Background(), domain.RefreshToken{Hash: "a", FamilyID: "fam"})
	_ = s.PutRefresh(context.Background(), domain.RefreshToken{Hash: "b", FamilyID: "fam"})
	_ = s.PutRefresh(context.Background(), domain.RefreshToken{Hash: "c", FamilyID: "other"})
	if err := s.RevokeFamily(context.Background(), "fam"); err != nil {
		t.Fatal(err)
	}
	a, _ := s.GetRefresh(context.Background(), "a")
	c, _ := s.GetRefresh(context.Background(), "c")
	if !a.Revoked || c.Revoked {
		t.Fatalf("revoked a=%v c=%v", a.Revoked, c.Revoked)
	}
}
