// Package account registers users and checks passwords.
package account

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/portfolio/pf-identity-server/internal/clock"
	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/id"
	"github.com/portfolio/pf-identity-server/internal/password"
)

// Service creates and authenticates users.
type Service struct {
	Users domain.Users
	Clock clock.Clock
}

// RegisterInput is a new account.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

// Register stores a new user. Duplicate emails become domain.ErrConflict.
func (s *Service) Register(ctx context.Context, in RegisterInput) (domain.User, error) {
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return domain.User{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return domain.User{}, fmt.Errorf("name is required")
	}
	hash, err := password.Hash(in.Password)
	if err != nil {
		return domain.User{}, err
	}
	u := domain.User{
		ID:           id.New(),
		Email:        email,
		Name:         name,
		PasswordHash: hash,
		CreatedAt:    s.Clock.Now(),
	}
	if err := s.Users.Create(ctx, u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// Authenticate returns the user if the password matches and the account is enabled.
func (s *Service) Authenticate(ctx context.Context, email, plain string) (domain.User, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return domain.User{}, err
	}
	u, err := s.Users.GetByEmail(ctx, email)
	if err != nil {
		// Same error shape as a bad password so callers cannot probe registered emails.
		return domain.User{}, errInvalidCreds
	}
	if u.Disabled {
		return domain.User{}, errInvalidCreds
	}
	ok, err := password.Verify(plain, u.PasswordHash)
	if err != nil || !ok {
		return domain.User{}, errInvalidCreds
	}
	return u, nil
}

func normalizeEmail(raw string) (string, error) {
	e := strings.ToLower(strings.TrimSpace(raw))
	if _, err := mail.ParseAddress(e); err != nil {
		return "", fmt.Errorf("invalid email")
	}
	if !strings.Contains(e, "@") {
		return "", fmt.Errorf("invalid email")
	}
	return e, nil
}
