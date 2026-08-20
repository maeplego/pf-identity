package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/portfolio/pf-identity-server/internal/clock"
	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/id"
	"github.com/portfolio/pf-identity-server/internal/oauth"
	"github.com/portfolio/pf-identity-server/internal/org"
)

func TestOrganizationAPIAndOrgClaims(t *testing.T) {
	srv, repos := testServer(t)
	clk := clock.Fixed{T: time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)}
	user := domain.User{ID: id.New(), Email: "owner@example.com", Name: "Owner", PasswordHash: "x", CreatedAt: clk.Now()}
	if err := repos.Create(t.Context(), user); err != nil {
		t.Fatal(err)
	}

	access := putAccess(t, repos, user.ID, []string{"openid", "org"}, clk.Now().Add(time.Hour))

	body, _ := json.Marshal(map[string]string{"name": "Acme"})
	req := httptest.NewRequest(http.MethodPost, "/v1/organizations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create org: %d %s", rr.Code, rr.Body.String())
	}
	var created domain.Organization
	if err := json.NewDecoder(rr.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/organizations", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list orgs: %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/organizations/"+created.ID+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list members: %d %s", rr.Code, rr.Body.String())
	}
	var memberPayload struct {
		Members []domain.OrgMemberDetail `json:"members"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&memberPayload); err != nil {
		t.Fatal(err)
	}
	if len(memberPayload.Members) != 1 || memberPayload.Members[0].UserID != user.ID {
		t.Fatalf("members %+v", memberPayload.Members)
	}
	if memberPayload.Members[0].Email != "owner@example.com" || memberPayload.Members[0].Name != "Owner" {
		t.Fatalf("member profile %+v", memberPayload.Members[0])
	}

	primary, all, err := org.PrimaryOrg(t.Context(), repos, user.ID, "")
	if err != nil || len(all) != 1 || primary.OrgID != created.ID {
		t.Fatalf("primary %+v all=%d err=%v", primary, len(all), err)
	}

	req = httptest.NewRequest(http.MethodGet, "/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("userinfo: %d %s", rr.Code, rr.Body.String())
	}
	var ui map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&ui); err != nil {
		t.Fatal(err)
	}
	if ui["org_id"] != created.ID {
		t.Fatalf("userinfo org_id %+v", ui)
	}
}

func putAccess(t *testing.T, repos domain.Repos, userID string, scopes []string, exp time.Time) string {
	t.Helper()
	plain := id.New()
	if err := repos.PutAccess(t.Context(), domain.AccessToken{
		Hash: oauth.HashToken(plain), ClientID: "c1", UserID: userID, Scopes: scopes, ExpiresAt: exp,
	}); err != nil {
		t.Fatal(err)
	}
	return plain
}
