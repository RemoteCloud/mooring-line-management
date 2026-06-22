package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/auth"
	"github.com/ncl/mooring-api/internal/store"
)

// publicPathPrefixes are reachable without authentication. The auth endpoints
// must be public (you can't log in if login requires login) and /health is for
// liveness probes.
var publicPathPrefixes = []string{"/auth/", "/health"}

func isPublicPath(p string) bool {
	for _, pre := range publicPathPrefixes {
		if p == pre || strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// AuthMiddleware enforces app-wide authentication and group-based authorization.
//
//   - Public paths (/auth/*, /health) pass through untouched.
//   - Unauthenticated requests to any other operation are rejected 401.
//   - Authenticated non-admin users may read (GET/HEAD/OPTIONS) but mutating
//     methods are rejected 403.
//   - Admin users may do anything.
//
// On success the user + permissions are attached to the request context for
// downstream handlers. Runs AFTER ScopeMiddleware.
func AuthMiddleware(api huma.API, s *Server) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		op := ctx.Operation()
		path := ""
		if op != nil {
			path = op.Path
		}
		if isPublicPath(path) {
			// Public paths are never rejected, but we still resolve the session
			// best-effort so /auth/session can report the authenticated principal.
			// (Without this it always looks unauthenticated, and the SPA's
			// RequireAuth bounces straight back into /auth/login — a redirect loop.)
			s.attachUserBestEffort(&ctx)
			next(ctx)
			return
		}

		// If the store/auth stack isn't wired (e.g. DB down at boot), fail closed.
		if s.Store == nil {
			huma.WriteErr(api, ctx, http.StatusServiceUnavailable, "authentication unavailable")
			return
		}

		user, perms, ok := s.resolveSession(ctx)
		if !ok {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "authentication required")
			return
		}

		if isMutating(ctx.Method()) && !perms.CanWrite {
			huma.WriteErr(api, ctx, http.StatusForbidden, "read-only: write access requires the admin group")
			return
		}

		ctx = huma.WithValue(ctx, ctxUserKey, user)
		ctx = huma.WithValue(ctx, ctxPermsKey, perms)
		next(ctx)
	}
}

// resolveSession looks up the user + permissions for the request's session
// cookie. ok is false when there is no cookie, no matching session, or the
// store is unavailable — callers decide whether that is fatal.
func (s *Server) resolveSession(ctx huma.Context) (store.User, auth.Permissions, bool) {
	if s.Store == nil {
		return store.User{}, auth.Permissions{}, false
	}
	sid := cookieValue(ctx.Header("Cookie"), sessionCookieName)
	if sid == "" {
		return store.User{}, auth.Permissions{}, false
	}
	sess, err := s.Store.GetSession(ctx.Context(), sid)
	if err != nil {
		return store.User{}, auth.Permissions{}, false
	}
	user, err := s.Store.GetUser(ctx.Context(), sess.UserID)
	if err != nil {
		return store.User{}, auth.Permissions{}, false
	}
	perms := auth.PermissionsFor(user.Groups, s.Cfg.OIDCAdminGroup)
	// Best-effort liveness touch; ignore errors.
	_ = s.Store.TouchSession(ctx.Context(), sid)
	return user, perms, true
}

// attachUserBestEffort resolves the session and, if present, attaches the user
// and permissions to the context without ever rejecting the request. Used for
// public paths like /auth/session that must report auth state, not enforce it.
func (s *Server) attachUserBestEffort(ctx *huma.Context) {
	user, perms, ok := s.resolveSession(*ctx)
	if !ok {
		return
	}
	*ctx = huma.WithValue(*ctx, ctxUserKey, user)
	*ctx = huma.WithValue(*ctx, ctxPermsKey, perms)
}

// userFromContext returns the authenticated user, if any.
func userFromContext(ctx context.Context) (store.User, bool) {
	u, ok := ctx.Value(ctxUserKey).(store.User)
	return u, ok
}

// permsFromContext returns the resolved permissions, if any.
func permsFromContext(ctx context.Context) (auth.Permissions, bool) {
	p, ok := ctx.Value(ctxPermsKey).(auth.Permissions)
	return p, ok
}

// storeFlow / storeSession adapt the auth flow into store structs.
func storeFlow(state, verifier, nonce, returnTo string) store.OIDCFlow {
	return store.OIDCFlow{State: state, CodeVerifier: verifier, Nonce: nonce, ReturnTo: returnTo}
}

func storeSession(sid, userID, accessEnc, refreshEnc, idEnc string, expiresAt *time.Time) store.AuthSession {
	return store.AuthSession{
		SID: sid, UserID: userID,
		AccessTokenEnc: accessEnc, RefreshTokenEnc: refreshEnc, IDTokenEnc: idEnc,
		AccessExpiresAt: expiresAt,
	}
}
