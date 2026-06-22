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
//   - Authenticated users whose resolved level is "denied" are rejected 403 for
//     ALL non-public operations (the SPA shows a "no access" screen).
//   - "view" users may read (GET/HEAD); mutating methods are rejected 403.
//   - "edit" users and admins may do anything.
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

		user, perms, sess, ok := s.resolveSession(ctx)
		if !ok {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "authentication required")
			return
		}

		// No access level at all: reject every operation so the SPA can render a
		// dedicated "no access" screen.
		if !perms.CanRead {
			huma.WriteErr(api, ctx, http.StatusForbidden, "no access: ask an administrator to grant your group access")
			return
		}

		if isMutating(ctx.Method()) && !perms.CanWrite {
			huma.WriteErr(api, ctx, http.StatusForbidden, "read-only: write access requires an 'edit' grant")
			return
		}

		ctx = huma.WithValue(ctx, ctxUserKey, user)
		ctx = huma.WithValue(ctx, ctxPermsKey, perms)
		ctx = huma.WithValue(ctx, ctxSessionKey, sess)
		next(ctx)
	}
}

// resolveSession looks up the user + permissions for the request's session
// cookie. ok is false when there is no cookie, no matching session, or the
// store is unavailable — callers decide whether that is fatal.
func (s *Server) resolveSession(ctx huma.Context) (store.User, auth.Permissions, store.AuthSession, bool) {
	if s.Store == nil {
		return store.User{}, auth.Permissions{}, store.AuthSession{}, false
	}
	sid := cookieValue(ctx.Header("Cookie"), sessionCookieName)
	if sid == "" {
		return store.User{}, auth.Permissions{}, store.AuthSession{}, false
	}
	sess, err := s.Store.GetSession(ctx.Context(), sid)
	if err != nil {
		return store.User{}, auth.Permissions{}, store.AuthSession{}, false
	}
	user, err := s.Store.GetUser(ctx.Context(), sess.UserID)
	if err != nil {
		return store.User{}, auth.Permissions{}, store.AuthSession{}, false
	}
	// Load the group->level grants (one query) and resolve the effective
	// permissions per request, so access changes take effect without re-login.
	grants, err := s.Store.GrantsMap(ctx.Context())
	if err != nil {
		return store.User{}, auth.Permissions{}, store.AuthSession{}, false
	}
	perms := auth.Resolve(user, s.Cfg.OIDCAdminGroups, grants)
	// Best-effort liveness touch; ignore errors.
	_ = s.Store.TouchSession(ctx.Context(), sid)
	return user, perms, sess, true
}

// attachUserBestEffort resolves the session and, if present, attaches the user
// and permissions to the context without ever rejecting the request. Used for
// public paths like /auth/session that must report auth state, not enforce it.
func (s *Server) attachUserBestEffort(ctx *huma.Context) {
	user, perms, sess, ok := s.resolveSession(*ctx)
	if !ok {
		return
	}
	*ctx = huma.WithValue(*ctx, ctxUserKey, user)
	*ctx = huma.WithValue(*ctx, ctxPermsKey, perms)
	*ctx = huma.WithValue(*ctx, ctxSessionKey, sess)
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

// sessionFromContext returns the authenticated session (with encrypted tokens),
// if any. Used by handlers that call upstream APIs on the user's behalf.
func sessionFromContext(ctx context.Context) (store.AuthSession, bool) {
	sess, ok := ctx.Value(ctxSessionKey).(store.AuthSession)
	return sess, ok
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
