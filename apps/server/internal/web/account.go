package web

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/portfolio/pf-identity-server/internal/account"
	"github.com/portfolio/pf-identity-server/internal/domain"
)

func (s *Server) handleRegisterForm(w http.ResponseWriter, r *http.Request) {
	tok, err := s.issueCSRF(w)
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	s.render(w, "register", pageData{Title: "登録", CSRF: tok, Continue: safeContinue(r.URL.Query().Get("continue"))}, http.StatusOK)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || !s.checkCSRF(r) {
		s.failForm(w, r, "register", "登録", "フォームをやり直してください。", http.StatusBadRequest)
		return
	}
	cont := safeContinue(r.FormValue("continue"))
	u, err := s.Accounts.Register(r.Context(), account.RegisterInput{
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
		Name:     r.FormValue("name"),
	})
	if err != nil {
		msg := "登録できませんでした。"
		if err == domain.ErrConflict {
			msg = "このメールは既に使われています。"
		}
		s.failForm(w, r, "register", "登録", msg, http.StatusBadRequest)
		return
	}
	s.startSessionAndRedirect(w, r, u.ID, cont)
}

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	tok, err := s.issueCSRF(w)
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	s.render(w, "login", pageData{Title: "ログイン", CSRF: tok, Continue: safeContinue(r.URL.Query().Get("continue"))}, http.StatusOK)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || !s.checkCSRF(r) {
		s.failForm(w, r, "login", "ログイン", "フォームをやり直してください。", http.StatusBadRequest)
		return
	}
	cont := safeContinue(r.FormValue("continue"))
	email := r.FormValue("email")
	limitKey := loginKey(r, email)
	if s.Logins != nil && s.Logins.Limited(limitKey) {
		s.failForm(w, r, "login", "ログイン", "試行が多すぎます。しばらく待ってください。", http.StatusTooManyRequests)
		return
	}
	u, err := s.Accounts.Authenticate(r.Context(), email, r.FormValue("password"))
	if err != nil {
		if s.Logins != nil {
			s.Logins.Failure(limitKey)
		}
		s.audit(r, domain.AuditLoginFail, "", "", strings.ToLower(strings.TrimSpace(email)))
		s.failForm(w, r, "login", "ログイン", "メールまたはパスワードが違います。", http.StatusBadRequest)
		return
	}
	if s.Logins != nil {
		s.Logins.Success(limitKey)
	}
	s.startSessionAndRedirect(w, r, u.ID, cont)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || !s.checkCSRF(r) {
		http.Error(w, "csrf", http.StatusBadRequest)
		return
	}
	s.finishLogout(w, r, endSessionReq{}, "/login")
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	u, ok := s.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	tok, err := s.issueCSRF(w)
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	s.render(w, "home", pageData{Title: "ホーム", CSRF: tok, Name: u.Name, Email: u.Email}, http.StatusOK)
}

func (s *Server) startSessionAndRedirect(w http.ResponseWriter, r *http.Request, userID, cont string) {
	plain, exp, err := s.Sessions.Start(r.Context(), userID)
	if err != nil {
		http.Error(w, "session", http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, plain, exp, s.Cfg.CookieSecure)
	if cont == "" {
		cont = "/"
	}
	http.Redirect(w, r, cont, http.StatusSeeOther)
}

func (s *Server) failForm(w http.ResponseWriter, r *http.Request, view, title, msg string, status int) {
	tok, err := s.issueCSRF(w)
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	s.render(w, view, pageData{Title: title, CSRF: tok, Error: msg, Continue: safeContinue(r.FormValue("continue"))}, status)
}

// safeContinue only allows a relative /authorize URL so login cannot bounce to an open redirect.
func safeContinue(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Host != "" || u.Scheme != "" {
		return ""
	}
	if u.Path != "/authorize" {
		return ""
	}
	if strings.Contains(raw, "\\") {
		return ""
	}
	return u.String()
}

func loginKey(r *http.Request, email string) string {
	return requestIP(r) + "\n" + strings.ToLower(strings.TrimSpace(email))
}
