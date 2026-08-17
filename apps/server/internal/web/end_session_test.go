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

	"github.com/portfolio/pf-identity-server/internal/domain"
)

func TestRPInitiatedLogoutRedirectsAndEndsSession(t *testing.T) {
	srv, store := testServer(t)
	redirect := "http://127.0.0.1/cb"
	postLogout := "http://127.0.0.1/logged-out"
	_ = store.CreateClient(context.Background(), domain.Client{
		ID:                     "rp-logout",
		Name:                   "RP",
		Type:                   domain.ClientPublic,
		RedirectURIs:           []string{redirect},
		PostLogoutRedirectURIs: []string{postLogout},
	})
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	client, tokens := loginAndMint(t, ts, "rp-logout", redirect, "out@example.com", []string{redirect, postLogout})
	idToken, _ := tokens["id_token"].(string)
	if idToken == "" {
		t.Fatal("missing id_token")
	}

	res, err := client.Get(ts.URL + "/end-session?" + url.Values{
		"id_token_hint":            {idToken},
		"client_id":                {"rp-logout"},
		"post_logout_redirect_uri": {postLogout},
		"state":                    {"logout-state"},
	}.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d %s", res.StatusCode, body)
	}
	loc, err := res.Location()
	if err != nil {
		t.Fatal(err)
	}
	if loc.Scheme+"://"+loc.Host+loc.Path != postLogout || loc.Query().Get("state") != "logout-state" {
		t.Fatalf("redirect %s", loc)
	}

	home, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	home.Body.Close()
	if home.Request.URL.Path != "/login" {
		t.Fatalf("session survived logout, path %s", home.Request.URL.Path)
	}
}

func TestEndSessionRejectsUnregisteredPostLogout(t *testing.T) {
	srv, store := testServer(t)
	redirect := "http://127.0.0.1/cb"
	_ = store.CreateClient(context.Background(), domain.Client{
		ID:                     "rp-logout-bad",
		Name:                   "RP",
		Type:                   domain.ClientPublic,
		RedirectURIs:           []string{redirect},
		PostLogoutRedirectURIs: []string{"http://127.0.0.1/logged-out"},
	})
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	client, tokens := loginAndMint(t, ts, "rp-logout-bad", redirect, "bad@example.com", []string{redirect})
	res, err := client.Get(ts.URL + "/end-session?" + url.Values{
		"id_token_hint":            {tokens["id_token"].(string)},
		"client_id":                {"rp-logout-bad"},
		"post_logout_redirect_uri": {"http://evil.example/phish"},
		"state":                    {"x"},
	}.Encode())
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d %s", res.StatusCode, body)
	}

	home, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(home.Body)
	home.Body.Close()
	if !strings.Contains(string(raw), "ログイン済み") {
		t.Fatalf("expected session to remain: %s", raw)
	}
}

func TestEndSessionWithoutHintShowsConfirm(t *testing.T) {
	srv, store := testServer(t)
	redirect := "http://127.0.0.1/cb"
	_ = store.CreateClient(context.Background(), domain.Client{
		ID:           "rp-logout-confirm",
		Name:         "RP",
		Type:         domain.ClientPublic,
		RedirectURIs: []string{redirect},
	})
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	client, _ := loginAndMint(t, ts, "rp-logout-confirm", redirect, "c@example.com", []string{redirect})
	res, err := client.Get(ts.URL + "/end-session")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "ログアウトしますか") {
		t.Fatalf("status %d %s", res.StatusCode, body)
	}

	home, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(home.Body)
	home.Body.Close()
	if !strings.Contains(string(raw), "ログイン済み") {
		t.Fatalf("confirm must not log out yet: %s", raw)
	}
}

func TestDiscoveryAdvertisesEndSession(t *testing.T) {
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
	if doc["end_session_endpoint"] != srv.Cfg.Issuer+"/end-session" {
		t.Fatalf("%v", doc["end_session_endpoint"])
	}
	if doc["frontchannel_logout_supported"] != true || doc["frontchannel_logout_session_supported"] != true {
		t.Fatalf("%v", doc)
	}
}

func TestFrontChannelLogoutRendersIframes(t *testing.T) {
	rpA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(rpA.Close)
	rpB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(rpB.Close)

	srv, store := testServer(t)
	redirectA := "http://127.0.0.1/cb-a"
	redirectB := "http://127.0.0.1/cb-b"
	postLogout := "http://127.0.0.1/logged-out"
	_ = store.CreateClient(context.Background(), domain.Client{
		ID:                     "rp-a",
		Name:                   "A",
		Type:                   domain.ClientPublic,
		RedirectURIs:           []string{redirectA},
		PostLogoutRedirectURIs: []string{postLogout},
		FrontChannelLogoutURI:  rpA.URL + "/frontchannel-logout",
	})
	_ = store.CreateClient(context.Background(), domain.Client{
		ID:                    "rp-b",
		Name:                  "B",
		Type:                  domain.ClientPublic,
		RedirectURIs:          []string{redirectB},
		FrontChannelLogoutURI: rpB.URL + "/frontchannel-logout",
	})
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	client, tokens := loginAndMint(t, ts, "rp-a", redirectA, "fc@example.com", []string{redirectA, redirectB, postLogout})
	authorizeAndToken(t, client, ts, "rp-b", redirectB)

	res, err := client.Get(ts.URL + "/end-session?" + url.Values{
		"id_token_hint":            {tokens["id_token"].(string)},
		"client_id":                {"rp-a"},
		"post_logout_redirect_uri": {postLogout},
		"state":                    {"st"},
	}.Encode())
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d %s", res.StatusCode, html)
	}
	if !strings.Contains(html, rpA.URL+"/frontchannel-logout") || !strings.Contains(html, rpB.URL+"/frontchannel-logout") {
		t.Fatalf("missing iframes: %s", html)
	}
	if !strings.Contains(html, "iss=") || !strings.Contains(html, "sid=") {
		t.Fatalf("missing iss/sid: %s", html)
	}
	if !strings.Contains(html, postLogout) {
		t.Fatalf("missing continue: %s", html)
	}
}

func loginAndMint(t *testing.T, ts *httptest.Server, clientID, redirect, email string, stopPrefixes []string) (*http.Client, map[string]any) {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			for _, prefix := range stopPrefixes {
				if strings.HasPrefix(req.URL.String(), prefix) {
					return http.ErrUseLastResponse
				}
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
		"name":     {"L"},
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	verifier := strings.Repeat("c", 43)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	res, err = client.Get(ts.URL + "/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirect},
		"scope":                 {"openid"},
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
	return client, payload
}

func authorizeAndToken(t *testing.T, client *http.Client, ts *httptest.Server, clientID, redirect string) {
	t.Helper()
	verifier := strings.Repeat("d", 43)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	res, err := client.Get(ts.URL + "/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirect},
		"scope":                 {"openid"},
		"state":                 {"st2"},
		"nonce":                 {"n2"},
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
	tokRes.Body.Close()
	if tokRes.StatusCode != http.StatusOK {
		t.Fatalf("token %d", tokRes.StatusCode)
	}
}
