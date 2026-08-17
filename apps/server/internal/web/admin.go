package web

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/id"
	"github.com/portfolio/pf-identity-server/internal/oauth"
	"github.com/portfolio/pf-identity-server/internal/password"
	"github.com/portfolio/pf-identity-server/internal/random"
)

var clientIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

type adminClientView struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Type              string   `json:"type"`
	RedirectURIs      []string `json:"redirect_uris"`
	TokenEndpointAuth string   `json:"token_endpoint_auth"`
	HasSecret         bool     `json:"has_secret"`
}

type adminUserView struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Disabled  bool   `json:"disabled"`
	CreatedAt string `json:"created_at"`
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if s.Cfg.AdminToken == "" {
		http.Error(w, "admin api disabled", http.StatusUnauthorized)
		return false
	}
	got := bearer(r)
	if len(got) != len(s.Cfg.AdminToken) || subtle.ConstantTimeCompare([]byte(got), []byte(s.Cfg.AdminToken)) != 1 {
		w.Header().Set("WWW-Authenticate", `Bearer realm="admin"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func (s *Server) handleAdminListClients(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	list, err := s.Repos.ListClients(r.Context())
	if err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	out := make([]adminClientView, 0, len(list))
	for _, c := range list {
		out = append(out, viewClient(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAdminCreateClient(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var in struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Type         string   `json:"type"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := decodeJSON(r, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	cid := strings.TrimSpace(in.ID)
	if cid == "" {
		cid = strings.ToLower(id.New())
	}
	if !clientIDPattern.MatchString(cid) {
		http.Error(w, "id must be lowercase alphanumeric with hyphens", http.StatusBadRequest)
		return
	}
	uris, err := normalizeRedirects(in.RedirectURIs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	c := domain.Client{ID: cid, Name: name, RedirectURIs: uris}
	plainSecret := ""
	switch domain.ClientType(in.Type) {
	case domain.ClientConfidential:
		c.Type = domain.ClientConfidential
		c.TokenEndpointAuth = "client_secret_basic"
		plainSecret, err = random.Token()
		if err != nil {
			http.Error(w, "secret", http.StatusInternalServerError)
			return
		}
		hash, err := password.Hash(plainSecret)
		if err != nil {
			http.Error(w, "secret", http.StatusInternalServerError)
			return
		}
		c.SecretHash = hash
	case domain.ClientPublic, "":
		c.Type = domain.ClientPublic
		c.TokenEndpointAuth = "none"
	default:
		http.Error(w, "type must be public or confidential", http.StatusBadRequest)
		return
	}
	if err := s.Repos.CreateClient(r.Context(), c); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			http.Error(w, "client id already exists", http.StatusConflict)
			return
		}
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	body := map[string]any{"client": viewClient(c)}
	if plainSecret != "" {
		body["client_secret"] = plainSecret
	}
	writeJSON(w, http.StatusCreated, body)
}

func (s *Server) handleAdminGetClient(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	c, err := s.Repos.GetClient(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, viewClient(c))
}

func (s *Server) handleAdminUpdateClient(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var in struct {
		Name         string   `json:"name"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := decodeJSON(r, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	uris, err := normalizeRedirects(in.RedirectURIs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Repos.UpdateClient(r.Context(), r.PathValue("id"), name, uris); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	c, err := s.Repos.GetClient(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, viewClient(c))
}

func (s *Server) handleAdminRotateSecret(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	c, err := s.Repos.GetClient(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if c.Type != domain.ClientConfidential {
		http.Error(w, "public clients have no secret", http.StatusBadRequest)
		return
	}
	plain, err := random.Token()
	if err != nil {
		http.Error(w, "secret", http.StatusInternalServerError)
		return
	}
	hash, err := password.Hash(plain)
	if err != nil {
		http.Error(w, "secret", http.StatusInternalServerError)
		return
	}
	if err := s.Repos.SetClientSecret(r.Context(), c.ID, hash); err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"client_secret": plain})
}

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	list, err := s.Repos.ListUsers(r.Context())
	if err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	out := make([]adminUserView, 0, len(list))
	for _, u := range list {
		out = append(out, viewUser(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAdminSetDisabled(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var in struct {
		Disabled bool `json:"disabled"`
	}
	if err := decodeJSON(r, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := s.Repos.SetUserDisabled(r.Context(), r.PathValue("id"), in.Disabled); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	u, err := s.Repos.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, viewUser(u))
}

func (s *Server) handleAdminListAudits(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	list, err := s.Repos.ListAudits(r.Context(), 100)
	if err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func viewClient(c domain.Client) adminClientView {
	uris := c.RedirectURIs
	if uris == nil {
		uris = []string{}
	}
	return adminClientView{
		ID:                c.ID,
		Name:              c.Name,
		Type:              string(c.Type),
		RedirectURIs:      uris,
		TokenEndpointAuth: c.TokenEndpointAuth,
		HasSecret:         c.SecretHash != "",
	}
}

func viewUser(u domain.User) adminUserView {
	return adminUserView{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Disabled:  u.Disabled,
		CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func normalizeRedirects(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, errors.New("at least one redirect_uri is required")
	}
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, err := oauth.ParseRedirectURI(item); err != nil {
			return nil, errors.New("redirect_uri must be an absolute http(s) URL without a fragment")
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil, errors.New("at least one redirect_uri is required")
	}
	return out, nil
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
