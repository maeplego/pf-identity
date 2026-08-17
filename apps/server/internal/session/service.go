// Package session issues hashed browser sessions for the IdP login cookie.
package session

import (
	"context"
	"fmt"
	"time"

	"github.com/portfolio/pf-identity-server/internal/clock"
	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/oauth"
	"github.com/portfolio/pf-identity-server/internal/random"
)

const cookieName = "idp_session"

// CookieName is the IdP session cookie. It is not an OAuth access token.
func CookieName() string { return cookieName }

// Service writes and reads sessions.
type Service struct {
	Sessions domain.Sessions
	Clock    clock.Clock
	TTL      time.Duration
}

// Start creates a session and returns the plaintext cookie value.
func (s *Service) Start(ctx context.Context, userID string) (plain string, expires time.Time, err error) {
	plain, err = random.Token()
	if err != nil {
		return "", time.Time{}, err
	}
	sid, err := random.Token()
	if err != nil {
		return "", time.Time{}, err
	}
	exp := s.Clock.Now().Add(s.TTL)
	row := domain.Session{TokenHash: oauth.HashToken(plain), UserID: userID, SID: sid, ExpiresAt: exp}
	if err := s.Sessions.PutSession(ctx, row); err != nil {
		return "", time.Time{}, err
	}
	return plain, exp, nil
}

// Lookup returns the user id for a still-valid session cookie.
func (s *Service) Lookup(ctx context.Context, plain string) (string, error) {
	row, err := s.LookupSession(ctx, plain)
	if err != nil {
		return "", err
	}
	return row.UserID, nil
}

// LookupSession returns the full session row so logout can fan out by sid.
func (s *Service) LookupSession(ctx context.Context, plain string) (domain.Session, error) {
	if plain == "" {
		return domain.Session{}, domain.ErrNotFound
	}
	row, err := s.Sessions.GetSession(ctx, oauth.HashToken(plain))
	if err != nil {
		return domain.Session{}, err
	}
	if !row.ExpiresAt.After(s.Clock.Now()) {
		_ = s.Sessions.DeleteSession(ctx, row.TokenHash)
		return domain.Session{}, domain.ErrNotFound
	}
	return row, nil
}

// End forgets the session. Missing rows are ignored so logout is idempotent.
func (s *Service) End(ctx context.Context, plain string) error {
	if plain == "" {
		return nil
	}
	if err := s.Sessions.DeleteSession(ctx, oauth.HashToken(plain)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
