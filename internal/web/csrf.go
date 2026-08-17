package web

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/portfolio/pf-identity-server/internal/random"
)

const csrfCookie = "idp_csrf"

func (s *Server) issueCSRF(w http.ResponseWriter) (string, error) {
	tok, err := random.Token()
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookie,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.Cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  s.now().Add(2 * time.Hour),
	})
	return tok, nil
}

func (s *Server) checkCSRF(r *http.Request) bool {
	c, err := r.Cookie(csrfCookie)
	if err != nil {
		return false
	}
	form := r.FormValue("csrf")
	if c.Value == "" || form == "" {
		return false
	}
	if len(c.Value) != len(form) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(form)) == 1
}

func (s *Server) now() time.Time { return s.Clock.Now() }

func setSessionCookie(w http.ResponseWriter, plain string, exp time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "idp_session",
		Value:    plain,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  exp,
	})
}

func clearCookie(w http.ResponseWriter, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
