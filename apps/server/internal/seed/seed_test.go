package seed

import (
	"context"
	"testing"

	"github.com/portfolio/pf-identity-server/internal/config"
	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/store/memory"
)

func TestPublicClientInsertsOnce(t *testing.T) {
	store := memory.NewStore()
	cfg := config.Config{
		SeedPublicClientID:   "sample-rp",
		SeedPublicClientName: "Sample RP",
		SeedPublicRedirect:   "http://localhost:3001/callback",
	}
	if err := PublicClient(context.Background(), store, cfg); err != nil {
		t.Fatal(err)
	}
	if err := PublicClient(context.Background(), store, cfg); err != nil {
		t.Fatal(err)
	}
	c, err := store.GetClient(context.Background(), "sample-rp")
	if err != nil {
		t.Fatal(err)
	}
	if c.Type != domain.ClientPublic || c.RedirectURIs[0] != cfg.SeedPublicRedirect {
		t.Fatalf("%+v", c)
	}
	if len(c.PostLogoutRedirectURIs) != 1 || c.PostLogoutRedirectURIs[0] != "http://localhost:3001/logged-out" {
		t.Fatalf("post_logout %+v", c.PostLogoutRedirectURIs)
	}
	if c.FrontChannelLogoutURI != "http://localhost:3001/frontchannel-logout" {
		t.Fatalf("frontchannel %q", c.FrontChannelLogoutURI)
	}
	if c.BackChannelLogoutURI != "http://localhost:3001/backchannel-logout" {
		t.Fatalf("backchannel %q", c.BackChannelLogoutURI)
	}
}

func TestPublicClientBackfillsPostLogout(t *testing.T) {
	store := memory.NewStore()
	cfg := config.Config{
		SeedPublicClientID:   "sample-rp",
		SeedPublicClientName: "Sample RP",
		SeedPublicRedirect:   "http://localhost:3001/callback",
	}
	if err := store.CreateClient(context.Background(), domain.Client{
		ID:           cfg.SeedPublicClientID,
		Name:         cfg.SeedPublicClientName,
		Type:         domain.ClientPublic,
		RedirectURIs: []string{cfg.SeedPublicRedirect},
	}); err != nil {
		t.Fatal(err)
	}
	if err := PublicClient(context.Background(), store, cfg); err != nil {
		t.Fatal(err)
	}
	c, err := store.GetClient(context.Background(), "sample-rp")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.PostLogoutRedirectURIs) != 1 || c.PostLogoutRedirectURIs[0] != "http://localhost:3001/logged-out" {
		t.Fatalf("%+v", c.PostLogoutRedirectURIs)
	}
}

func TestPublicClientSkippedWhenUnset(t *testing.T) {
	store := memory.NewStore()
	if err := PublicClient(context.Background(), store, config.Config{}); err != nil {
		t.Fatal(err)
	}
}

func TestPublicClientRequiresRedirect(t *testing.T) {
	err := PublicClient(context.Background(), memory.NewStore(), config.Config{SeedPublicClientID: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDemoRPBInsertsSecondClient(t *testing.T) {
	store := memory.NewStore()
	cfg := config.Config{
		SeedDemoRPBRedirect: "http://localhost:3003/callback",
	}
	if err := DemoRPB(context.Background(), store, cfg); err != nil {
		t.Fatal(err)
	}
	c, err := store.GetClient(context.Background(), "sample-rp-b")
	if err != nil {
		t.Fatal(err)
	}
	if c.RedirectURIs[0] != cfg.SeedDemoRPBRedirect || c.BackChannelLogoutURI != "http://localhost:3003/backchannel-logout" {
		t.Fatalf("%+v", c)
	}
}

func TestDemoRPBSkippedWhenUnset(t *testing.T) {
	store := memory.NewStore()
	if err := DemoRPB(context.Background(), store, config.Config{}); err != nil {
		t.Fatal(err)
	}
}
