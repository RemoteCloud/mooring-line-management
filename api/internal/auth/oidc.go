package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	// umBase is the UserManagement origin (issuer minus the tenant path segment);
	// tenant is that segment. Used to call the external positionTeams API.
	umBase string
	tenant string
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

	umBase, tenant := splitIssuer(cfg.OIDCIssuer)

	return &Authenticator{
		provider: provider,
		verifier: verifier,
		oauth:    oauthCfg,
		prompt:   cfg.OIDCPrompt,
		clientID: cfg.OIDCClientID,
		umBase:   umBase,
		tenant:   tenant,
	}, nil
}

// splitIssuer separates a UserManagement issuer URL into its origin (scheme +
// host) and the trailing tenant path segment. For
// "https://administration.example.com/nightly" it returns
// ("https://administration.example.com", "nightly"). A path-less issuer yields
// an empty tenant.
func splitIssuer(issuer string) (base, tenant string) {
	u, err := url.Parse(strings.TrimRight(issuer, "/"))
	if err != nil || u.Host == "" {
		return strings.TrimRight(issuer, "/"), ""
	}
	tenant = strings.Trim(u.Path, "/")
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), tenant
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

// PositionTeam is one UserManagement position team (the unit in-app access
// grants are keyed on): an opaque id and its human name.
type PositionTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FetchPositionTeams lists the tenant's position teams via the UM external API
// (GET {umBase}/external/api/positionTeams?onlyActive=true) using the caller's
// access token. Mirrors the reference app: it provides the id->name mapping the
// access-control admin UI shows instead of raw GUIDs.
//
// Best-effort — on any transport / auth / shape failure it returns an error the
// caller logs and ignores (the UI degrades to GUIDs), so a UM hiccup never
// breaks access management.
func (a *Authenticator) FetchPositionTeams(ctx context.Context, accessToken string) ([]PositionTeam, error) {
	if a.umBase == "" || accessToken == "" {
		return nil, fmt.Errorf("position teams: missing base url or access token")
	}
	endpoint := a.umBase + "/external/api/positionTeams?onlyActive=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	if a.tenant != "" {
		req.Header.Set("Tenant", a.tenant)
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("position teams: status %d", resp.StatusCode)
	}

	// UM wraps list results as {"data":[...]}; tolerate a bare array too. Field
	// casing varies (id/Id, name/Name).
	var body struct {
		Data []map[string]any `json:"data"`
	}
	raw, _ := io.ReadAll(resp.Body)
	teams := []PositionTeam{}
	if err := json.Unmarshal(raw, &body); err == nil && body.Data != nil {
		teams = mapTeams(body.Data)
	} else {
		var arr []map[string]any
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, fmt.Errorf("position teams: decode: %w", err)
		}
		teams = mapTeams(arr)
	}
	return teams, nil
}

func mapTeams(rows []map[string]any) []PositionTeam {
	out := make([]PositionTeam, 0, len(rows))
	pick := func(m map[string]any, keys ...string) string {
		for _, k := range keys {
			if s, ok := m[k].(string); ok && s != "" {
				return s
			}
		}
		return ""
	}
	for _, r := range rows {
		id := pick(r, "id", "Id", "ID")
		if id == "" {
			continue
		}
		out = append(out, PositionTeam{ID: id, Name: pick(r, "name", "Name")})
	}
	return out
}

// GenerateVerifier returns a fresh PKCE code verifier.
func GenerateVerifier() string { return oauth2.GenerateVerifier() }

// S256Challenge derives the S256 PKCE challenge for a verifier.
func S256Challenge(verifier string) string { return oauth2.S256ChallengeFromVerifier(verifier) }
