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
