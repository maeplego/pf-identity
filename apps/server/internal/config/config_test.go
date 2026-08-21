package config

import (
	"os"
	"testing"
	"time"
)

func TestFromEnvMemoryDefaults(t *testing.T) {
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "true")
	t.Setenv("IDENTITY_STORE", "memory")
	os.Unsetenv("IDENTITY_DATABASE_URL")
	os.Unsetenv("IDENTITY_RSA_PRIVATE_KEY_PATH")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if cfg.Issuer != "http://localhost:8080" {
		t.Fatalf("issuer = %q", cfg.Issuer)
	}
	if cfg.SessionTTL != 8*time.Hour {
		t.Fatalf("session ttl = %s", cfg.SessionTTL)
	}
	if cfg.CookieSecure {
		t.Fatal("cookie secure should default false for local HTTP")
	}
}

func TestFromEnvPostgresRequiresURL(t *testing.T) {
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "true")
	t.Setenv("IDENTITY_STORE", "postgres")
	os.Unsetenv("IDENTITY_DATABASE_URL")

	if _, err := FromEnv(); err == nil {
		t.Fatal("expected error when postgres store has no URL")
	}
}

func TestFromEnvBlankIssuerFallsBack(t *testing.T) {
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "true")
	t.Setenv("IDENTITY_ISSUER", "   ")
	t.Setenv("IDENTITY_STORE", "memory")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Issuer != "http://localhost:8080" {
		t.Fatalf("issuer = %q", cfg.Issuer)
	}
}

func TestFromEnvInvalidDuration(t *testing.T) {
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "true")
	t.Setenv("IDENTITY_STORE", "memory")
	t.Setenv("IDENTITY_CODE_TTL", "nope")

	if _, err := FromEnv(); err == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestFromEnvRequiresKeySource(t *testing.T) {
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "false")
	t.Setenv("IDENTITY_STORE", "memory")
	os.Unsetenv("IDENTITY_RSA_PRIVATE_KEY_PATH")

	if _, err := FromEnv(); err == nil {
		t.Fatal("expected error when no key material is configured")
	}
}

func TestFromEnvSeedPublicClient(t *testing.T) {
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "true")
	t.Setenv("IDENTITY_STORE", "memory")
	t.Setenv("IDENTITY_SEED_PUBLIC_CLIENT_ID", "sample-rp")
	t.Setenv("IDENTITY_SEED_PUBLIC_REDIRECT_URI", "http://localhost:3001/callback")
	t.Setenv("IDENTITY_SEED_PUBLIC_CLIENT_NAME", "Demo RP")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SeedPublicClientID != "sample-rp" || cfg.SeedPublicRedirect != "http://localhost:3001/callback" {
		t.Fatalf("%+v", cfg)
	}
	if cfg.SeedPublicClientName != "Demo RP" {
		t.Fatalf("name %q", cfg.SeedPublicClientName)
	}
}

func TestFromEnvSeedPostLogout(t *testing.T) {
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "true")
	t.Setenv("IDENTITY_STORE", "memory")
	t.Setenv("IDENTITY_SEED_PUBLIC_CLIENT_ID", "sample-rp")
	t.Setenv("IDENTITY_SEED_PUBLIC_REDIRECT_URI", "http://localhost:3001/callback")
	t.Setenv("IDENTITY_SEED_PUBLIC_POST_LOGOUT_REDIRECT_URI", "http://localhost:3001/logged-out")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SeedPublicPostLogout != "http://localhost:3001/logged-out" {
		t.Fatalf("post_logout %q", cfg.SeedPublicPostLogout)
	}
}

func TestFromEnvAdminToken(t *testing.T) {
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "true")
	t.Setenv("IDENTITY_STORE", "memory")
	t.Setenv("IDENTITY_ADMIN_TOKEN", "dev-admin-token")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AdminToken != "dev-admin-token" {
		t.Fatalf("admin token %q", cfg.AdminToken)
	}
}

func TestFromEnvSeedDemoUser(t *testing.T) {
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "true")
	t.Setenv("IDENTITY_STORE", "memory")
	t.Setenv("IDENTITY_SEED_DEMO_EMAIL", "demo@example.test")
	t.Setenv("IDENTITY_SEED_DEMO_PASSWORD", "demo-pass-change-me")
	t.Setenv("IDENTITY_SEED_DEMO_NAME", "Demo User")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SeedDemoEmail != "demo@example.test" || cfg.SeedDemoName != "Demo User" {
		t.Fatalf("%+v", cfg)
	}
	if cfg.SeedDemoPassword == "" {
		t.Fatal("password should be loaded")
	}
	if cfg.Env != EnvDevelopment {
		t.Fatalf("env = %q", cfg.Env)
	}
}

func TestFromEnvProductionRejectsDevKeys(t *testing.T) {
	t.Setenv("IDENTITY_ENV", "production")
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "true")
	t.Setenv("IDENTITY_STORE", "postgres")
	t.Setenv("IDENTITY_DATABASE_URL", "postgres://idp:idp@localhost/idp")
	t.Setenv("IDENTITY_RSA_PRIVATE_KEY_PATH", "/keys/idp.pem")
	t.Setenv("IDENTITY_COOKIE_SECURE", "true")

	if _, err := FromEnv(); err == nil {
		t.Fatal("expected error when production enables IDENTITY_DEV_GENERATE_KEYS")
	}
}

func TestFromEnvProductionRequiresSecureCookie(t *testing.T) {
	t.Setenv("IDENTITY_ENV", "production")
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "false")
	t.Setenv("IDENTITY_STORE", "postgres")
	t.Setenv("IDENTITY_DATABASE_URL", "postgres://idp:idp@localhost/idp")
	t.Setenv("IDENTITY_RSA_PRIVATE_KEY_PATH", "/keys/idp.pem")
	t.Setenv("IDENTITY_COOKIE_SECURE", "false")

	if _, err := FromEnv(); err == nil {
		t.Fatal("expected error when production has CookieSecure false")
	}
}

func TestFromEnvStagingAllowsInsecureCookieWithFileKeys(t *testing.T) {
	t.Setenv("IDENTITY_ENV", "staging")
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "false")
	t.Setenv("IDENTITY_STORE", "postgres")
	t.Setenv("IDENTITY_DATABASE_URL", "postgres://idp:idp@localhost/idp")
	t.Setenv("IDENTITY_RSA_PRIVATE_KEY_PATH", "/keys/idp.pem")
	t.Setenv("IDENTITY_COOKIE_SECURE", "false")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != EnvStaging {
		t.Fatalf("env = %q", cfg.Env)
	}
}

func TestFromEnvProductionOK(t *testing.T) {
	t.Setenv("IDENTITY_ENV", "production")
	t.Setenv("IDENTITY_DEV_GENERATE_KEYS", "false")
	t.Setenv("IDENTITY_STORE", "postgres")
	t.Setenv("IDENTITY_DATABASE_URL", "postgres://idp:idp@localhost/idp")
	t.Setenv("IDENTITY_RSA_PRIVATE_KEY_PATH", "/keys/idp.pem")
	t.Setenv("IDENTITY_COOKIE_SECURE", "true")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != EnvProduction || !cfg.CookieSecure {
		t.Fatalf("%+v", cfg)
	}
}
