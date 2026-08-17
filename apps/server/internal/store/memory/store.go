// Package memory is a process-local implementation of domain repositories.
// It exists so HTTP tests do not need Postgres, and so a laptop demo can boot
// without Docker. Data vanishes on restart; that is intentional.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/portfolio/pf-identity-server/internal/domain"
)

// Store implements all IdP repositories behind one mutex.
type Store struct {
	mu       sync.RWMutex
	users    map[string]domain.User // id
	byEmail  map[string]string
	sessions map[string]domain.Session
	clients  map[string]domain.Client
	codes    map[string]domain.AuthCode
	refresh  map[string]domain.RefreshToken
	access   map[string]domain.AccessToken
	consent  map[string]domain.Consent
	audits   []domain.AuditEvent
}

// NewStore returns an empty store.
func NewStore() *Store {
	return &Store{
		users:    map[string]domain.User{},
		byEmail:  map[string]string{},
		sessions: map[string]domain.Session{},
		clients:  map[string]domain.Client{},
		codes:    map[string]domain.AuthCode{},
		refresh:  map[string]domain.RefreshToken{},
		access:   map[string]domain.AccessToken{},
		consent:  map[string]domain.Consent{},
		audits:   []domain.AuditEvent{},
	}
}

func (s *Store) Create(_ context.Context, u domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	email := strings.ToLower(u.Email)
	if _, ok := s.byEmail[email]; ok {
		return domain.ErrConflict
	}
	u.Email = email
	s.users[u.ID] = u
	s.byEmail[email] = u.ID
	return nil
}

func (s *Store) GetByEmail(_ context.Context, email string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byEmail[strings.ToLower(email)]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return s.users[id], nil
}

func (s *Store) GetByID(_ context.Context, id string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (s *Store) ListUsers(_ context.Context) ([]domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	sortUsers(out)
	return out, nil
}

func (s *Store) SetUserDisabled(_ context.Context, id string, disabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	u.Disabled = disabled
	s.users[id] = u
	return nil
}

func (s *Store) PutSession(_ context.Context, sess domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.TokenHash] = sess
	return nil
}

func (s *Store) GetSession(_ context.Context, tokenHash string) (domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[tokenHash]
	if !ok {
		return domain.Session{}, domain.ErrNotFound
	}
	return sess, nil
}

func (s *Store) DeleteSession(_ context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, tokenHash)
	return nil
}

func (s *Store) CreateClient(_ context.Context, c domain.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clients[c.ID]; ok {
		return domain.ErrConflict
	}
	s.clients[c.ID] = c
	return nil
}

func (s *Store) GetClient(_ context.Context, id string) (domain.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clients[id]
	if !ok {
		return domain.Client{}, domain.ErrNotFound
	}
	return c, nil
}

func (s *Store) ListClients(_ context.Context) ([]domain.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Client, 0, len(s.clients))
	for _, c := range s.clients {
		out = append(out, c)
	}
	sortClients(out)
	return out, nil
}

func (s *Store) UpdateClient(_ context.Context, id, name string, redirectURIs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	if !ok {
		return domain.ErrNotFound
	}
	c.Name = name
	c.RedirectURIs = append([]string{}, redirectURIs...)
	s.clients[id] = c
	return nil
}

func (s *Store) SetClientSecret(_ context.Context, id, secretHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	if !ok {
		return domain.ErrNotFound
	}
	c.SecretHash = secretHash
	s.clients[id] = c
	return nil
}

func (s *Store) PutCode(_ context.Context, c domain.AuthCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[c.Hash] = c
	return nil
}

func (s *Store) TakeCode(_ context.Context, hash string) (domain.AuthCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[hash]
	if !ok {
		return domain.AuthCode{}, domain.ErrNotFound
	}
	if c.Used {
		return domain.AuthCode{}, domain.ErrUsed
	}
	c.Used = true
	s.codes[hash] = c
	return c, nil
}

func (s *Store) PutRefresh(_ context.Context, t domain.RefreshToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refresh[t.Hash] = t
	return nil
}

func (s *Store) GetRefresh(_ context.Context, hash string) (domain.RefreshToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.refresh[hash]
	if !ok {
		return domain.RefreshToken{}, domain.ErrNotFound
	}
	return t, nil
}

func (s *Store) TakeRefresh(_ context.Context, hash string) (domain.RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.refresh[hash]
	if !ok {
		return domain.RefreshToken{}, domain.ErrNotFound
	}
	if t.Revoked {
		return domain.RefreshToken{}, domain.ErrUsed
	}
	t.Revoked = true
	s.refresh[hash] = t
	return t, nil
}

func (s *Store) RevokeFamily(_ context.Context, familyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, t := range s.refresh {
		if t.FamilyID == familyID {
			t.Revoked = true
			s.refresh[k] = t
		}
	}
	return nil
}

func (s *Store) PutAccess(_ context.Context, t domain.AccessToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.access[t.Hash] = t
	return nil
}

func (s *Store) GetAccess(_ context.Context, hash string) (domain.AccessToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.access[hash]
	if !ok {
		return domain.AccessToken{}, domain.ErrNotFound
	}
	return t, nil
}

func (s *Store) PutConsent(_ context.Context, c domain.Consent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consent[c.UserID+"|"+c.ClientID] = c
	return nil
}

func (s *Store) GetConsent(_ context.Context, userID, clientID string) (domain.Consent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.consent[userID+"|"+clientID]
	if !ok {
		return domain.Consent{}, domain.ErrNotFound
	}
	return c, nil
}

func (s *Store) AppendAudit(_ context.Context, e domain.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, e)
	return nil
}

func (s *Store) ListAudits(_ context.Context, limit int) ([]domain.AuditEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	n := len(s.audits)
	if limit > n {
		limit = n
	}
	out := make([]domain.AuditEvent, 0, limit)
	for i := n - 1; i >= n-limit; i-- {
		out = append(out, s.audits[i])
	}
	return out, nil
}

func sortUsers(out []domain.User) {
	sort.Slice(out, func(i, j int) bool { return out[i].Email < out[j].Email })
}

func sortClients(out []domain.Client) {
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
}
