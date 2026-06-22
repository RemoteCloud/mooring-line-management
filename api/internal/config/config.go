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

	// OIDC / authentication. The app authenticates users against an external
	// OpenID Connect provider (Backend-for-Frontend pattern: tokens stay server
	// side, the browser only holds an opaque session cookie).
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURI  string
	OIDCScopes       string // space-separated scope list
	OIDCPrompt       string // e.g. "login"
	OIDCAdminGroup   string // group name that grants admin (read+write)

	// SessionSecret is a server secret; in dev it doubles as the source for the
	// token-encryption key when TokenEncKey is unset.
	SessionSecret string
	// TokenEncKey is a base64-encoded 32-byte key for AES-256-GCM token-at-rest
	// encryption. If empty, derived from SessionSecret (dev only).
	TokenEncKey string
	// AppBaseURL is the public origin (and base path) the browser reaches the app
	// on. Used to decide cookie Secure flag and to build redirects.
	AppBaseURL string

	// AutoMigrate applies pending migrations at startup. Convenient for container
	// boot; default true.
	AutoMigrate bool
}

// DefaultOIDCIssuer is the confirmed nightly provider issuer.
const DefaultOIDCIssuer = "https://administration.cloud.maranics-nightly.com/nightly"

// Load reads configuration from the environment and validates invariants.
func Load() (*Config, error) {
	c := &Config{
		Scope:            Scope(getenv("SCOPE", string(ScopeShore))),
		VesselID:         os.Getenv("VESSEL_ID"),
		HTTPAddr:         getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:      getenv("DATABASE_URL", "postgres://mooring:mooring@localhost:5432/mooring?sslmode=disable"),
		S3Endpoint:       getenv("S3_ENDPOINT", "http://localhost:9000"),
		S3PublicEndpoint: getenv("S3_PUBLIC_ENDPOINT", ""),
		S3Region:         getenv("S3_REGION", "us-east-1"),
		S3Bucket:         getenv("S3_BUCKET", "mooring"),
		S3AccessKey:      getenv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:      getenv("S3_SECRET_KEY", "minioadmin"),
		S3UseSSL:         getenv("S3_USE_SSL", "false") == "true",
		OIDCIssuer:       getenv("OIDC_ISSUER", DefaultOIDCIssuer),
		OIDCClientID:     os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURI:  getenv("OIDC_REDIRECT_URI", "http://localhost:8091/api/auth/callback"),
		OIDCScopes:       getenv("OIDC_SCOPES", "openid email profile roles offline_access"),
		OIDCPrompt:       getenv("OIDC_PROMPT", "login"),
		OIDCAdminGroup:   getenv("OIDC_ADMIN_GROUP", "admin"),
		SessionSecret:    os.Getenv("SESSION_SECRET"),
		TokenEncKey:      os.Getenv("TOKEN_ENC_KEY"),
		AppBaseURL:       getenv("APP_BASE_URL", "http://localhost:8091"),
		AutoMigrate:      getenv("AUTO_MIGRATE", "true") != "false",
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

// ValidateServe checks invariants required only when actually serving traffic.
// Codegen/migration subcommands don't need OIDC credentials, so this is gated to
// the serve path rather than folded into Load().
func (c *Config) ValidateServe() error {
	var missing []string
	if strings.TrimSpace(c.OIDCClientID) == "" {
		missing = append(missing, "OIDC_CLIENT_ID")
	}
	if strings.TrimSpace(c.OIDCClientSecret) == "" {
		missing = append(missing, "OIDC_CLIENT_SECRET")
	}
	if strings.TrimSpace(c.OIDCRedirectURI) == "" {
		missing = append(missing, "OIDC_REDIRECT_URI")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required auth config: %s", strings.Join(missing, ", "))
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
