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
