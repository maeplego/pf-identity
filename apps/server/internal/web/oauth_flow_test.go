package web

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/portfolio/pf-identity-server/internal/account"
	"github.com/portfolio/pf-identity-server/internal/clock"
	"github.com/portfolio/pf-identity-server/internal/config"
	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/oidc"
	"github.com/portfolio/pf-identity-server/internal/session"
	"github.com/portfolio/pf-identity-server/internal/store/memory"
)

func testServer(t *testing.T) (*Server, *memory.Store) {
	t.Helper()
	store := memory.NewStore()
	clk := clock.Real{}
	cfg := config.Config{
		HTTPAddr:     ":0",
		Issuer:       "http://127.0.0.1",
		CookieSecure: false,
		Store:        config.StoreMemory,
		SessionTTL:   time.Hour,
		CodeTTL:      time.Minute,
		AccessTTL:    time.Hour,
		RefreshTTL:   24 * time.Hour,
	}
	signer, err := oidc.GenerateRSA()
	if err != nil {
		t.Fatal(err)
	}
	acc := &account.Service{Users: store, Clock: clk}
	sess := &session.Service{Sessions: store, Clock: clk, TTL: cfg.SessionTTL}
	srv, err := NewServer(cfg, acc, sess, store, signer, clk)
	if err != nil {
		t.Fatal(err)
	}
	return srv, store
}

func csrfFrom(body string) string {
	const marker = `name="csrf" value="`
	i := strings.Index(body, marker)
	if i < 0 {
		return ""
	}
	rest := body[i+len(marker):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

func TestRegisterLoginLogout(t *testing.T) {
	srv, _ := testServer(t)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	res, err := client.Get(ts.URL + "/register")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := csrfFrom(string(b))
	if csrf == "" {
		t.Fatal("missing csrf")
	}
	form := url.Values{
		"csrf":     {csrf},
		"email":    {"dev@example.com"},
		"password": {"long-enough"},
		"name":     {"Dev"},
	}
	res, err = client.PostForm(ts.URL+"/register", form)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.Request.URL.Path != "/" {
		t.Fatalf("after register path %s", res.Request.URL.Path)
	}

	res, err = client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	home, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf = csrfFrom(string(home))
	res, err = client.PostForm(ts.URL+"/logout", url.Values{"csrf": {csrf}})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.Request.URL.Path != "/login" {
		t.Fatalf("after logout %s", res.Request.URL.Path)
	}
}

// TestAuthorizationCodePKCEAndReplay is the DESIGN demo lock for code replay.
// Browser e2e cannot observe /token as reliably as this HTTP client.
func TestAuthorizationCodePKCEAndReplay(t *testing.T) {
	srv, store := testServer(t)
	redirect := "http://127.0.0.1/cb"
	_ = store.CreateClient(context.Background(), domain.Client{
		ID:           "rp-public",
		Name:         "Sample RP",
		Type:         domain.ClientPublic,
		RedirectURIs: []string{redirect},
	})
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if strings.HasPrefix(req.URL.String(), redirect) {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	res, err := client.Get(ts.URL + "/register")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	form := url.Values{"csrf": {csrfFrom(string(body))}, "email": {"u@example.com"}, "password": {"long-enough"}, "name": {"U"}}
	res, err = client.PostForm(ts.URL+"/register", form)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	verifier := strings.Repeat("a", 43)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	authURL := ts.URL + "/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {"rp-public"},
		"redirect_uri":          {redirect},
		"scope":                 {"openid profile email offline_access"},
		"state":                 {"state-1"},
		"nonce":                 {"nonce-1"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}.Encode()
	res, err = client.Get(authURL)
	if err != nil {
		t.Fatal(err)
	}
	consentBody, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(consentBody)
	reqID := between(html, `name="request_id" value="`, `"`)
	csrf := csrfFrom(html)
	if reqID == "" || csrf == "" {
		t.Fatalf("consent page missing fields: %s", html)
	}
	res, err = client.PostForm(ts.URL+"/consent", url.Values{
		"csrf":       {csrf},
		"request_id": {reqID},
		"decision":   {"allow"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("status %d", res.StatusCode)
	}
	loc, err := res.Location()
	if err != nil {
		t.Fatal(err)
	}
	code := loc.Query().Get("code")
	if code == "" || loc.Query().Get("state") != "state-1" {
		t.Fatalf("redirect %s", loc)
	}

	tokRes, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"rp-public"},
		"code":          {code},
		"redirect_uri":  {redirect},
		"code_verifier": {verifier},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(tokRes.Body)
	tokRes.Body.Close()
	if tokRes.StatusCode != http.StatusOK {
		t.Fatalf("token %d %s", tokRes.StatusCode, raw)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["id_token"] == "" || payload["access_token"] == "" || payload["refresh_token"] == "" {
		t.Fatalf("payload %+v", payload)
	}

	replay, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"rp-public"},
		"code":          {code},
		"redirect_uri":  {redirect},
		"code_verifier": {verifier},
	})
	if err != nil {
		t.Fatal(err)
	}
	replay.Body.Close()
	if replay.StatusCode == http.StatusOK {
		t.Fatal("replayed code must fail")
	}

	ui, err := http.NewRequest(http.MethodGet, ts.URL+"/userinfo", nil)
	if err != nil {
		t.Fatal(err)
	}
	ui.Header.Set("Authorization", "Bearer "+payload["access_token"].(string))
	uiRes, err := http.DefaultClient.Do(ui)
	if err != nil {
		t.Fatal(err)
	}
	uiBody, _ := io.ReadAll(uiRes.Body)
	uiRes.Body.Close()
	if uiRes.StatusCode != http.StatusOK || !strings.Contains(string(uiBody), "u@example.com") {
		t.Fatalf("userinfo %d %s", uiRes.StatusCode, uiBody)
	}
}

func TestRefreshRotationReuseRevokesFamily(t *testing.T) {
	srv, store := testServer(t)
	redirect := "http://127.0.0.1/cb"
	_ = store.CreateClient(context.Background(), domain.Client{
		ID:           "rp-refresh",
		Name:         "RP",
		Type:         domain.ClientPublic,
		RedirectURIs: []string{redirect},
	})
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	tokens := mintOfflineTokens(t, ts, "rp-refresh", redirect, "r@example.com")
	oldRefresh := tokens["refresh_token"].(string)

	rotated, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"rp-refresh"},
		"refresh_token": {oldRefresh},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(rotated.Body)
	rotated.Body.Close()
	if rotated.StatusCode != http.StatusOK {
		t.Fatalf("rotate %d %s", rotated.StatusCode, raw)
	}
	var next map[string]any
	if err := json.Unmarshal(raw, &next); err != nil {
		t.Fatal(err)
	}
	newRefresh := next["refresh_token"].(string)
	if newRefresh == "" || newRefresh == oldRefresh {
		t.Fatalf("expected a new refresh token: %v", next)
	}

	reuse, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"rp-refresh"},
		"refresh_token": {oldRefresh},
	})
	if err != nil {
		t.Fatal(err)
	}
	reuse.Body.Close()
	if reuse.StatusCode == http.StatusOK {
		t.Fatal("reused refresh must fail")
	}

	child, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"rp-refresh"},
		"refresh_token": {newRefresh},
	})
	if err != nil {
		t.Fatal(err)
	}
	child.Body.Close()
	if child.StatusCode == http.StatusOK {
		t.Fatal("family must be revoked after reuse")
	}
}

