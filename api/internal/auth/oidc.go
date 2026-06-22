package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/ncl/mooring-api/internal/config"
)

// Authenticator wraps the OIDC provider + oauth2 config. The underlying go-oidc
// Provider caches the JWKS / discovery document, so a single instance is reused
// for the process lifetime.
type Authenticator struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth    oauth2.Config
	prompt   string
	clientID string
}

// NewAuthenticator performs OIDC discovery against the issuer and builds the
// oauth2 config. RS512 is the only id_token signing alg the provider offers, so
// the verifier is pinned to it.
func NewAuthenticator(ctx context.Context, cfg *config.Config) (*Authenticator, error) {
	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery (%s): %w", cfg.OIDCIssuer, err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID:             cfg.OIDCClientID,
		SupportedSigningAlgs: []string{oidc.RS512},
	})

	oauthCfg := oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.OIDCRedirectURI,
		Scopes:       strings.Fields(cfg.OIDCScopes),
	}

	return &Authenticator{
		provider: provider,
		verifier: verifier,
		oauth:    oauthCfg,
		prompt:   cfg.OIDCPrompt,
		clientID: cfg.OIDCClientID,
	}, nil
}

// AuthCodeURL builds the provider authorization URL for the auth-code + PKCE flow.
func (a *Authenticator) AuthCodeURL(state, nonce, pkceChallenge string) string {
	opts := []oauth2.AuthCodeOption{
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	if a.prompt != "" {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", a.prompt))
	}
	return a.oauth.AuthCodeURL(state, opts...)
}

// Exchange swaps an auth code for tokens, supplying the PKCE verifier.
func (a *Authenticator) Exchange(ctx context.Context, code, verifier string) (*oauth2.Token, error) {
	return a.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
}

// VerifyIDToken verifies an id_token's signature (RS512), issuer and audience.
func (a *Authenticator) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	return a.verifier.Verify(ctx, rawIDToken)
}

// UserInfo fetches the userinfo endpoint using the access token.
func (a *Authenticator) UserInfo(ctx context.Context, tok *oauth2.Token) (*oidc.UserInfo, error) {
	return a.provider.UserInfo(ctx, oauth2.StaticTokenSource(tok))
}

// Refresh exchanges a refresh token for a fresh access token.
func (a *Authenticator) Refresh(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	ts := a.oauth.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	return ts.Token()
}

// GenerateVerifier returns a fresh PKCE code verifier.
func GenerateVerifier() string { return oauth2.GenerateVerifier() }

// S256Challenge derives the S256 PKCE challenge for a verifier.
func S256Challenge(verifier string) string { return oauth2.S256ChallengeFromVerifier(verifier) }
