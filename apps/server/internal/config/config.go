// Package config loads process settings from the environment.
// Values that could be secrets (database URL, key paths) never have defaults that
// point at real infrastructure; callers must opt in.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// StoreMemory keeps users and tokens in process memory (tests and first-run demos).
const StoreMemory = "memory"

// StorePostgres is the durable store required beyond local experiments.
const StorePostgres = "postgres"

// EnvDevelopment is the default local/demo profile (dev key gen and memory store allowed).
const EnvDevelopment = "development"

// EnvStaging is a pre-production profile: durable store and file keys required.
const EnvStaging = "staging"

// EnvProduction rejects demo shortcuts (ephemeral keys, memory store, insecure cookies).
const EnvProduction = "production"

// Config is the process configuration. Fields are value types so tests can copy them.
type Config struct {
	Env                    string
	HTTPAddr               string
	Issuer                 string
	DatabaseURL            string
	CookieSecure           bool
	DevGenerateKeys        bool
	RSAPrivateKeyPath      string
	Store                  string
	SessionTTL             time.Duration
	CodeTTL                time.Duration
	AccessTTL              time.Duration
	RefreshTTL             time.Duration
	SeedPublicClientID     string
	SeedPublicClientName   string
	SeedPublicRedirect     string
	SeedPublicPostLogout   string
	SeedDemoRPBClientID    string
	SeedDemoRPBClientName  string
	SeedDemoRPBRedirect    string
	SeedDemoRPBPostLogout  string
	AdminToken             string
	SeedDemoEmail          string
	SeedDemoPassword       string
	SeedDemoName           string
	SeedExtraClientsJSON   string
}

// FromEnv reads IDENTITY_* variables. Missing optional values use conservative locals.
func FromEnv() (Config, error) {
	cfg := Config{
		Env:                    normalizeEnv(os.Getenv("IDENTITY_ENV")),
		HTTPAddr:               envOr("IDENTITY_HTTP_ADDR", ":8080"),
		Issuer:                 strings.TrimRight(envOr("IDENTITY_ISSUER", "http://localhost:8080"), "/"),
		DatabaseURL:            os.Getenv("IDENTITY_DATABASE_URL"),
		CookieSecure:           envBool("IDENTITY_COOKIE_SECURE", false),
		DevGenerateKeys:        envBool("IDENTITY_DEV_GENERATE_KEYS", false),
		RSAPrivateKeyPath:      os.Getenv("IDENTITY_RSA_PRIVATE_KEY_PATH"),
		Store:                  strings.ToLower(envOr("IDENTITY_STORE", StoreMemory)),
		SeedPublicClientID:     strings.TrimSpace(os.Getenv("IDENTITY_SEED_PUBLIC_CLIENT_ID")),
		SeedPublicClientName:   envOr("IDENTITY_SEED_PUBLIC_CLIENT_NAME", "Sample RP"),
		SeedPublicRedirect:     strings.TrimSpace(os.Getenv("IDENTITY_SEED_PUBLIC_REDIRECT_URI")),
		SeedPublicPostLogout:   strings.TrimSpace(os.Getenv("IDENTITY_SEED_PUBLIC_POST_LOGOUT_REDIRECT_URI")),
		SeedDemoRPBClientID:    strings.TrimSpace(os.Getenv("IDENTITY_SEED_DEMO_RP_B_CLIENT_ID")),
		SeedDemoRPBClientName:  envOr("IDENTITY_SEED_DEMO_RP_B_CLIENT_NAME", "Sample RP B"),
		SeedDemoRPBRedirect:    strings.TrimSpace(os.Getenv("IDENTITY_SEED_DEMO_RP_B_REDIRECT_URI")),
		SeedDemoRPBPostLogout:  strings.TrimSpace(os.Getenv("IDENTITY_SEED_DEMO_RP_B_POST_LOGOUT_REDIRECT_URI")),
		AdminToken:             strings.TrimSpace(os.Getenv("IDENTITY_ADMIN_TOKEN")),
		SeedDemoEmail:          strings.TrimSpace(os.Getenv("IDENTITY_SEED_DEMO_EMAIL")),
		SeedDemoPassword:       os.Getenv("IDENTITY_SEED_DEMO_PASSWORD"),
		SeedDemoName:           envOr("IDENTITY_SEED_DEMO_NAME", "Demo User"),
		SeedExtraClientsJSON:   strings.TrimSpace(os.Getenv("IDENTITY_SEED_EXTRA_CLIENTS")),
	}

	var err error
	if cfg.SessionTTL, err = envDuration("IDENTITY_SESSION_TTL", 8*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.CodeTTL, err = envDuration("IDENTITY_CODE_TTL", time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.AccessTTL, err = envDuration("IDENTITY_ACCESS_TTL", 15*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.RefreshTTL, err = envDuration("IDENTITY_REFRESH_TTL", 7*24*time.Hour); err != nil {
		return Config{}, err
	}

	if cfg.Store != StoreMemory && cfg.Store != StorePostgres {
		return Config{}, fmt.Errorf("unsupported IDENTITY_STORE %q", cfg.Store)
	}
	if cfg.Store == StorePostgres && cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("IDENTITY_DATABASE_URL is required when IDENTITY_STORE=postgres")
	}
	if !cfg.DevGenerateKeys && cfg.RSAPrivateKeyPath == "" {
		return Config{}, fmt.Errorf("set IDENTITY_RSA_PRIVATE_KEY_PATH or IDENTITY_DEV_GENERATE_KEYS=true")
	}
	if err := cfg.validateProfile(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func normalizeEnv(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "dev", "development", "local", "demo":
		return EnvDevelopment
	case "staging", "stage":
		return EnvStaging
	case "production", "prod":
		return EnvProduction
	default:
		return strings.ToLower(strings.TrimSpace(v))
	}
}

// validateProfile rejects demo shortcuts when IDENTITY_ENV is staging or production.
func (cfg Config) validateProfile() error {
	switch cfg.Env {
	case EnvDevelopment:
		return nil
	case EnvStaging, EnvProduction:
		if cfg.DevGenerateKeys {
			return fmt.Errorf("IDENTITY_DEV_GENERATE_KEYS must be false when IDENTITY_ENV=%s", cfg.Env)
		}
		if cfg.Store != StorePostgres {
			return fmt.Errorf("IDENTITY_STORE=postgres is required when IDENTITY_ENV=%s", cfg.Env)
		}
		if strings.TrimSpace(cfg.RSAPrivateKeyPath) == "" {
			return fmt.Errorf("IDENTITY_RSA_PRIVATE_KEY_PATH is required when IDENTITY_ENV=%s", cfg.Env)
		}
		if cfg.Env == EnvProduction && !cfg.CookieSecure {
			return fmt.Errorf("IDENTITY_COOKIE_SECURE=true is required when IDENTITY_ENV=production")
		}
		return nil
	default:
		return fmt.Errorf("unsupported IDENTITY_ENV %q (use development, staging, or production)", cfg.Env)
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return d, nil
}
