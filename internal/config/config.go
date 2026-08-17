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

// Config is the process configuration. Fields are value types so tests can copy them.
type Config struct {
	HTTPAddr            string
	Issuer              string
	DatabaseURL         string
	CookieSecure        bool
	DevGenerateKeys     bool
	RSAPrivateKeyPath   string
	Store               string
	SessionTTL          time.Duration
	CodeTTL             time.Duration
	AccessTTL           time.Duration
	RefreshTTL          time.Duration
}

// FromEnv reads IDENTITY_* variables. Missing optional values use conservative locals.
func FromEnv() (Config, error) {
	cfg := Config{
		HTTPAddr:        envOr("IDENTITY_HTTP_ADDR", ":8080"),
		Issuer:          strings.TrimRight(envOr("IDENTITY_ISSUER", "http://localhost:8080"), "/"),
		DatabaseURL:     os.Getenv("IDENTITY_DATABASE_URL"),
		CookieSecure:    envBool("IDENTITY_COOKIE_SECURE", false),
		DevGenerateKeys: envBool("IDENTITY_DEV_GENERATE_KEYS", false),
		RSAPrivateKeyPath: os.Getenv("IDENTITY_RSA_PRIVATE_KEY_PATH"),
		Store:           strings.ToLower(envOr("IDENTITY_STORE", StoreMemory)),
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
	return cfg, nil
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
