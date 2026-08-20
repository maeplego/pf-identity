package web

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/oauth"
	"github.com/portfolio/pf-identity-server/internal/random"
)

func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirectURI := q.Get("redirect_uri")
	clientID := q.Get("client_id")
	state := q.Get("state")
	if q.Get("response_type") != "code" {
		s.oauthFail(w, r, redirectURI, state, "unsupported_response_type", "only response_type=code is supported")
		return
	}
	if q.Get("code_challenge_method") != oauth.ChallengeS256 || q.Get("code_challenge") == "" {
		s.oauthFail(w, r, redirectURI, state, "invalid_request", "PKCE S256 is required")
		return
	}
	client, err := s.Repos.GetClient(r.Context(), clientID)
	if err != nil {
		http.Error(w, "unknown client", http.StatusBadRequest)
		return
	}
	if _, err := oauth.ParseRedirectURI(redirectURI); err != nil || !oauth.RedirectURIExact(redirectURI, client.RedirectURIs) {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	scopes, err := oauth.NormalizeScopes(q.Get("scope"), true)
	if err != nil {
		s.oauthFail(w, r, redirectURI, state, "invalid_scope", err.Error())
		return
	}
	user, ok := s.currentUser(r)
	if !ok {
		next := url.QueryEscape(r.URL.RequestURI())
		http.Redirect(w, r, "/login?continue="+next, http.StatusSeeOther)
		return
	}

	pending := pendingAuth{
		ClientID:    client.ID,
		RedirectURI: redirectURI,
		State:       state,
		Nonce:       q.Get("nonce"),
		Scopes:      scopes,
		Challenge:   q.Get("code_challenge"),
		ExpiresAt:   s.now().Add(s.Cfg.CodeTTL * 10),
	}
	if s.hasConsent(r, user.ID, client.ID, scopes) {
		s.issueCodeRedirect(w, r, user.ID, pending)
		return
	}
	reqID, err := random.Token()
	if err != nil {
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	s.mu.Lock()
	s.pending[reqID] = pending
	s.mu.Unlock()
	tok, err := s.issueCSRF(w)
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	s.render(w, "consent", pageData{
		Title:      "同意",
		CSRF:       tok,
		ClientName: client.Name,
		Scopes:     scopes,
		ScopeViews: scopeViews(scopes),
		RequestID:  reqID,
	}, http.StatusOK)
}

func (s *Server) handleConsent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || !s.checkCSRF(r) {
		http.Error(w, "csrf", http.StatusBadRequest)
		return
	}
	user, ok := s.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	reqID := r.FormValue("request_id")
	s.mu.Lock()
	pending, okp := s.pending[reqID]
	if okp {
		delete(s.pending, reqID)
	}
	s.mu.Unlock()
	if !okp || !pending.ExpiresAt.After(s.now()) {
		http.Error(w, "expired authorization request", http.StatusBadRequest)
		return
	}
	if r.FormValue("decision") != "allow" {
		s.oauthFail(w, r, pending.RedirectURI, pending.State, "access_denied", "the user denied the request")
		return
	}
	_ = s.Repos.PutConsent(r.Context(), domain.Consent{UserID: user.ID, ClientID: pending.ClientID, Scopes: pending.Scopes})
	s.audit(r, domain.AuditConsent, user.ID, pending.ClientID, strings.Join(pending.Scopes, " "))
	s.issueCodeRedirect(w, r, user.ID, pending)
}

func (s *Server) issueCodeRedirect(w http.ResponseWriter, r *http.Request, userID string, p pendingAuth) {
	plain, err := random.Token()
	if err != nil {
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	sid := s.currentSID(r)
	row := domain.AuthCode{
		Hash:          oauth.HashToken(plain),
		ClientID:      p.ClientID,
		UserID:        userID,
		RedirectURI:   p.RedirectURI,
		Scopes:        p.Scopes,
		Nonce:         p.Nonce,
		CodeChallenge: p.Challenge,
		SessionSID:    sid,
		ExpiresAt:     s.now().Add(s.Cfg.CodeTTL),
	}
	if err := s.Repos.PutCode(r.Context(), row); err != nil {
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	_ = s.Repos.AddSessionClient(r.Context(), sid, p.ClientID)
	u, err := url.Parse(p.RedirectURI)
	if err != nil {
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	q := u.Query()
	q.Set("code", plain)
	if p.State != "" {
		q.Set("state", p.State)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (s *Server) hasConsent(r *http.Request, userID, clientID string, want []string) bool {
	c, err := s.Repos.GetConsent(r.Context(), userID, clientID)
	if err != nil {
		return false
	}
	for _, scope := range want {
		if !oauth.Contains(c.Scopes, scope) {
			return false
		}
	}
	return true
}

func (s *Server) oauthFail(w http.ResponseWriter, r *http.Request, redirectURI, state, code, desc string) {
	if redirectURI == "" {
		http.Error(w, code+": "+desc, http.StatusBadRequest)
		return
	}
	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, code, http.StatusBadRequest)
		return
	}
	q := u.Query()
	q.Set("error", code)
	q.Set("error_description", desc)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func clientError(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `","error_description":"` + jsonEscape(desc) + `"}`))
}

func jsonEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

var errClientAuth = errors.New("invalid client")
