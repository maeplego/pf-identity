// Package domain is the persistence-agnostic model of the IdP.
package domain

import "time"

// User is an account at this issuer. Email is unique and stored lowercased.
type User struct {
	ID            string
	Email         string
	Name          string
	PasswordHash  string
	EmailVerified bool
	Disabled      bool
	CreatedAt     time.Time
}

// ClientType distinguishes secret-bearing confidential RPs from public RPs (PKCE-only).
type ClientType string

const (
	ClientConfidential ClientType = "confidential"
	ClientPublic       ClientType = "public"
)

// Client is an OAuth client (relying party).
type Client struct {
	ID                     string
	Name                   string
	Type                   ClientType
	SecretHash             string
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	FrontChannelLogoutURI  string
	BackChannelLogoutURI   string
	TokenEndpointAuth      string
}

// ClientPatch is the mutable registration fields (secret and type stay put).
type ClientPatch struct {
	Name                   string
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	FrontChannelLogoutURI  string
	BackChannelLogoutURI   string
}

// Session is a browser login at the IdP (not an OAuth access token).
type Session struct {
	TokenHash   string
	UserID      string
	SID         string
	ActiveOrgID string
	ExpiresAt   time.Time
}

// OrgRole is tenant-scoped membership at an organization.
type OrgRole string

const (
	OrgRoleOwner  OrgRole = "owner"
	OrgRoleMember OrgRole = "member"
)

// Organization is a tenant boundary shared across relying parties.
type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// OrganizationMembership binds a global user to an organization with a tenant role.
type OrganizationMembership struct {
	OrgID    string
	UserID   string
	Role     OrgRole
	JoinedAt time.Time
}

// OrgMemberDetail is the public org-member list item for relying parties (invite UX).
type OrgMemberDetail struct {
	UserID   string    `json:"userId"`
	Role     string    `json:"role"`
	Email    string    `json:"email,omitempty"`
	Name     string    `json:"name,omitempty"`
	JoinedAt time.Time `json:"joinedAt"`
}

// OrgMembershipView is membership with org metadata for userinfo.
type OrgMembershipView struct {
	OrgID   string `json:"org_id"`
	OrgName string `json:"org_name"`
	Role    string `json:"role"`
}

// AuthCode is a one-time authorization code. The plaintext is never stored.
type AuthCode struct {
	Hash          string
	ClientID      string
	UserID        string
	RedirectURI   string
	Scopes        []string
	Nonce         string
	CodeChallenge string
	SessionSID    string
	ExpiresAt     time.Time
	Used          bool
}

// RefreshToken is rotated on use. FamilyID groups a chain so reuse can revoke all.
type RefreshToken struct {
	Hash       string
	FamilyID   string
	ClientID   string
	UserID     string
	Scopes     []string
	SessionSID string
	ExpiresAt  time.Time
	Revoked    bool
}

// AccessToken is opaque. Resource servers call UserInfo or introspect later.
type AccessToken struct {
	Hash       string
	ClientID   string
	UserID     string
	Scopes     []string
	SessionSID string
	ExpiresAt  time.Time
}

// Consent records that a user allowed a client a set of scopes.
type Consent struct {
	UserID   string
	ClientID string
	Scopes   []string
}

// Audit event types from DESIGN.md. Notes must never contain secrets.
const (
	AuditLoginFail  = "login_fail"
	AuditConsent    = "consent"
	AuditTokenIssue = "token_issue"
	AuditRevoke     = "revoke"
)

// AuditEvent is an operator-facing trail. Passwords and tokens are not stored.
type AuditEvent struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"`
	At       time.Time `json:"at"`
	Subject  string    `json:"subject"`
	ClientID string    `json:"client_id"`
	IP       string    `json:"ip"`
	Note     string    `json:"note"`
}
