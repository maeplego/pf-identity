package domain

import "context"

// Users persists accounts.
type Users interface {
	Create(ctx context.Context, u User) error
	GetByEmail(ctx context.Context, email string) (User, error)
	GetByID(ctx context.Context, id string) (User, error)
	ListUsers(ctx context.Context) ([]User, error)
	SetUserDisabled(ctx context.Context, id string, disabled bool) error
}

// Sessions persists IdP browser sessions, keyed by token hash.
type Sessions interface {
	PutSession(ctx context.Context, s Session) error
	GetSession(ctx context.Context, tokenHash string) (Session, error)
	DeleteSession(ctx context.Context, tokenHash string) error
}

// Clients persists relying parties.
type Clients interface {
	CreateClient(ctx context.Context, c Client) error
	GetClient(ctx context.Context, id string) (Client, error)
	ListClients(ctx context.Context) ([]Client, error)
	UpdateClient(ctx context.Context, id, name string, redirectURIs []string) error
	SetClientSecret(ctx context.Context, id, secretHash string) error
}

// AuthCodes persists hashed authorization codes.
type AuthCodes interface {
	PutCode(ctx context.Context, c AuthCode) error
	TakeCode(ctx context.Context, hash string) (AuthCode, error)
}

// RefreshTokens persists hashed refresh tokens.
type RefreshTokens interface {
	PutRefresh(ctx context.Context, t RefreshToken) error
	GetRefresh(ctx context.Context, hash string) (RefreshToken, error)
	// TakeRefresh marks one still-active token revoked so concurrent rotation cannot mint two children.
	TakeRefresh(ctx context.Context, hash string) (RefreshToken, error)
	RevokeFamily(ctx context.Context, familyID string) error
}

// AccessTokens persists hashed opaque access tokens.
type AccessTokens interface {
	PutAccess(ctx context.Context, t AccessToken) error
	GetAccess(ctx context.Context, hash string) (AccessToken, error)
}

// Consents persists user-to-client grants.
type Consents interface {
	PutConsent(ctx context.Context, c Consent) error
	GetConsent(ctx context.Context, userID, clientID string) (Consent, error)
}

// Audits persists operator-facing events. Append must not store secrets.
type Audits interface {
	AppendAudit(ctx context.Context, e AuditEvent) error
	// ListAudits returns newest-first pages. AfterID is the last id from the previous page.
	ListAudits(ctx context.Context, limit int, afterID string) (AuditPage, error)
}

// AuditPage is one newest-first window. Next is empty when there is no older page.
type AuditPage struct {
	Items []AuditEvent `json:"items"`
	Next  string       `json:"next"`
}