func mintOfflineTokens(t *testing.T, ts *httptest.Server, clientID, redirect, email string) map[string]any {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if strings.HasPrefix(req.URL.String(), redirect) {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	res, err := client.Get(ts.URL + "/register")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	res, err = client.PostForm(ts.URL+"/register", url.Values{
		"csrf":     {csrfFrom(string(body))},
		"email":    {email},
		"password": {"long-enough"},
		"name":     {"R"},
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	verifier := strings.Repeat("b", 43)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	res, err = client.Get(ts.URL + "/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirect},
		"scope":                 {"openid offline_access"},
		"state":                 {"st"},
		"nonce":                 {"n"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}.Encode())
	if err != nil {
		t.Fatal(err)
	}
	htmlBytes, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(htmlBytes)
	res, err = client.PostForm(ts.URL+"/consent", url.Values{
		"csrf":       {csrfFrom(html)},
		"request_id": {between(html, `name="request_id" value="`, `"`)},
		"decision":   {"allow"},
	})
	if err != nil {
		t.Fatal(err)
	}
	loc, err := res.Location()
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	tokRes, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {loc.Query().Get("code")},
		"redirect_uri":  {redirect},
		"code_verifier": {verifier},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(tokRes.Body)
	tokRes.Body.Close()
	if tokRes.StatusCode != http.StatusOK {
		t.Fatalf("token %d %s", tokRes.StatusCode, raw)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

// TestRejectsAlteredRedirectURI is the DESIGN demo lock for exact redirect_uri match.
func TestRejectsAlteredRedirectURI(t *testing.T) {
	srv, store := testServer(t)
	_ = store.CreateClient(context.Background(), domain.Client{
		ID:           "rp",
		Name:         "RP",
		Type:         domain.ClientPublic,
		RedirectURIs: []string{"http://127.0.0.1/cb"},
	})
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	res, err := http.Get(ts.URL + "/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {"rp"},
		"redirect_uri":          {"http://127.0.0.1/cb?extra=1"},
		"scope":                 {"openid"},
		"code_challenge":        {"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		"code_challenge_method": {"S256"},
	}.Encode())
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestDiscovery(t *testing.T) {
	srv, _ := testServer(t)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	res, err := http.Get(ts.URL + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var doc map[string]any
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc["token_endpoint"] == "" || doc["end_session_endpoint"] == "" {
		t.Fatalf("%v", doc)
	}
}

func between(s, a, b string) string {
	i := strings.Index(s, a)
	if i < 0 {
		return ""
	}
	s = s[i+len(a):]
	j := strings.Index(s, b)
	if j < 0 {
		return ""
	}
	return s[:j]
}
