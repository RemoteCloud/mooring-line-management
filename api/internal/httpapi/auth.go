package httpapi

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/auth"
)

const sessionCookieName = "mlm_session"

// oauthStateCookieName binds the in-flight login's state to the browser that
// started it, defeating login-CSRF / session-fixation on the callback.
const oauthStateCookieName = "mlm_oauth_state"

// flowTTL bounds how long an in-flight login may take before its state expires.
const flowTTL = 10 * time.Minute

// ctxUserKey / ctxPermsKey carry the authenticated principal through the request
// context for downstream handlers.
type ctxKey string

const (
	ctxUserKey  ctxKey = "auth.user"
	ctxPermsKey ctxKey = "auth.perms"
)

// registerAuth wires the OIDC login/callback/logout/session endpoints. The
// redirect+cookie endpoints are raw net/http handlers (they need full control of
// Set-Cookie and 302 redirects); /auth/session is a normal Huma JSON endpoint.
func registerAuth(api huma.API, s *Server, mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/login", s.handleLogin)
	mux.HandleFunc("GET /auth/callback", s.handleCallback)
	mux.HandleFunc("POST /auth/logout", s.handleLogout)

	registerSession(api, s)
}

// secureCookie reports whether the Secure cookie attribute should be set, based
// on the configured public base URL.
func (s *Server) secureCookie() bool {
	return strings.HasPrefix(strings.ToLower(s.Cfg.AppBaseURL), "https://")
}

func (s *Server) setSessionCookie(w http.ResponseWriter, sid string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// setOAuthStateCookie stores the login's state value in a short-lived,
// HttpOnly cookie so the callback can prove the same browser started the flow.
//
// SameSite=Lax is correct here: the callback is a top-level GET navigation
// (the IdP 302s the browser back to /auth/callback), and Lax cookies ARE sent
// on top-level cross-site navigations. SameSite=None is wrong — it is treated
// as a third-party cookie and blocked by default in incognito/strict modes.
func (s *Server) setOAuthStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(flowTTL / time.Second),
	})
}

func (s *Server) clearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// handleLogin starts the OIDC auth-code + PKCE flow and redirects to the provider.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.Auth == nil || s.Store == nil {
		http.Error(w, "auth not configured", http.StatusServiceUnavailable)
		return
	}
	returnTo := r.URL.Query().Get("return_to")
	if returnTo == "" {
		returnTo = "/"
	}

	state, err1 := auth.RandomID(32)
	nonce, err2 := auth.RandomID(32)
	if err1 != nil || err2 != nil {
		http.Error(w, "failed to start login", http.StatusInternalServerError)
		return
	}
	verifier := auth.GenerateVerifier()
	challenge := auth.S256Challenge(verifier)

	if err := s.Store.CreateFlow(r.Context(), storeFlow(state, verifier, nonce, returnTo)); err != nil {
		slog.Error("persist oidc flow", "err", err)
		http.Error(w, "failed to start login", http.StatusInternalServerError)
		return
	}

	// Bind the state to this browser; the callback verifies it before consuming
	// the server-side flow, so a state minted in one session can't be redeemed
	// in another (login CSRF / fixation).
	s.setOAuthStateCookie(w, state)

	http.Redirect(w, r, s.Auth.AuthCodeURL(state, nonce, challenge), http.StatusFound)
}

// handleCallback completes the flow: validate state, exchange code, verify
// id_token, fetch userinfo, upsert user, create session, set cookie, redirect.
func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	if s.Auth == nil || s.Store == nil {
		http.Error(w, "auth not configured", http.StatusServiceUnavailable)
		return
	}
	ctx := r.Context()
	q := r.URL.Query()

	if errParam := q.Get("error"); errParam != "" {
		s.redirectAuthError(w, r, errParam)
		return
	}
	code := q.Get("code")
	state := q.Get("state")
	if code == "" || state == "" {
		s.redirectAuthError(w, r, "missing_code_or_state")
		return
	}

	// Prove the callback lands in the same browser that started the login: the
	// state-binding cookie must match the state query param (constant-time). The
	// cookie is single-use regardless of outcome.
	boundState := ""
	if c, err := r.Cookie(oauthStateCookieName); err == nil {
		boundState = c.Value
	}
	s.clearOAuthStateCookie(w)
	if boundState == "" || subtle.ConstantTimeCompare([]byte(boundState), []byte(state)) != 1 {
		slog.Warn("oauth state binding failed",
			"cookie_present", boundState != "",
			"cookie_matches_query", boundState != "" && boundState == state,
			"has_state_query", state != "")
		s.redirectAuthError(w, r, "state_mismatch")
		return
	}

	flow, err := s.Store.TakeFlow(ctx, state)
	if err != nil {
		s.redirectAuthError(w, r, "invalid_state")
		return
	}
	if time.Since(flow.CreatedAt) > flowTTL {
		s.redirectAuthError(w, r, "state_expired")
		return
	}

	tok, err := s.Auth.Exchange(ctx, code, flow.CodeVerifier)
	if err != nil {
		slog.Error("oidc token exchange", "err", err)
		s.redirectAuthError(w, r, "exchange_failed")
		return
	}

	rawID, _ := tok.Extra("id_token").(string)
	if rawID == "" {
		s.redirectAuthError(w, r, "no_id_token")
		return
	}
	idToken, err := s.Auth.VerifyIDToken(ctx, rawID)
	if err != nil {
		slog.Error("verify id_token", "err", err)
		s.redirectAuthError(w, r, "invalid_id_token")
		return
	}
	if idToken.Nonce != flow.Nonce {
		s.redirectAuthError(w, r, "nonce_mismatch")
		return
	}

	// Merge claims from id_token and userinfo to extract identity + groups.
	claims := map[string]any{}
	_ = idToken.Claims(&claims)

	sub := idToken.Subject
	email, _ := claims["email"].(string)
	name := firstString(claims, "name", "preferred_username", "given_name")

	if ui, err := s.Auth.UserInfo(ctx, tok); err == nil {
		uiClaims := map[string]any{}
		if err := ui.Claims(&uiClaims); err == nil {
			for k, v := range uiClaims {
				if _, exists := claims[k]; !exists {
					claims[k] = v
				}
			}
		}
		if ui.Subject != "" {
			sub = ui.Subject
		}
		if ui.Email != "" {
			email = ui.Email
		}
		if name == "" {
			name = firstString(uiClaims, "name", "preferred_username", "given_name")
		}
	}

	groups := auth.ExtractGroups(claims)
	isAdmin := auth.IsAdmin(groups, s.Cfg.OIDCAdminGroup)

	user, err := s.Store.UpsertUserByOIDC(ctx, sub, email, name, groups, isAdmin)
	if err != nil {
		slog.Error("upsert user", "err", err)
		s.redirectAuthError(w, r, "user_store_failed")
		return
	}

	// Encrypt tokens at rest.
	accessEnc, e1 := s.Cipher.Encrypt(tok.AccessToken)
	refreshEnc, e2 := s.Cipher.Encrypt(tok.RefreshToken)
	idEnc, e3 := s.Cipher.Encrypt(rawID)
	if e1 != nil || e2 != nil || e3 != nil {
		s.redirectAuthError(w, r, "encrypt_failed")
		return
	}

	sid, err := auth.RandomID(32)
	if err != nil {
		s.redirectAuthError(w, r, "session_failed")
		return
	}
	var expiresAt *time.Time
	if !tok.Expiry.IsZero() {
		e := tok.Expiry
		expiresAt = &e
	}
	if err := s.Store.CreateSession(ctx, storeSession(sid, user.ID, accessEnc, refreshEnc, idEnc, expiresAt)); err != nil {
		slog.Error("create session", "err", err)
		s.redirectAuthError(w, r, "session_failed")
		return
	}

	s.setSessionCookie(w, sid)
	http.Redirect(w, r, safeReturnTo(flow.ReturnTo), http.StatusFound)
}

