package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/id"
	"github.com/portfolio/pf-identity-server/internal/oauth"
	"github.com/portfolio/pf-identity-server/internal/oidc"
	"github.com/portfolio/pf-identity-server/internal/password"
	"github.com/portfolio/pf-identity-server/internal/random"
)

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		clientError(w, http.StatusBadRequest, "invalid_request", "malformed body")
		return
	}
	grant := r.FormValue("grant_type")
	switch grant {
	case "authorization_code":
		s.tokenAuthCode(w, r)
	case "refresh_token":
		s.tokenRefresh(w, r)
	default:
		clientError(w, http.StatusBadRequest, "unsupported_grant_type", "use authorization_code or refresh_token")
	}
}

func (s *Server) tokenAuthCode(w http.ResponseWriter, r *http.Request) {
	client, err := s.authenticateClient(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="idp"`)
		clientError(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}
	plain := r.FormValue("code")
	if plain == "" {
		clientError(w, http.StatusBadRequest, "invalid_request", "code is required")
		return
	}
	row, err := s.Repos.TakeCode(r.Context(), oauth.HashToken(plain))
	if err != nil {
		clientError(w, http.StatusBadRequest, "invalid_grant", "code is invalid or already used")
		return
	}
	if !row.ExpiresAt.After(s.now()) {
		clientError(w, http.StatusBadRequest, "invalid_grant", "code expired")
		return
	}
	if row.ClientID != client.ID {
		clientError(w, http.StatusBadRequest, "invalid_grant", "code was issued to another client")
		return
	}
	if row.RedirectURI != r.FormValue("redirect_uri") {
		clientError(w, http.StatusBadRequest, "invalid_grant", "redirect_uri mismatch")
		return
	}
	if err := oauth.VerifyS256(r.FormValue("code_verifier"), row.CodeChallenge); err != nil {
		clientError(w, http.StatusBadRequest, "invalid_grant", "pkce verification failed")
		return
	}
	s.writeTokenResponse(w, r, client, row.UserID, row.Scopes, row.Nonce, oauth.Contains(row.Scopes, "offline_access"))
}

func (s *Server) tokenRefresh(w http.ResponseWriter, r *http.Request) {
	client, err := s.authenticateClient(r)
	if err != nil {
		clientError(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}
	plain := r.FormValue("refresh_token")
	row, err := s.Repos.GetRefresh(r.Context(), oauth.HashToken(plain))
	if err != nil {
		clientError(w, http.StatusBadRequest, "invalid_grant", "refresh token invalid")
		return
	}
	if row.Revoked || !row.ExpiresAt.After(s.now()) || row.ClientID != client.ID {
		_ = s.Repos.RevokeFamily(r.Context(), row.FamilyID)
		clientError(w, http.StatusBadRequest, "invalid_grant", "refresh token reuse or expiry")
		return
	}
	// Rotation: the presented token is retired by revoking the whole family then minting a child.
	// Reuse of an old token therefore kills the family (detected via Revoked on the next attempt).
	if err := s.Repos.RevokeFamily(r.Context(), row.FamilyID); err != nil {
		clientError(w, http.StatusInternalServerError, "server_error", "could not rotate")
		return
	}
	s.writeTokenResponse(w, r, client, row.UserID, row.Scopes, "", true)
}

func (s *Server) writeTokenResponse(w http.ResponseWriter, r *http.Request, client domain.Client, userID string, scopes []string, nonce string, withRefresh bool) {
	user, err := s.Repos.GetByID(r.Context(), userID)
	if err != nil {
		clientError(w, http.StatusBadRequest, "invalid_grant", "user missing")
		return
	}
	accessPlain, err := random.Token()
	if err != nil {
		clientError(w, http.StatusInternalServerError, "server_error", "token")
		return
	}
	if err := s.Repos.PutAccess(r.Context(), domain.AccessToken{
		Hash:      oauth.HashToken(accessPlain),
		ClientID:  client.ID,
		UserID:    userID,
		Scopes:    scopes,
		ExpiresAt: s.now().Add(s.Cfg.AccessTTL),
	}); err != nil {
		clientError(w, http.StatusInternalServerError, "server_error", "token")
		return
	}
	idTok, err := s.Signer.SignIDToken(oidcIDToken(s, user, client.ID, nonce))
	if err != nil {
		clientError(w, http.StatusInternalServerError, "server_error", "id_token")
		return
	}
	resp := map[string]any{
		"access_token": accessPlain,
		"token_type":   "Bearer",
		"expires_in":   int(s.Cfg.AccessTTL.Seconds()),
		"id_token":     idTok,
		"scope":        strings.Join(scopes, " "),
	}
	if withRefresh {
		refreshPlain, err := random.Token()
		if err != nil {
			clientError(w, http.StatusInternalServerError, "server_error", "refresh")
			return
		}
		family := id.New()
		if err := s.Repos.PutRefresh(r.Context(), domain.RefreshToken{
			Hash:      oauth.HashToken(refreshPlain),
			FamilyID:  family,
			ClientID:  client.ID,
			UserID:    userID,
			Scopes:    scopes,
			ExpiresAt: s.now().Add(s.Cfg.RefreshTTL),
		}); err != nil {
			clientError(w, http.StatusInternalServerError, "server_error", "refresh")
			return
		}
		resp["refresh_token"] = refreshPlain
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(resp)
}

func oidcIDToken(s *Server, user domain.User, aud, nonce string) oidc.IDTokenInput {
	return oidc.IDTokenInput{
		Issuer:   s.Cfg.Issuer,
		Subject:  user.ID,
		Audience: aud,
		Nonce:    nonce,
		Email:    user.Email,
		Name:     user.Name,
		Verified: user.EmailVerified,
		Now:      s.now(),
		TTL:      s.Cfg.AccessTTL,
	}
}

func (s *Server) authenticateClient(r *http.Request) (domain.Client, error) {
	id, secret, ok := r.BasicAuth()
	if !ok {
		id = r.FormValue("client_id")
		secret = r.FormValue("client_secret")
	}
	c, err := s.Repos.GetClient(r.Context(), id)
	if err != nil {
		return domain.Client{}, errClientAuth
	}
	if c.Type == domain.ClientConfidential {
		ok, verr := password.Verify(secret, c.SecretHash)
		if verr != nil || !ok {
			return domain.Client{}, errClientAuth
		}
		return c, nil
	}
	// Public clients must not send a secret; PKCE already proved possession of the verifier.
	if secret != "" {
		return domain.Client{}, errClientAuth
	}
	return c, nil
}

func (s *Server) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	raw := bearer(r)
	if raw == "" {
		w.Header().Set("WWW-Authenticate", "Bearer")
		clientError(w, http.StatusUnauthorized, "invalid_token", "missing bearer token")
		return
	}
	tok, err := s.Repos.GetAccess(r.Context(), oauth.HashToken(raw))
	if err != nil || !tok.ExpiresAt.After(s.now()) {
		clientError(w, http.StatusUnauthorized, "invalid_token", "access token invalid")
		return
	}
	user, err := s.Repos.GetByID(r.Context(), tok.UserID)
	if err != nil {
		clientError(w, http.StatusUnauthorized, "invalid_token", "user")
		return
	}
	body := map[string]any{"sub": user.ID}
	if oauth.Contains(tok.Scopes, "email") {
		body["email"] = user.Email
		body["email_verified"] = user.EmailVerified
	}
	if oauth.Contains(tok.Scopes, "profile") {
		body["name"] = user.Name
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) < 8 || !strings.EqualFold(h[:7], "bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}

