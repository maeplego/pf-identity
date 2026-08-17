// Package seed inserts the local sample RP so a laptop demo does not need admin UI yet.
package seed

import (
	"context"
	"errors"
	"log"

	"github.com/portfolio/pf-identity-server/internal/config"
	"github.com/portfolio/pf-identity-server/internal/domain"
)

// PublicClient creates the env-configured public client if it is missing.
func PublicClient(ctx context.Context, clients domain.Clients, cfg config.Config) error {
	if cfg.SeedPublicClientID == "" {
		return nil
	}
	if cfg.SeedPublicRedirect == "" {
		return errors.New("IDENTITY_SEED_PUBLIC_REDIRECT_URI is required when seeding a public client")
	}
	_, err := clients.GetClient(ctx, cfg.SeedPublicClientID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	c := domain.Client{
		ID:           cfg.SeedPublicClientID,
		Name:         cfg.SeedPublicClientName,
		Type:         domain.ClientPublic,
		RedirectURIs: []string{cfg.SeedPublicRedirect},
	}
	if err := clients.CreateClient(ctx, c); err != nil {
		return err
	}
	log.Printf("seeded public client id=%s redirect=%s", c.ID, cfg.SeedPublicRedirect)
	return nil
}
