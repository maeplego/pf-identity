package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/id"
)

func TestAdminRequiresToken(t *testing.T) {
	srv, _ := testServer(t)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	res, err := http.Get(ts.URL + "/admin/api/clients")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestAdminClientCRUDAndSecretOnce(t *testing.T) {
	srv, _ := testServer(t)
	srv.Cfg.AdminToken = "admin-test-token"
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	createBody := `{
		"id":"blog-cms",
		"name":"Blog",
		"type":"confidential",
		"redirect_uris":["http://localhost:3000/callback"]
	}`
	res := adminJSON(t, ts.URL, "admin-test-token", http.MethodPost, "/admin/api/clients", createBody)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("create %d %s", res.StatusCode, b)
	}
	var created struct {
		Client       adminClientView `json:"client"`
		ClientSecret string          `json:"client_secret"`
	}
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ClientSecret == "" || created.Client.HasSecret != true {
		t.Fatalf("%+v", created)
	}

	got := adminJSON(t, ts.URL, "admin-test-token", http.MethodGet, "/admin/api/clients/blog-cms", "")
	raw, _ := io.ReadAll(got.Body)
	got.Body.Close()
	if got.StatusCode != http.StatusOK {
		t.Fatalf("get %d %s", got.StatusCode, raw)
	}
	if strings.Contains(string(raw), created.ClientSecret) || strings.Contains(string(raw), "secret_hash") {
		t.Fatalf("secret leaked: %s", raw)
	}

	patch := adminJSON(t, ts.URL, "admin-test-token", http.MethodPatch, "/admin/api/clients/blog-cms", `{
		"name":"Blog CMS",
		"redirect_uris":["http://localhost:3000/callback","http://localhost:3000/cb2"],
		"post_logout_redirect_uris":["http://localhost:3000/logged-out"]
	}`)
	patch.Body.Close()
	if patch.StatusCode != http.StatusOK {
		t.Fatalf("patch %d", patch.StatusCode)
	}
	gotAfter := adminJSON(t, ts.URL, "admin-test-token", http.MethodGet, "/admin/api/clients/blog-cms", "")
	afterRaw, _ := io.ReadAll(gotAfter.Body)
	gotAfter.Body.Close()
	if !strings.Contains(string(afterRaw), "logged-out") {
		t.Fatalf("post_logout missing: %s", afterRaw)
	}

	frag := adminJSON(t, ts.URL, "admin-test-token", http.MethodPost, "/admin/api/clients", `{
		"id":"frag",
		"name":"Frag",
		"type":"public",
		"redirect_uris":["http://localhost:3000/callback#x"]
	}`)
	frag.Body.Close()
	if frag.StatusCode != http.StatusBadRequest {
		t.Fatalf("fragment should be rejected, got %d", frag.StatusCode)
	}

	wrong := adminJSON(t, ts.URL, "wrong-token", http.MethodGet, "/admin/api/clients", "")
	wrong.Body.Close()
	if wrong.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong token %d", wrong.StatusCode)
	}
}

