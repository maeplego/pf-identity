package account

import (
	"context"
	"testing"
	"time"

	"github.com/portfolio/pf-identity-server/internal/clock"
	"github.com/portfolio/pf-identity-server/internal/store/memory"
)

func newSvc() *Service {
	return &Service{Users: memory.NewStore(), Clock: clock.Fixed{T: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}}
}

func TestRegisterAndAuthenticate(t *testing.T) {
	s := newSvc()
	u, err := s.Register(context.Background(), RegisterInput{Email: "Dev@Example.com", Password: "long-enough", Name: "Dev"})
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "dev@example.com" {
		t.Fatalf("email %q", u.Email)
	}
	got, err := s.Authenticate(context.Background(), "dev@example.com", "long-enough")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID {
		t.Fatalf("id %s", got.ID)
	}
}

func TestAuthenticateWrongPassword(t *testing.T) {
	s := newSvc()
	_, err := s.Register(context.Background(), RegisterInput{Email: "a@b.co", Password: "long-enough", Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Authenticate(context.Background(), "a@b.co", "nope")
	if !IsInvalidCredentials(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestAuthenticateUnknownUserLooksTheSame(t *testing.T) {
	s := newSvc()
	_, err := s.Authenticate(context.Background(), "missing@b.co", "long-enough")
	if !IsInvalidCredentials(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestRegisterRejectsBadEmail(t *testing.T) {
	s := newSvc()
	if _, err := s.Register(context.Background(), RegisterInput{Email: "not-an-email", Password: "long-enough", Name: "A"}); err == nil {
		t.Fatal("expected email error")
	}
}

func TestDuplicateEmail(t *testing.T) {
	s := newSvc()
	in := RegisterInput{Email: "a@b.co", Password: "long-enough", Name: "A"}
	if _, err := s.Register(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Register(context.Background(), in); err == nil {
		t.Fatal("expected conflict")
	}
}