// handleLogout deletes the session and clears the cookie.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" && s.Store != nil {
		if err := s.Store.DeleteSession(r.Context(), c.Value); err != nil {
			slog.Warn("delete session on logout", "err", err)
		}
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// redirectAuthError sends the browser back to the app base with an error marker.
func (s *Server) redirectAuthError(w http.ResponseWriter, r *http.Request, reason string) {
	slog.Warn("auth callback rejected", "reason", reason, "path", r.URL.Path)
	http.Redirect(w, r, "/?auth_error="+reason, http.StatusFound)
}

// --- /auth/session (Huma JSON endpoint) -----------------------------------

type sessionUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Sub   string `json:"sub"`
}

type SessionOutput struct {
	Body struct {
		Authenticated bool              `json:"authenticated"`
		User          *sessionUser      `json:"user,omitempty"`
		Groups        []string          `json:"groups,omitempty"`
		Permissions   *auth.Permissions `json:"permissions,omitempty"`
	}
}

func registerSession(api huma.API, s *Server) {
	huma.Register(api, huma.Operation{
		OperationID: "auth-session",
		Method:      http.MethodGet,
		Path:        "/auth/session",
		Summary:     "Current session",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, _ *struct{}) (*SessionOutput, error) {
		out := &SessionOutput{}
		u, ok := userFromContext(ctx)
		if !ok {
			out.Body.Authenticated = false
			return out, huma.Error401Unauthorized("not authenticated")
		}
		out.Body.Authenticated = true
		out.Body.User = &sessionUser{ID: u.ID, Email: u.Email, Name: u.Name, Sub: u.OIDCSub}
		out.Body.Groups = u.Groups
		perms := auth.PermissionsFor(u.Groups, s.Cfg.OIDCAdminGroup)
		out.Body.Permissions = &perms
		return out, nil
	})
}

// firstString returns the first non-empty string value among the given keys.
func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// safeReturnTo guards against open-redirects: only same-origin relative paths
// are honored; anything that carries a scheme/host, a backslash escape, or a
// protocol-relative prefix falls back to "/". The value is reconstructed from
// the parsed path+query rather than echoed raw.
func safeReturnTo(rt string) string {
	if rt == "" || !strings.HasPrefix(rt, "/") {
		return "/"
	}
	// Reject protocol-relative ("//host") and backslash variants ("/\", "\")
	// that some browsers normalize to an absolute URL.
	if strings.HasPrefix(rt, "//") || strings.HasPrefix(rt, "/\\") || strings.HasPrefix(rt, "\\") {
		return "/"
	}
	u, err := url.Parse(rt)
	if err != nil || u.Scheme != "" || u.Host != "" || u.Opaque != "" || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return "/"
	}
	// Never bounce back into the auth endpoints — that turns a transient login
	// hiccup into an endless redirect loop with an ever-nesting return_to.
	lp := strings.ToLower(u.Path)
	if strings.HasPrefix(lp, "/auth/") || strings.HasPrefix(lp, "/api/auth/") {
		return "/"
	}
	out := u.Path
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		out += "#" + u.EscapedFragment()
	}
	return out
}

// cookieValue parses a named cookie out of a raw Cookie header.
func cookieValue(cookieHeader, name string) string {
	header := http.Header{}
	header.Set("Cookie", cookieHeader)
	req := http.Request{Header: header}
	if c, err := req.Cookie(name); err == nil {
		return c.Value
	}
	return ""
}
