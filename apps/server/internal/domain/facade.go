package domain

// Repos is the full persistence surface of the IdP.
type Repos interface {
	Users
	Sessions
	Clients
	AuthCodes
	RefreshTokens
	AccessTokens
	Consents
	Audits
}
