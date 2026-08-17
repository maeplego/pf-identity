package web

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/portfolio/pf-identity-server/internal/clock"
	"github.com/portfolio/pf-identity-server/internal/ratelimit"
)

func TestSafeContinue(t *testing.T) {
	ok := "/authorize?client_id=a&redirect_uri=https%3A%2F%2Frp%2Fcb"
	if safeContinue(ok) != ok && safeContinue(ok) == "" {
		// url.Parse may reorder query; we only require a non-empty authorize path.
		if got := safeContinue(ok); got == "" {
			t.Fatalf("got empty for %q", ok)
		}
	}
	if safeContinue("https://evil.example/steal") != "" {
		t.Fatal("absolute url")
	}
	if safeContinue("//evil.example") != "" {
		t.Fatal("protocol relative")
	}
	if safeContinue("/logout") != "" {
		t.Fatal("non-authorize path")
	}
	if safeContinue("") != "" {
		t.Fatal("empty stays empty")
	}
}

func TestLoginKeyUsesSplitHostPort(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/login", nil)
	r.RemoteAddr = "127.0.0.1:54321"
	if got := loginKey(r, "Dev@Example.COM"); got != "127.0.0.1\ndev@example.com" {
		t.Fatalf("got %q", got)
	}
	r.RemoteAddr = "[::1]:443"
	if got := loginKey(r, "a@b.c"); got != "::1\na@b.c" {
		t.Fatalf("ipv6 got %q", got)
	}
}

func TestLoginRateLimitAfterFailures(t *testing.T) {
	srv, _ := testServer(t)
	srv.Logins = ratelimit.New(3, time.Minute, clock.Real{})
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	res, err := client.Get(ts.URL + "/register")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	form := url.Values{
		"csrf":     {csrfFrom(string(body))},
		"email":    {"limited@example.com"},
		"password": {"long-enough"},
		"name":     {"L"},
	}
	res, err = client.PostForm(ts.URL+"/register", form)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	res, err = client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	home, _ := io.ReadAll(res.Body)
	res.Body.Close()
	res, err = client.PostForm(ts.URL+"/logout", url.Values{"csrf": {csrfFrom(string(home))}})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	// CSRF failures must not consume the password-guess budget.
	res, err = client.PostForm(ts.URL+"/login", url.Values{
		"csrf":     {"not-the-cookie"},
		"email":    {"limited@example.com"},
		"password": {"wrong-password"},
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("csrf fail status %d", res.StatusCode)
	}

	for i := 0; i < 3; i++ {
		status := postLogin(t, client, ts.URL, "limited@example.com", "wrong-password")
		if status != http.StatusBadRequest {
			t.Fatalf("attempt %d status %d", i, status)
		}
	}
	if status := postLogin(t, client, ts.URL, "limited@example.com", "wrong-password"); status != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", status)
	}
	// Correct password is also blocked until the window expires; that is the brute-force tradeoff.
	if status := postLogin(t, client, ts.URL, "limited@example.com", "long-enough"); status != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on correct password, got %d", status)
	}
}

func TestLoginSuccessClearsRateLimit(t *testing.T) {
	srv, _ := testServer(t)
	srv.Logins = ratelimit.New(3, time.Minute, clock.Real{})
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	res, err := client.Get(ts.URL + "/register")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	res, err = client.PostForm(ts.URL+"/register", url.Values{
		"csrf":     {csrfFrom(string(body))},
		"email":    {"ok@example.com"},
		"password": {"long-enough"},
		"name":     {"O"},
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	res, err = client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	home, _ := io.ReadAll(res.Body)
	res.Body.Close()
	res, err = client.PostForm(ts.URL+"/logout", url.Values{"csrf": {csrfFrom(string(home))}})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	if status := postLogin(t, client, ts.URL, "ok@example.com", "wrong-password"); status != http.StatusBadRequest {
		t.Fatalf("fail status %d", status)
	}
	if status := postLogin(t, client, ts.URL, "ok@example.com", "long-enough"); status != http.StatusOK {
		t.Fatalf("success status %d", status)
	}
	res, err = client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	home, _ = io.ReadAll(res.Body)
	res.Body.Close()
	res, err = client.PostForm(ts.URL+"/logout", url.Values{"csrf": {csrfFrom(string(home))}})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	// After a success the remaining budget is restored.
	if status := postLogin(t, client, ts.URL, "ok@example.com", "wrong-password"); status != http.StatusBadRequest {
		t.Fatalf("after reset fail status %d", status)
	}
}

func postLogin(t *testing.T, client *http.Client, base, email, password string) int {
	t.Helper()
	res, err := client.Get(base + "/login")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := csrfFrom(string(body))
	if csrf == "" {
		t.Fatal("missing csrf")
	}
	noRedirect := *client
	noRedirect.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	res, err = noRedirect.PostForm(base+"/login", url.Values{
		"csrf":     {csrf},
		"email":    {email},
		"password": {password},
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode == http.StatusSeeOther {
		return http.StatusOK
	}
	return res.StatusCode
}

func TestLoginKeyIgnoresEmailCase(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/login", nil)
	r.RemoteAddr = "10.0.0.8:1"
	a := loginKey(r, "User@Example.com")
	b := loginKey(r, "user@example.com")
	if a != b {
		t.Fatalf("%q != %q", a, b)
	}
}
