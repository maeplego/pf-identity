package web

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/portfolio/pf-identity-server/internal/account"
	"github.com/portfolio/pf-identity-server/internal/clock"
	"github.com/portfolio/pf-identity-server/internal/config"
	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/oidc"
	"github.com/portfolio/pf-identity-server/internal/ratelimit"
	"github.com/portfolio/pf-identity-server/internal/session"
)

//go:embed templates/*.html
var templateFS embed.FS

type pageData struct {
	Title                 string
	Error                 string
	CSRF                  string
	Continue              string
	Name                  string
	Email                 string
	ClientName            string
	Scopes                []string
	ScopeViews            []ScopeView
	RequestID             string
	IDTokenHint           string
	PostLogoutRedirectURI string
	State                 string
	ClientID              string
	IFrames               []string
	ContinueURL           template.URL
}

type ScopeView struct {
	ID    string
	Label string
}

func scopeViews(scopes []string) []ScopeView {
	out := make([]ScopeView, 0, len(scopes))
	for _, id := range scopes {
		label := id
		switch id {
		case "openid":
			label = "ログイン識別（OpenID）"
		case "profile":
			label = "表示名などのプロフィール"
		case "email":
			label = "メールアドレス"
		case "offline_access":
			label = "再ログインなしで継続（リフレッシュ）"
		case "org":
			label = "所属組織（テナント）"
		}
		out = append(out, ScopeView{ID: id, Label: label})
	}
	return out
}

type pendingAuth struct {
	ClientID    string
	RedirectURI string
	State       string
	Nonce       string
	Scopes      []string
	Challenge   string
	ExpiresAt   time.Time
}

// Server is the HTTP surface of the authorization server.
type Server struct {
	Cfg      config.Config
	Accounts *account.Service
	Sessions *session.Service
	Repos    domain.Repos
	Signer   *oidc.Signer
	Clock    clock.Clock
	Logins     *ratelimit.Limiter
	LogoutHTTP *http.Client

	mux *http.ServeMux
	tpl map[string]*template.Template

	mu      sync.Mutex
	pending map[string]pendingAuth
}

// NewServer wires routes.
func NewServer(cfg config.Config, acc *account.Service, sess *session.Service, repos domain.Repos, signer *oidc.Signer, clk clock.Clock) (*Server, error) {
	s := &Server{
		Cfg:      cfg,
		Accounts: acc,
		Sessions: sess,
		Repos:    repos,
		Signer:   signer,
		Clock:    clk,
		Logins:     ratelimit.New(5, time.Minute, clk),
		LogoutHTTP: &http.Client{Timeout: 3 * time.Second},
		pending:  map[string]pendingAuth{},
		tpl:      map[string]*template.Template{},
		mux:      http.NewServeMux(),
	}
	for _, name := range []string{"login", "register", "consent", "home", "end_session", "logged_out", "front_channel"} {
		t, err := template.ParseFS(templateFS, "templates/layout.html", "templates/"+name+".html")
		if err != nil {
			return nil, err
		}
		s.tpl[name] = t
	}
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /register", s.handleRegisterForm)
	s.mux.HandleFunc("POST /register", s.handleRegister)
	s.mux.HandleFunc("GET /login", s.handleLoginForm)
	s.mux.HandleFunc("POST /login", s.handleLogin)
	s.mux.HandleFunc("POST /logout", s.handleLogout)
	s.mux.HandleFunc("GET /end-session", s.handleEndSession)
	s.mux.HandleFunc("POST /end-session", s.handleEndSession)
	s.mux.HandleFunc("GET /{$}", s.handleHome)
	s.mux.HandleFunc("GET /authorize", s.handleAuthorize)
	s.mux.HandleFunc("POST /consent", s.handleConsent)
	s.mux.HandleFunc("POST /token", s.handleToken)
	s.mux.HandleFunc("GET /userinfo", s.handleUserInfo)
	s.mux.HandleFunc("GET /.well-known/openid-configuration", s.handleDiscovery)
	s.mux.HandleFunc("GET /jwks.json", s.handleJWKS)
	s.mux.HandleFunc("GET /.well-known/jwks.json", s.handleJWKS)
	s.mux.HandleFunc("GET /admin/api/clients", s.handleAdminListClients)
	s.mux.HandleFunc("POST /admin/api/clients", s.handleAdminCreateClient)
	s.mux.HandleFunc("GET /admin/api/clients/{id}", s.handleAdminGetClient)
	s.mux.HandleFunc("PATCH /admin/api/clients/{id}", s.handleAdminUpdateClient)
	s.mux.HandleFunc("POST /admin/api/clients/{id}/rotate-secret", s.handleAdminRotateSecret)
	s.mux.HandleFunc("GET /admin/api/users", s.handleAdminListUsers)
	s.mux.HandleFunc("POST /admin/api/users/{id}/disabled", s.handleAdminSetDisabled)
	s.mux.HandleFunc("GET /admin/api/audits", s.handleAdminListAudits)
	s.mux.HandleFunc("POST /v1/organizations", s.handleCreateOrganization)
	s.mux.HandleFunc("GET /v1/organizations", s.handleListOrganizations)
	s.mux.HandleFunc("GET /v1/organizations/{id}/members", s.handleListOrganizationMembers)
	s.mux.HandleFunc("PUT /account/active-org", s.handleSetActiveOrg)
	s.mux.HandleFunc("PUT /v1/active-org", s.handleSetActiveOrgAPI)
	return s, nil
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) render(w http.ResponseWriter, name string, data pageData, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := s.tpl[name].ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = io.WriteString(w, "ok")
}

func (s *Server) currentUser(r *http.Request) (domain.User, bool) {
	c, err := r.Cookie(session.CookieName())
	if err != nil {
		return domain.User{}, false
	}
	uid, err := s.Sessions.Lookup(r.Context(), c.Value)
	if err != nil {
		return domain.User{}, false
	}
	u, err := s.Repos.GetByID(r.Context(), uid)
	if err != nil || u.Disabled {
		return domain.User{}, false
	}
	return u, true
}

func (s *Server) currentSID(r *http.Request) string {
	c, err := r.Cookie(session.CookieName())
	if err != nil {
		return ""
	}
	sess, err := s.Sessions.LookupSession(r.Context(), c.Value)
	if err != nil {
		return ""
	}
	return sess.SID
}
