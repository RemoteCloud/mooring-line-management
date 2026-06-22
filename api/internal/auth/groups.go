package auth

import (
	"strings"

	"github.com/ncl/mooring-api/internal/store"
)

// Access levels, lowest to highest. denied = no access at all (the frontend
// shows a "no access" screen); view = read-only; edit = read + write.
const (
	LevelDenied = "denied"
	LevelView   = "view"
	LevelEdit   = "edit"
)

// Permissions is the resolved access level for an authenticated user.
//
//   - Level is one of "denied" | "view" | "edit".
//   - CanRead is true when Level != denied.
//   - CanWrite is true when Level == edit.
//   - Admin is true for members of a configured admin group (OIDC_ADMIN_GROUP,
//     a set of group GUIDs); admins are always granted edit.
type Permissions struct {
	Admin    bool   `json:"admin"`
	Level    string `json:"level"`
	CanRead  bool   `json:"canRead"`
	CanWrite bool   `json:"canWrite"`
}

// groupClaimKeys are the claim names the provider may use to convey group/team/
// role membership. Maranics UserManagement exposes the user's POSITION TEAMS in
// userinfo (the reference app reads `position_team_ids`); other providers surface
// roles under different keys, so we check all of them defensively. These are the
// ids that the in-app access grants are keyed on.
var groupClaimKeys = []string{
	"position_team_ids", "positionTeamIds", "position_teams",
	"roles", "groups", "role", "wids",
}

// positionIDKeys are the claim names that may carry the user's single
// UserManagement POSITION id. Admin is derived from this id (matching the
// reference app's `position_id` -> UM_ADMIN_POSITION_IDS check), NOT from the
// team/role lists above.
var positionIDKeys = []string{"position_id", "positionId", "position"}

// ExtractPositionID returns the user's UserManagement position id from the
// decoded claims, or "" if none is present. Lowercased for comparison with the
// configured admin position ids.
func ExtractPositionID(claims map[string]any) string {
	for _, k := range positionIDKeys {
		if s, ok := claims[k].(string); ok {
			if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
				return s
			}
		}
	}
	return ""
}

// ExtractGroups pulls group/role membership from a decoded claims map. Values may
// arrive as a JSON array (["a","b"]) or a CSV/space-separated string ("a,b").
// Everything is normalized to a lowercased, de-duplicated slice.
func ExtractGroups(claims map[string]any) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	for _, key := range groupClaimKeys {
		v, ok := claims[key]
		if !ok {
			continue
		}
		switch val := v.(type) {
		case string:
			for _, part := range splitlist(val) {
				add(part)
			}
		case []string:
			for _, s := range val {
				add(s)
			}
		case []any:
			for _, item := range val {
				if s, ok := item.(string); ok {
					add(s)
				}
			}
		}
	}
	return out
}

// splitlist splits a CSV or whitespace-separated list into parts.
func splitlist(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == ';'
	})
}

// IsAdmin reports whether any of the configured admin groups is present in the
// user's groups. adminGroups is a set of group ids (GUIDs or names) configured
// via OIDC_ADMIN_GROUP — mirroring the reference app's UM_ADMIN_POSITION_IDS.
// Both sides are compared lowercased (groups are already lowercased).
func IsAdmin(groups []string, adminGroups []string) bool {
	if len(adminGroups) == 0 {
		return false
	}
	admin := make(map[string]struct{}, len(adminGroups))
	for _, a := range adminGroups {
		a = strings.ToLower(strings.TrimSpace(a))
		if a != "" {
			admin[a] = struct{}{}
		}
	}
	for _, g := range groups {
		if _, ok := admin[g]; ok {
			return true
		}
	}
	return false
}

// levelRank ranks access levels so the highest grant wins. Unknown levels rank
// as denied.
func levelRank(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case LevelEdit:
		return 2
	case LevelView:
		return 1
	default:
		return 0
	}
}

// Resolve computes the effective permissions for a user.
//
// A user is admin if any of their groups is in the configured adminGroups set
// (OIDC_ADMIN_GROUP — group GUIDs, like the reference's UM_ADMIN_POSITION_IDS).
// Admins always get edit.
//
// For non-admins the effective level is the HIGHEST level among the grants whose
// groupId matches one of the user's groups (grants: groupId -> "view"|"edit").
// No matching grant means "denied".
func Resolve(user store.User, adminGroups []string, grants map[string]string) Permissions {
	if IsAdmin(user.Groups, adminGroups) {
		return Permissions{Admin: true, Level: LevelEdit, CanRead: true, CanWrite: true}
	}

	best := LevelDenied
	bestRank := 0
	for _, g := range user.Groups {
		lvl, ok := grants[g]
		if !ok {
			continue
		}
		if r := levelRank(lvl); r > bestRank {
			bestRank = r
			best = strings.ToLower(strings.TrimSpace(lvl))
		}
	}

	return permsForLevel(false, best)
}

// permsForLevel builds a Permissions from an admin flag + level string, deriving
// CanRead/CanWrite from the level.
func permsForLevel(admin bool, level string) Permissions {
	level = strings.ToLower(strings.TrimSpace(level))
	switch level {
	case LevelEdit, LevelView:
		// ok
	default:
		level = LevelDenied
	}
	return Permissions{
		Admin:    admin,
		Level:    level,
		CanRead:  level != LevelDenied,
		CanWrite: level == LevelEdit,
	}
}
