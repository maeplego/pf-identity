package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/oauth"
	"github.com/portfolio/pf-identity-server/internal/org"
	"github.com/portfolio/pf-identity-server/internal/session"
)

func (s *Server) orgService() *org.Service {
	return &org.Service{Repos: s.Repos, Clock: s.Clock}
}

type bearerUser struct {
	User   domain.User
	Scopes []string
}

func (s *Server) bearerUser(r *http.Request) (bearerUser, bool) {
	raw := bearer(r)
	if raw == "" {
		return bearerUser{}, false
	}
	tok, err := s.Repos.GetAccess(r.Context(), oauth.HashToken(raw))
	if err != nil || !tok.ExpiresAt.After(s.now()) {
		return bearerUser{}, false
	}
	user, err := s.Repos.GetByID(r.Context(), tok.UserID)
	if err != nil || user.Disabled {
		return bearerUser{}, false
	}
	return bearerUser{User: user, Scopes: tok.Scopes}, true
}

func (s *Server) handleCreateOrganization(w http.ResponseWriter, r *http.Request) {
	u, ok := s.bearerUser(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		clientError(w, http.StatusUnauthorized, "invalid_token", "bearer required")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		clientError(w, http.StatusBadRequest, "invalid_request", "json")
		return
	}
	created, err := s.orgService().Create(r.Context(), u.User.ID, body.Name)
	if err != nil {
		if err == domain.ErrInvalid {
			clientError(w, http.StatusBadRequest, "invalid_request", "name")
			return
		}
		clientError(w, http.StatusInternalServerError, "server_error", "org")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleListOrganizations(w http.ResponseWriter, r *http.Request) {
	u, ok := s.bearerUser(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		clientError(w, http.StatusUnauthorized, "invalid_token", "bearer required")
		return
	}
	list, err := s.orgService().ListForUser(r.Context(), u.User.ID)
	if err != nil {
		clientError(w, http.StatusInternalServerError, "server_error", "org")
		return
	}
	if list == nil {
		list = []domain.Organization{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"organizations": list})
}

func (s *Server) handleListOrganizationMembers(w http.ResponseWriter, r *http.Request) {
	u, ok := s.bearerUser(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		clientError(w, http.StatusUnauthorized, "invalid_token", "bearer required")
		return
	}
	orgID := strings.TrimPrefix(r.URL.Path, "/v1/organizations/")
	orgID = strings.TrimSuffix(orgID, "/members")
	members, err := s.orgService().ListMembers(r.Context(), u.User.ID, orgID)
	if err != nil {
		if err == domain.ErrForbidden {
			clientError(w, http.StatusForbidden, "access_denied", "not a member")
			return
		}
		clientError(w, http.StatusInternalServerError, "server_error", "org")
		return
	}
	if members == nil {
		members = []domain.OrgMemberDetail{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members})
}

func (s *Server) handleSetActiveOrg(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	c, err := r.Cookie(session.CookieName())
	if err != nil {
		clientError(w, http.StatusUnauthorized, "invalid_request", "session")
		return
	}
	var body struct {
		OrgID string `json:"orgId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		clientError(w, http.StatusBadRequest, "invalid_request", "json")
		return
	}
	if err := s.orgService().SetActiveOrg(r.Context(), user.ID, oauth.HashToken(c.Value), strings.TrimSpace(body.OrgID)); err != nil {
		if err == domain.ErrForbidden {
			clientError(w, http.StatusForbidden, "access_denied", "not a member")
			return
		}
		clientError(w, http.StatusBadRequest, "invalid_request", "org")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) appendOrgClaims(ctx context.Context, body map[string]any, userID, sessionSID string, scopes []string) {
	if !oauth.Contains(scopes, "org") {
		return
	}
	primary, all, err := org.PrimaryOrg(ctx, s.Repos, userID, sessionSID)
	if err != nil {
		return
	}
	body["org_id"] = primary.OrgID
	body["org_role"] = primary.Role
	if len(all) > 0 {
		body["organizations"] = all
	}
}
