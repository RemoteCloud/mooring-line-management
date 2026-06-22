package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/auth"
	"github.com/ncl/mooring-api/internal/store"
)

// accessGroup is one row in the admin access-control list: a group GUID, its
// human name (resolved live from UserManagement; "" when unknown), its current
// level ("denied" when no grant exists), and how many users carry the group.
type accessGroup struct {
	GroupID   string `json:"groupId"`
	Name      string `json:"name"`
	Level     string `json:"level"`
	UserCount int    `json:"userCount"`
}

// requireAdmin returns nil if the caller is admin, else a 403. Admin status is
// resolved per request by the auth middleware and carried on the context.
func requireAdmin(ctx context.Context) error {
	perms, ok := permsFromContext(ctx)
	if !ok || !perms.Admin {
		return huma.Error403Forbidden("admin access required")
	}
	return nil
}

// registerAccess wires the admin-only group access-control endpoints. They are
// NOT public: AuthMiddleware 401s the unauthenticated; each handler additionally
// asserts the caller is admin.
func registerAccess(api huma.API, s *Server) {
	tag := []string{"access"}

	// GET /access/groups — discovered groups (from users) merged with grants.
	huma.Register(api, huma.Operation{
		OperationID: "list-access-groups",
		Method:      http.MethodGet,
		Path:        "/access/groups",
		Summary:     "List groups and their access levels (admin only)",
		Tags:        tag,
	}, func(ctx context.Context, _ *struct{}) (*struct {
		Body struct {
			Groups []accessGroup `json:"groups"`
		}
	}, error) {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}

		seen, err := s.Store.GroupsSeen(ctx)
		if err != nil {
			return nil, mapErr(err)
		}
		grants, err := s.Store.ListGroupAccess(ctx)
		if err != nil {
			return nil, mapErr(err)
		}

		// Merge: every group id from either source. Grants supply the level;
		// groups with no grant are "denied". userCount comes from GroupsSeen
		// (0 if a grant exists for a group no user currently carries).
		byID := map[string]*accessGroup{}
		for id, n := range seen {
			byID[id] = &accessGroup{GroupID: id, Level: auth.LevelDenied, UserCount: n}
		}
		for _, g := range grants {
			row, ok := byID[g.GroupID]
			if !ok {
				row = &accessGroup{GroupID: g.GroupID, UserCount: 0}
				byID[g.GroupID] = row
			}
			row.Level = g.Level
		}

		// Resolve human names live from UserManagement using the admin's own
		// access token (best-effort: on failure rows keep their GUIDs and the UI
		// shows a "Reload" affordance). Mirrors the reference app.
		for id, name := range s.positionTeamNames(ctx) {
			if row, ok := byID[strings.ToLower(id)]; ok {
				row.Name = name
			}
		}

		// Only surface real, manageable groups: those with a resolved name, or
		// those that already carry a grant. This hides un-nameable noise ids
		// (legacy role/wids GUIDs, the admin position id) the user can't act on.
		out := make([]accessGroup, 0, len(byID))
		for _, g := range byID {
			if g.Name == "" && g.Level == auth.LevelDenied {
				continue
			}
			out = append(out, *g)
		}
		// Sort named groups first, then by userCount desc, then id for stability.
		sort.Slice(out, func(i, j int) bool {
			if (out[i].Name == "") != (out[j].Name == "") {
				return out[i].Name != ""
			}
			if out[i].UserCount != out[j].UserCount {
				return out[i].UserCount > out[j].UserCount
			}
			return out[i].GroupID < out[j].GroupID
		})

		resp := &struct {
			Body struct {
				Groups []accessGroup `json:"groups"`
			}
		}{}
		resp.Body.Groups = out
		return resp, nil
	})

	// PUT /access/grants/{groupId} — upsert a grant.
	huma.Register(api, huma.Operation{
		OperationID: "put-access-grant",
		Method:      http.MethodPut,
		Path:        "/access/grants/{groupId}",
		Summary:     "Grant or update a group's access level (admin only)",
		Tags:        tag,
	}, func(ctx context.Context, in *struct {
		GroupID string `path:"groupId"`
		Body    struct {
			Level string `json:"level" enum:"view,edit"`
		}
	}) (*struct{ Body store.GroupAccess }, error) {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		level := strings.ToLower(strings.TrimSpace(in.Body.Level))
		if level != auth.LevelView && level != auth.LevelEdit {
			return nil, huma.Error400BadRequest("level must be 'view' or 'edit'")
		}
		groupID := strings.TrimSpace(in.GroupID)
		if groupID == "" {
			return nil, huma.Error400BadRequest("groupId is required")
		}

		updatedBy := ""
		if u, ok := userFromContext(ctx); ok {
			updatedBy = u.Email
		}

		g, err := s.Store.UpsertGroupAccess(ctx, groupID, level, updatedBy)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.GroupAccess }{Body: g}, nil
	})

	// DELETE /access/grants/{groupId} — remove a grant (-> denied).
	huma.Register(api, huma.Operation{
		OperationID:   "delete-access-grant",
		Method:        http.MethodDelete,
		Path:          "/access/grants/{groupId}",
		Summary:       "Revoke a group's access (admin only)",
		Tags:          tag,
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *struct {
		GroupID string `path:"groupId"`
	}) (*struct{}, error) {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if err := s.Store.DeleteGroupAccess(ctx, strings.TrimSpace(in.GroupID)); err != nil {
			return nil, mapErr(err)
		}
		return nil, nil
	})
}

// positionTeamNames resolves a lowercased groupId -> name map from the UM
// position-teams API, using the requesting admin's decrypted access token.
// Best-effort: any failure (no session, decrypt error, UM down) yields an empty
// map and a warning, so the admin UI degrades to GUIDs rather than erroring.
func (s *Server) positionTeamNames(ctx context.Context) map[string]string {
	out := map[string]string{}
	if s.Auth == nil || s.Cipher == nil {
		return out
	}
	sess, ok := sessionFromContext(ctx)
	if !ok || sess.AccessTokenEnc == "" {
		return out
	}
	token, err := s.Cipher.Decrypt(sess.AccessTokenEnc)
	if err != nil || token == "" {
		slog.Warn("access: decrypt access token for team names", "err", err)
		return out
	}
	teams, err := s.Auth.FetchPositionTeams(ctx, token)
	if err != nil {
		slog.Warn("access: fetch position teams", "err", err)
		return out
	}
	for _, t := range teams {
		if t.Name != "" {
			out[strings.ToLower(t.ID)] = t.Name
		}
	}
	return out
}
