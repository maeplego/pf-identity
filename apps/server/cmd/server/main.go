package main

import (
	"log"
	"net/http"
	"os"

	"github.com/portfolio/pf-identity-server/internal/account"
	"github.com/portfolio/pf-identity-server/internal/clock"
	"github.com/portfolio/pf-identity-server/internal/config"
	"github.com/portfolio/pf-identity-server/internal/oidc"
	"github.com/portfolio/pf-identity-server/internal/session"
	"github.com/portfolio/pf-identity-server/internal/store/memory"
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
	if cfg.Store != config.StoreMemory {
		log.Fatal("only IDENTITY_STORE=memory is implemented in this revision; postgres follows")
	}
	store := memory.NewStore()
	clk := clock.Real{}
	acc := &account.Service{Users: store, Clock: clk}
	sess := &session.Service{Sessions: store, Clock: clk, TTL: cfg.SessionTTL}
	srv, err := web.NewServer(cfg, acc, sess, store, signer, clk)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s issuer=%s (learning IdP, not for production)", cfg.HTTPAddr, cfg.Issuer)
	if err := http.ListenAndServe(cfg.HTTPAddr, srv); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
