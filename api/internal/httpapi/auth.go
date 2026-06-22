package httpapi

import (
	"context"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/store"
)

// ctxKey is an unexported type for context keys set by this package, to avoid
// collisions with keys from other packages.
type ctxKey int

const userKey ctxKey = iota

// AuthedUser returns the authenticated user stashed by AuthMiddleware, if any.
func AuthedUser(ctx context.Context) (store.AuthUser, bool) {
	u, ok := ctx.Value(userKey).(store.AuthUser)
	return u, ok
}

// requireAdmin returns a 403 unless the request is authenticated as an admin.
func requireAdmin(ctx context.Context) error {
	u, ok := AuthedUser(ctx)
	if !ok || u.Role != "admin" {
		return huma.Error403Forbidden("admin only")
	}
	return nil
}

// AuthMiddleware enforces API-key auth (basic, temporary). Every request must carry a
// valid key except the liveness probe. The resolved user is stashed in the context for
// downstream middleware and handlers. Registered BEFORE ScopeMiddleware so scope and
// handlers see the authenticated user.
func AuthMiddleware(api huma.API, s *Server) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if ctx.Operation().Path == "/health" {
			next(ctx)
			return
		}
		key := apiKeyFromRequest(ctx)
		if key == "" {
			huma.WriteErr(api, ctx, 401, "missing API key")
			return
		}
		if s.Store == nil {
			huma.WriteErr(api, ctx, 401, "auth unavailable")
			return
		}
		user, err := s.Store.AuthenticateAPIKey(ctx.Context(), key)
		if err != nil {
			huma.WriteErr(api, ctx, 401, "invalid API key")
			return
		}
		next(huma.WithValue(ctx, userKey, user))
	}
}

// apiKeyFromRequest reads the key from Authorization: Bearer, X-API-Key, or the
// ?api_key= query param (the last lets <a href> downloads authenticate a browser nav).
func apiKeyFromRequest(ctx huma.Context) string {
	if h := ctx.Header("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if h := ctx.Header("X-API-Key"); h != "" {
		return h
	}
	return ctx.Query("api_key")
}