func TestAdminDisableUserAndAuditLoginFail(t *testing.T) {
	srv, store := testServer(t)
	srv.Cfg.AdminToken = "admin-test-token"
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
		"email":    {"victim@example.com"},
		"password": {"long-enough"},
		"name":     {"V"},
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	usersRes := adminJSON(t, ts.URL, "admin-test-token", http.MethodGet, "/admin/api/users", "")
	raw, _ := io.ReadAll(usersRes.Body)
	usersRes.Body.Close()
	if usersRes.StatusCode != http.StatusOK {
		t.Fatalf("users %d %s", usersRes.StatusCode, raw)
	}
	if strings.Contains(string(raw), "password") || strings.Contains(string(raw), "PasswordHash") {
		t.Fatalf("hash leaked: %s", raw)
	}
	var users []adminUserView
	if err := json.Unmarshal(raw, &users); err != nil || len(users) != 1 {
		t.Fatalf("%v %s", err, raw)
	}

	dis := adminJSON(t, ts.URL, "admin-test-token", http.MethodPost, "/admin/api/users/"+users[0].ID+"/disabled", `{"disabled":true}`)
	dis.Body.Close()
	if dis.StatusCode != http.StatusOK {
		t.Fatalf("disable %d", dis.StatusCode)
	}
	u, err := store.GetByID(context.Background(), users[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !u.Disabled {
		t.Fatal("expected disabled")
	}

	logoutHome, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	home, _ := io.ReadAll(logoutHome.Body)
	logoutHome.Body.Close()
	_, _ = client.PostForm(ts.URL+"/logout", url.Values{"csrf": {csrfFrom(string(home))}})

	if status := postLogin(t, client, ts.URL, "nobody@example.com", "wrong-password"); status != http.StatusBadRequest {
		t.Fatalf("fail login %d", status)
	}
	aud := adminJSON(t, ts.URL, "admin-test-token", http.MethodGet, "/admin/api/audits", "")
	audBody, _ := io.ReadAll(aud.Body)
	aud.Body.Close()
	if aud.StatusCode != http.StatusOK || !bytes.Contains(audBody, []byte(`"login_fail"`)) {
		t.Fatalf("audits %d %s", aud.StatusCode, audBody)
	}
}

func TestAdminAuditPaging(t *testing.T) {
	srv, store := testServer(t)
	srv.Cfg.AdminToken = "admin-test-token"
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	now := time.Date(2026, 8, 17, 7, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if err := store.AppendAudit(context.Background(), domain.AuditEvent{
			ID:   id.New(),
			Type: domain.AuditLoginFail,
			At:   now.Add(time.Duration(i) * time.Second),
			Note: strconv.Itoa(i),
		}); err != nil {
			t.Fatal(err)
		}
	}
	p1 := adminJSON(t, ts.URL, "admin-test-token", http.MethodGet, "/admin/api/audits?limit=2", "")
	raw, _ := io.ReadAll(p1.Body)
	p1.Body.Close()
	if p1.StatusCode != http.StatusOK {
		t.Fatalf("page1 %d %s", p1.StatusCode, raw)
	}
	var page struct {
		Items []struct {
			Note string `json:"note"`
			ID   string `json:"id"`
		} `json:"items"`
		Next string `json:"next"`
	}
	if err := json.Unmarshal(raw, &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].Note != "2" || page.Next == "" {
		t.Fatalf("%s", raw)
	}
	p2 := adminJSON(t, ts.URL, "admin-test-token", http.MethodGet, "/admin/api/audits?limit=2&after="+page.Next, "")
	raw2, _ := io.ReadAll(p2.Body)
	p2.Body.Close()
	if err := json.Unmarshal(raw2, &page); err != nil {
		t.Fatal(err)
	}
	if p2.StatusCode != http.StatusOK || len(page.Items) != 1 || page.Items[0].Note != "0" || page.Next != "" {
		t.Fatalf("page2 %d %s", p2.StatusCode, raw2)
	}
	bad := adminJSON(t, ts.URL, "admin-test-token", http.MethodGet, "/admin/api/audits?after=missing", "")
	bad.Body.Close()
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad cursor %d", bad.StatusCode)
	}
}

func TestAdminOrganizationMembers(t *testing.T) {
	srv, store := testServer(t)
	srv.Cfg.AdminToken = "admin-test-token"
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	clk := srv.Clock

	owner := domain.User{ID: id.New(), Email: "org-owner@example.com", Name: "Owner", PasswordHash: "x", CreatedAt: clk.Now()}
	member := domain.User{ID: id.New(), Email: "org-member@example.com", Name: "Member", PasswordHash: "x", CreatedAt: clk.Now()}
	if err := store.Create(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(context.Background(), member); err != nil {
		t.Fatal(err)
	}

	res := adminJSON(t, ts.URL, "admin-test-token", http.MethodPost, "/admin/api/organizations",
		`{"name":"Acme","owner_email":"org-owner@example.com"}`)
	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		t.Fatalf("create org %d %s", res.StatusCode, b)
	}
	var org struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&org); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	res = adminJSON(t, ts.URL, "admin-test-token", http.MethodPost, "/admin/api/organizations/"+org.ID+"/members",
		`{"email":"org-member@example.com","role":"member"}`)
	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		t.Fatalf("add member %d %s", res.StatusCode, b)
	}
	res.Body.Close()

	res = adminJSON(t, ts.URL, "admin-test-token", http.MethodGet, "/admin/api/organizations/"+org.ID+"/members", "")
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list members %d %s", res.StatusCode, raw)
	}
	var members []struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
		Email  string `json:"email"`
	}
	if err := json.Unmarshal(raw, &members); err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("want 2 members got %d", len(members))
	}

	res = adminJSON(t, ts.URL, "admin-test-token", http.MethodPatch,
		"/admin/api/organizations/"+org.ID+"/members/"+member.ID, `{"role":"owner"}`)
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		t.Fatalf("promote %d %s", res.StatusCode, b)
	}
	res.Body.Close()

	res = adminJSON(t, ts.URL, "admin-test-token", http.MethodDelete,
		"/admin/api/organizations/"+org.ID+"/members/"+owner.ID, "")
	if res.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		t.Fatalf("remove former owner %d %s", res.StatusCode, b)
	}
	res.Body.Close()

	res = adminJSON(t, ts.URL, "admin-test-token", http.MethodDelete,
		"/admin/api/organizations/"+org.ID+"/members/"+member.ID, "")
	if res.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		t.Fatalf("last owner remove want 403 got %d %s", res.StatusCode, b)
	}
	res.Body.Close()
}

func adminJSON(t *testing.T, base, token, method, path, body string) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, base+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}
