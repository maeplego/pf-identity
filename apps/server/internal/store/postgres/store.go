// Package postgres is the durable domain.Repos implementation.
// Authorization codes are consumed with UPDATE ... used=false so two /token
// requests cannot both succeed even under concurrent load.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/portfolio/pf-identity-server/internal/domain"
)

// Store is a Postgres-backed Repos.
type Store struct {
	pool *pgxpool.Pool
}

// Open connects, migrates, and returns a store. Caller must Close.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the pool.
func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Create(ctx context.Context, u domain.User) error {
	u.Email = strings.ToLower(u.Email)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, email, name, password_hash, email_verified, disabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		u.ID, u.Email, u.Name, u.PasswordHash, u.EmailVerified, u.Disabled, u.CreatedAt.UTC(),
	)
	return mapErr(err)
}

func (s *Store) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return s.scanUser(s.pool.QueryRow(ctx, `
		SELECT id, email, name, password_hash, email_verified, disabled, created_at
		FROM users WHERE email = $1`, strings.ToLower(email)))
}

func (s *Store) GetByID(ctx context.Context, id string) (domain.User, error) {
	return s.scanUser(s.pool.QueryRow(ctx, `
		SELECT id, email, name, password_hash, email_verified, disabled, created_at
		FROM users WHERE id = $1`, id))
}

func (s *Store) scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.EmailVerified, &u.Disabled, &u.CreatedAt)
	if err != nil {
		return domain.User{}, mapErr(err)
	}
	return u, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, email, name, password_hash, email_verified, disabled, created_at
		FROM users ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.EmailVerified, &u.Disabled, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) SetUserDisabled(ctx context.Context, id string, disabled bool) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET disabled = $2 WHERE id = $1`, id, disabled)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) PutSession(ctx context.Context, sess domain.Session) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (token_hash) DO UPDATE SET user_id = EXCLUDED.user_id, expires_at = EXCLUDED.expires_at`,
		sess.TokenHash, sess.UserID, sess.ExpiresAt.UTC(),
	)
	return mapErr(err)
}

func (s *Store) GetSession(ctx context.Context, tokenHash string) (domain.Session, error) {
	var sess domain.Session
	err := s.pool.QueryRow(ctx, `
		SELECT token_hash, user_id, expires_at FROM sessions WHERE token_hash = $1`, tokenHash,
	).Scan(&sess.TokenHash, &sess.UserID, &sess.ExpiresAt)
	if err != nil {
		return domain.Session{}, mapErr(err)
	}
	return sess, nil
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return mapErr(err)
}

func (s *Store) CreateClient(ctx context.Context, c domain.Client) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO clients (id, name, type, secret_hash, redirect_uris, token_endpoint_auth)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		c.ID, c.Name, string(c.Type), c.SecretHash, c.RedirectURIs, c.TokenEndpointAuth,
	)
	return mapErr(err)
}

func (s *Store) GetClient(ctx context.Context, id string) (domain.Client, error) {
	var c domain.Client
	var typ string
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, type, secret_hash, redirect_uris, token_endpoint_auth
		FROM clients WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &typ, &c.SecretHash, &c.RedirectURIs, &c.TokenEndpointAuth)
	if err != nil {
		return domain.Client{}, mapErr(err)
	}
	c.Type = domain.ClientType(typ)
	return c, nil
}

func (s *Store) ListClients(ctx context.Context) ([]domain.Client, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, type, secret_hash, redirect_uris, token_endpoint_auth
		FROM clients ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Client
	for rows.Next() {
		var c domain.Client
		var typ string
		if err := rows.Scan(&c.ID, &c.Name, &typ, &c.SecretHash, &c.RedirectURIs, &c.TokenEndpointAuth); err != nil {
			return nil, err
		}
		c.Type = domain.ClientType(typ)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) UpdateClient(ctx context.Context, id, name string, redirectURIs []string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE clients SET name = $2, redirect_uris = $3 WHERE id = $1`, id, name, redirectURIs)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) SetClientSecret(ctx context.Context, id, secretHash string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE clients SET secret_hash = $2 WHERE id = $1`, id, secretHash)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) PutCode(ctx context.Context, c domain.AuthCode) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO auth_codes (hash, client_id, user_id, redirect_uri, scopes, nonce, code_challenge, expires_at, used)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		c.Hash, c.ClientID, c.UserID, c.RedirectURI, c.Scopes, c.Nonce, c.CodeChallenge, c.ExpiresAt.UTC(), c.Used,
	)
	return mapErr(err)
}

