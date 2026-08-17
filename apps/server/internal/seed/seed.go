// Package seed inserts the local sample RP so a laptop demo does not need admin UI yet.
package seed

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/portfolio/pf-identity-server/internal/config"
	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/oauth"
)

// PublicClient creates or updates the env-configured public client.
func PublicClient(ctx context.Context, clients domain.Clients, cfg config.Config) error {
	if cfg.SeedPublicClientID == "" {
		return nil
	}
	if cfg.SeedPublicRedirect == "" {
		return errors.New("IDENTITY_SEED_PUBLIC_REDIRECT_URI is required when seeding a public client")
	}
	postLogout := cfg.SeedPublicPostLogout
	if postLogout == "" {
		var err error
		postLogout, err = defaultPostLogoutURI(cfg.SeedPublicRedirect)
		if err != nil {
			return err
		}
	}
	front, err := replaceCallbackPath(cfg.SeedPublicRedirect, "/frontchannel-logout")
	if err != nil {
		return err
	}
	back, err := replaceCallbackPath(cfg.SeedPublicRedirect, "/backchannel-logout")
	if err != nil {
		return err
	}
	want := domain.Client{
		ID:                     cfg.SeedPublicClientID,
		Name:                   cfg.SeedPublicClientName,
		Type:                   domain.ClientPublic,
		RedirectURIs:           []string{cfg.SeedPublicRedirect},
		PostLogoutRedirectURIs: []string{postLogout},
		FrontChannelLogoutURI:  front,
		BackChannelLogoutURI:   back,
	}
	existing, err := clients.GetClient(ctx, cfg.SeedPublicClientID)
	if err == nil {
		redirects := unionExact(existing.RedirectURIs, want.RedirectURIs)
		posts := unionExact(existing.PostLogoutRedirectURIs, want.PostLogoutRedirectURIs)
		if existing.Name == want.Name && sameStrings(existing.RedirectURIs, redirects) && sameStrings(existing.PostLogoutRedirectURIs, posts) && existing.FrontChannelLogoutURI == want.FrontChannelLogoutURI && existing.BackChannelLogoutURI == want.BackChannelLogoutURI {
			return nil
		}
		if err := clients.UpdateClient(ctx, existing.ID, domain.ClientPatch{
			Name:                   want.Name,
			RedirectURIs:           redirects,
			PostLogoutRedirectURIs: posts,
			FrontChannelLogoutURI:  want.FrontChannelLogoutURI,
			BackChannelLogoutURI:   want.BackChannelLogoutURI,
		}); err != nil {
			return err
		}
		log.Printf("updated seeded public client id=%s redirect=%s post_logout=%s frontchannel=%s backchannel=%s", want.ID, cfg.SeedPublicRedirect, postLogout, front, back)
		return nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if err := clients.CreateClient(ctx, want); err != nil {
		return err
	}
	log.Printf("seeded public client id=%s redirect=%s post_logout=%s frontchannel=%s backchannel=%s", want.ID, cfg.SeedPublicRedirect, postLogout, front, back)
	return nil
}

func defaultPostLogoutURI(redirect string) (string, error) {
	return replaceCallbackPath(redirect, "/logged-out")
}

func replaceCallbackPath(redirect, path string) (string, error) {
	u, err := oauth.ParseRedirectURI(redirect)
	if err != nil {
		return "", err
	}
	u.RawQuery = ""
	u.Fragment = ""
	if strings.HasSuffix(u.Path, "/callback") {
		u.Path = strings.TrimSuffix(u.Path, "/callback") + path
	} else {
		u.Path = path
	}
	return u.String(), nil
}

func unionExact(have, add []string) []string {
	out := append([]string{}, have...)
	seen := map[string]struct{}{}
	for _, item := range out {
		seen[item] = struct{}{}
	}
	for _, item := range add {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
