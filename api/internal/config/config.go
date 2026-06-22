// Package config loads runtime configuration from the environment.
// The same binary runs in two scopes (onboard / shore); scope is config, not code.
package config

import (
	"fmt"
	"os"
	"strings"
)

type Scope string

const (
	ScopeOnboard Scope = "onboard"
	ScopeShore   Scope = "shore"
)

type Config struct {
	Scope    Scope
	VesselID string // required when Scope == onboard; the single vessel this deployment serves

	HTTPAddr string

	DatabaseURL string

	// Object storage (S3-compatible) for certificates, manuals, condition photos.
	S3Endpoint string
	// S3PublicEndpoint is the browser-reachable endpoint used to sign GET URLs.
	// Defaults to S3Endpoint; differs in Docker (internal host vs host-mapped port).
	S3PublicEndpoint string
	S3Region         string
	S3Bucket         string
	S3AccessKey      string
	S3SecretKey      string
	S3UseSSL         bool

	JWTSecret string

	// AdminBootstrapKey, when set, is registered as the seeded admin's API key at startup
	// so the operator always holds a working credential (no lockout). Basic/temporary auth.
	AdminBootstrapKey string

	// WebDir, when set, makes the server also serve the built web bundle from this dir
	// (single-container deploy). Empty in local dev, where Vite serves the web.
	WebDir string

	// AutoMigrate applies pending migrations at startup. Convenient for container
	// boot; default true.
	AutoMigrate bool
}

// Load reads configuration from the environment and validates invariants.
func Load() (*Config, error) {
	c := &Config{
		Scope:             Scope(getenv("SCOPE", string(ScopeShore))),
		VesselID:          os.Getenv("VESSEL_ID"),
		HTTPAddr:          getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:       getenv("DATABASE_URL", "postgres://mooring:mooring@localhost:5432/mooring?sslmode=disable"),
		S3Endpoint:        getenv("S3_ENDPOINT", "http://localhost:9000"),
		S3PublicEndpoint:  getenv("S3_PUBLIC_ENDPOINT", ""),
		S3Region:          getenv("S3_REGION", "us-east-1"),
		S3Bucket:          getenv("S3_BUCKET", "mooring"),
		S3AccessKey:       getenv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:       getenv("S3_SECRET_KEY", "minioadmin"),
		S3UseSSL:          getenv("S3_USE_SSL", "false") == "true",
		JWTSecret:         getenv("JWT_SECRET", "dev-insecure-change-me"),
		AdminBootstrapKey: os.Getenv("ADMIN_BOOTSTRAP_KEY"),
		WebDir:            os.Getenv("WEB_DIR"),
		AutoMigrate:       getenv("AUTO_MIGRATE", "true") != "false",
	}

	switch c.Scope {
	case ScopeOnboard:
		if strings.TrimSpace(c.VesselID) == "" {
			return nil, fmt.Errorf("SCOPE=onboard requires VESSEL_ID")
		}
	case ScopeShore:
		// fleet-wide; no single vessel
	default:
		return nil, fmt.Errorf("invalid SCOPE %q (want onboard|shore)", c.Scope)
	}
	return c, nil
}

func (c *Config) IsOnboard() bool { return c.Scope == ScopeOnboard }

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
