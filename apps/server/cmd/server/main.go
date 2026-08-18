package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/portfolio/pf-identity-server/internal/account"
	"github.com/portfolio/pf-identity-server/internal/clock"
	"github.com/portfolio/pf-identity-server/internal/config"
	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/oidc"
	"github.com/portfolio/pf-identity-server/internal/seed"
	"github.com/portfolio/pf-identity-server/internal/session"
	"github.com/portfolio/pf-identity-server/internal/store/memory"
	"github.com/portfolio/pf-identity-server/internal/store/postgres"
	"github.com/portfolio/pf-identity-server/internal/web"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatal(err)
	}
	var signer *oidc.Signer
	if cfg.DevGenerateKeys {
		signer, err = oidc.GenerateRSA()
	} else {
		signer, err = oidc.LoadRSA(cfg.RSAPrivateKeyPath)
	}
	if err != nil {
		log.Fatal(err)
	}
	repos, err := openRepos(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := seed.PublicClient(context.Background(), repos, cfg); err != nil {
		log.Fatal(err)
	}
	if err := seed.DemoRPB(context.Background(), repos, cfg); err != nil {
		log.Fatal(err)
	}
	if err := seed.DemoUser(context.Background(), repos, cfg); err != nil {
		log.Fatal(err)
	}
	clk := clock.Real{}
	acc := &account.Service{Users: repos, Clock: clk}
	sess := &session.Service{Sessions: repos, Clock: clk, TTL: cfg.SessionTTL}
	srv, err := web.NewServer(cfg, acc, sess, repos, signer, clk)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s issuer=%s (learning IdP, not for production)", cfg.HTTPAddr, cfg.Issuer)
	if err := http.ListenAndServe(cfg.HTTPAddr, srv); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func openRepos(cfg config.Config) (domain.Repos, error) {
	switch cfg.Store {
	case config.StorePostgres:
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return postgres.Open(ctx, cfg.DatabaseURL)
	default:
		return memory.NewStore(), nil
	}
}