func (s *Store) TakeCode(ctx context.Context, hash string) (domain.AuthCode, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE auth_codes SET used = TRUE
		WHERE hash = $1 AND used = FALSE
		RETURNING hash, client_id, user_id, redirect_uri, scopes, nonce, code_challenge, expires_at, used`,
		hash,
	)
	c, err := scanCode(row)
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.AuthCode{}, err
	}
	var used bool
	err = s.pool.QueryRow(ctx, `SELECT used FROM auth_codes WHERE hash = $1`, hash).Scan(&used)
	if err != nil {
		return domain.AuthCode{}, mapErr(err)
	}
	return domain.AuthCode{}, domain.ErrUsed
}

func scanCode(row pgx.Row) (domain.AuthCode, error) {
	var c domain.AuthCode
	err := row.Scan(&c.Hash, &c.ClientID, &c.UserID, &c.RedirectURI, &c.Scopes, &c.Nonce, &c.CodeChallenge, &c.ExpiresAt, &c.Used)
	if err != nil {
		return domain.AuthCode{}, mapErr(err)
	}
	return c, nil
}

func (s *Store) PutRefresh(ctx context.Context, t domain.RefreshToken) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (hash, family_id, client_id, user_id, scopes, expires_at, revoked)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (hash) DO UPDATE SET
			family_id = EXCLUDED.family_id,
			client_id = EXCLUDED.client_id,
			user_id = EXCLUDED.user_id,
			scopes = EXCLUDED.scopes,
			expires_at = EXCLUDED.expires_at,
			revoked = EXCLUDED.revoked`,
		t.Hash, t.FamilyID, t.ClientID, t.UserID, t.Scopes, t.ExpiresAt.UTC(), t.Revoked,
	)
	return mapErr(err)
}

func (s *Store) GetRefresh(ctx context.Context, hash string) (domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := s.pool.QueryRow(ctx, `
		SELECT hash, family_id, client_id, user_id, scopes, expires_at, revoked
		FROM refresh_tokens WHERE hash = $1`, hash,
	).Scan(&t.Hash, &t.FamilyID, &t.ClientID, &t.UserID, &t.Scopes, &t.ExpiresAt, &t.Revoked)
	if err != nil {
		return domain.RefreshToken{}, mapErr(err)
	}
	return t, nil
}

func (s *Store) TakeRefresh(ctx context.Context, hash string) (domain.RefreshToken, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE refresh_tokens SET revoked = TRUE
		WHERE hash = $1 AND revoked = FALSE
		RETURNING hash, family_id, client_id, user_id, scopes, expires_at, revoked`,
		hash,
	)
	var t domain.RefreshToken
	err := row.Scan(&t.Hash, &t.FamilyID, &t.ClientID, &t.UserID, &t.Scopes, &t.ExpiresAt, &t.Revoked)
	if err == nil {
		return t, nil
	}
	if !errors.Is(mapErr(err), domain.ErrNotFound) {
		return domain.RefreshToken{}, mapErr(err)
	}
	var revoked bool
	err = s.pool.QueryRow(ctx, `SELECT revoked FROM refresh_tokens WHERE hash = $1`, hash).Scan(&revoked)
	if err != nil {
		return domain.RefreshToken{}, mapErr(err)
	}
	return domain.RefreshToken{}, domain.ErrUsed
}

func (s *Store) RevokeFamily(ctx context.Context, familyID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked = TRUE WHERE family_id = $1`, familyID)
	return mapErr(err)
}

func (s *Store) PutAccess(ctx context.Context, t domain.AccessToken) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO access_tokens (hash, client_id, user_id, scopes, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (hash) DO UPDATE SET
			client_id = EXCLUDED.client_id,
			user_id = EXCLUDED.user_id,
			scopes = EXCLUDED.scopes,
			expires_at = EXCLUDED.expires_at`,
		t.Hash, t.ClientID, t.UserID, t.Scopes, t.ExpiresAt.UTC(),
	)
	return mapErr(err)
}

func (s *Store) GetAccess(ctx context.Context, hash string) (domain.AccessToken, error) {
	var t domain.AccessToken
	err := s.pool.QueryRow(ctx, `
		SELECT hash, client_id, user_id, scopes, expires_at
		FROM access_tokens WHERE hash = $1`, hash,
	).Scan(&t.Hash, &t.ClientID, &t.UserID, &t.Scopes, &t.ExpiresAt)
	if err != nil {
		return domain.AccessToken{}, mapErr(err)
	}
	return t, nil
}

func (s *Store) PutConsent(ctx context.Context, c domain.Consent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO consents (user_id, client_id, scopes)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, client_id) DO UPDATE SET scopes = EXCLUDED.scopes`,
		c.UserID, c.ClientID, c.Scopes,
	)
	return mapErr(err)
}

func (s *Store) GetConsent(ctx context.Context, userID, clientID string) (domain.Consent, error) {
	var c domain.Consent
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, client_id, scopes FROM consents WHERE user_id = $1 AND client_id = $2`,
		userID, clientID,
	).Scan(&c.UserID, &c.ClientID, &c.Scopes)
	if err != nil {
		return domain.Consent{}, mapErr(err)
	}
	return c, nil
}

func (s *Store) AppendAudit(ctx context.Context, e domain.AuditEvent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_events (id, type, at, subject, client_id, ip, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.ID, e.Type, e.At.UTC(), e.Subject, e.ClientID, e.IP, e.Note,
	)
	return mapErr(err)
}

func (s *Store) ListAudits(ctx context.Context, limit int) ([]domain.AuditEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, type, at, subject, client_id, ip, note
		FROM audit_events ORDER BY at DESC, id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AuditEvent
	for rows.Next() {
		var e domain.AuditEvent
		if err := rows.Scan(&e.ID, &e.Type, &e.At, &e.Subject, &e.ClientID, &e.IP, &e.Note); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrConflict
	}
	return err
}
