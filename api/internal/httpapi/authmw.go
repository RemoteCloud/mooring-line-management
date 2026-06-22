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
			next(ctx)
			return
		}

		// If the store/auth stack isn't wired (e.g. DB down at boot), fail closed.
		if s.Store == nil {
			huma.WriteErr(api, ctx, http.StatusServiceUnavailable, "authentication unavailable")
			return
		}

		sid := cookieValue(ctx.Header("Cookie"), sessionCookieName)
		if sid == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "authentication required")
			return
		}

		sess, err := s.Store.GetSession(ctx.Context(), sid)
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "invalid session")
			return
		}
		user, err := s.Store.GetUser(ctx.Context(), sess.UserID)
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "invalid session")
			return
		}

		perms := auth.PermissionsFor(user.Groups, s.Cfg.OIDCAdminGroup)

		if isMutating(ctx.Method()) && !perms.CanWrite {
			huma.WriteErr(api, ctx, http.StatusForbidden, "read-only: write access requires the admin group")
			return
		}

		// Best-effort liveness touch; ignore errors.
		_ = s.Store.TouchSession(ctx.Context(), sid)

		ctx = huma.WithValue(ctx, ctxUserKey, user)
		ctx = huma.WithValue(ctx, ctxPermsKey, perms)
		next(ctx)
	}
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
