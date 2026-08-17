package web

import (
	"html/template"
	"net/http"
	"net/url"

	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/oauth"
	"github.com/portfolio/pf-identity-server/internal/oidc"
	"github.com/portfolio/pf-identity-server/internal/session"
)

type endSessionReq struct {
	IDTokenHint           string
	ClientID              string
	PostLogoutRedirectURI string
	State                 string
	Confirm               bool
}

func (s *Server) handleEndSession(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
	}
	req := parseEndSession(r)
	if req.PostLogoutRedirectURI != "" {
		if _, err := oauth.ParseRedirectURI(req.PostLogoutRedirectURI); err != nil {
			http.Error(w, "post_logout_redirect_uri must be an absolute http(s) URL without a fragment", http.StatusBadRequest)
			return
		}
	}

	hint, hintOK := s.parseHint(req.IDTokenHint)
	client, err := s.logoutClient(r, req, hint, hintOK)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.PostLogoutRedirectURI != "" {
		if client.ID == "" || !oauth.RedirectURIExact(req.PostLogoutRedirectURI, client.PostLogoutRedirectURIs) {
			http.Error(w, "post_logout_redirect_uri is not registered", http.StatusBadRequest)
			return
		}
	}

	user, loggedIn := s.currentUser(r)
	hintMatches := loggedIn && hintOK && hint.Issuer == s.Cfg.Issuer && hint.Subject == user.ID
	if req.ClientID != "" && hintOK && !containsAudience(hint.Audience, req.ClientID) {
		http.Error(w, "client_id does not match id_token_hint audience", http.StatusBadRequest)
		return
	}

	confirmed := req.Confirm && r.Method == http.MethodPost && s.checkCSRF(r)
	// GET from an RP sends SameSite=Lax cookies. A matching id_token_hint is
	// the spec's stand-in for CSRF so we do not also require the IdP form token.
	if loggedIn && !hintMatches && !confirmed {
		s.renderEndSessionConfirm(w, req)
		return
	}

	s.finishLogout(w, r, req, "")
}

func (s *Server) finishLogout(w http.ResponseWriter, r *http.Request, req endSessionReq, defaultNext string) {
	sid := s.currentSID(r)
	iframes := s.frontChannelIFrames(r, sid)
	s.endBrowserSession(w, r)
	next := defaultNext
	if req.PostLogoutRedirectURI != "" {
		u, err := url.Parse(req.PostLogoutRedirectURI)
		if err != nil {
			http.Error(w, "post_logout_redirect_uri is not registered", http.StatusBadRequest)
			return
		}
		q := u.Query()
		if req.State != "" {
			q.Set("state", req.State)
		}
		u.RawQuery = q.Encode()
		next = u.String()
	}
	if len(iframes) > 0 {
		s.render(w, "front_channel", pageData{
			Title:       "ログアウト",
			IFrames:     iframes,
			ContinueURL: template.URL(next),
		}, http.StatusOK)
		return
	}
	if next != "" {
		http.Redirect(w, r, next, http.StatusFound)
		return
	}
	s.render(w, "logged_out", pageData{Title: "ログアウト"}, http.StatusOK)
}

func (s *Server) renderEndSessionConfirm(w http.ResponseWriter, req endSessionReq) {
	tok, err := s.issueCSRF(w)
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	s.render(w, "end_session", pageData{
		Title:                 "ログアウトの確認",
		CSRF:                  tok,
		IDTokenHint:           req.IDTokenHint,
		PostLogoutRedirectURI: req.PostLogoutRedirectURI,
		State:                 req.State,
		ClientID:              req.ClientID,
	}, http.StatusOK)
}

func (s *Server) endBrowserSession(w http.ResponseWriter, r *http.Request) string {
	sid := ""
	c, _ := r.Cookie(session.CookieName())
	if c != nil {
		if sess, err := s.Sessions.LookupSession(r.Context(), c.Value); err == nil {
			sid = sess.SID
		}
		_ = s.Sessions.End(r.Context(), c.Value)
	}
	clearCookie(w, session.CookieName(), s.Cfg.CookieSecure)
	return sid
}

func (s *Server) frontChannelIFrames(r *http.Request, sid string) []string {
	if sid == "" {
		return nil
	}
	ids, err := s.Repos.ListSessionClients(r.Context(), sid)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		c, err := s.Repos.GetClient(r.Context(), id)
		if err != nil || c.FrontChannelLogoutURI == "" {
			continue
		}
		u, err := url.Parse(c.FrontChannelLogoutURI)
		if err != nil {
			continue
		}
		q := u.Query()
		q.Set("iss", s.Cfg.Issuer)
		q.Set("sid", sid)
		u.RawQuery = q.Encode()
		src := u.String()
		if _, ok := seen[src]; ok {
			continue
		}
		seen[src] = struct{}{}
		out = append(out, src)
	}
	return out
}

func parseEndSession(r *http.Request) endSessionReq {
	get := r.URL.Query().Get
	if r.Method == http.MethodPost {
		get = r.FormValue
	}
	return endSessionReq{
		IDTokenHint:           get("id_token_hint"),
		ClientID:              get("client_id"),
		PostLogoutRedirectURI: get("post_logout_redirect_uri"),
		State:                 get("state"),
		Confirm:               get("confirm") == "yes",
	}
}

func (s *Server) parseHint(compact string) (oidc.IDTokenHint, bool) {
	if compact == "" {
		return oidc.IDTokenHint{}, false
	}
	hint, err := s.Signer.ParseIDTokenHint(compact)
	if err != nil || hint.Issuer != s.Cfg.Issuer || hint.Subject == "" {
		return oidc.IDTokenHint{}, false
	}
	return hint, true
}

func (s *Server) logoutClient(r *http.Request, req endSessionReq, hint oidc.IDTokenHint, hintOK bool) (domain.Client, error) {
	id := req.ClientID
	if id == "" && hintOK && len(hint.Audience) == 1 {
		id = hint.Audience[0]
	}
	if id != "" {
		c, err := s.Repos.GetClient(r.Context(), id)
		if err != nil {
			return domain.Client{}, errBadLogoutClient
		}
		return c, nil
	}
	if req.PostLogoutRedirectURI == "" || !hintOK {
		return domain.Client{}, nil
	}
	for _, aud := range hint.Audience {
		c, err := s.Repos.GetClient(r.Context(), aud)
		if err != nil {
			continue
		}
		if oauth.RedirectURIExact(req.PostLogoutRedirectURI, c.PostLogoutRedirectURIs) {
			return c, nil
		}
	}
	return domain.Client{}, errBadLogoutRedirect
}

func containsAudience(aud []string, clientID string) bool {
	for _, a := range aud {
		if a == clientID {
			return true
		}
	}
	return false
}

type logoutParamError string

func (e logoutParamError) Error() string { return string(e) }

const (
	errBadLogoutClient   logoutParamError = "unknown client_id"
	errBadLogoutRedirect logoutParamError = "post_logout_redirect_uri is not registered"
)
