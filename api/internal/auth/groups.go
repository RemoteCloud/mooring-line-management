package auth

import (
	"strings"
)

// Permissions is the resolved access level for an authenticated user.
type Permissions struct {
	Admin    bool `json:"admin"`
	CanWrite bool `json:"canWrite"`
}

// groupClaimKeys are the claim names the provider may use to convey group/role
// membership. The `roles` scope is requested; different providers surface it
// under different keys, so we check all of them defensively.
var groupClaimKeys = []string{"roles", "groups", "role", "wids"}

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

// IsAdmin reports whether the admin group is present in the user's groups.
// adminGroup is matched case-insensitively (groups are already lowercased).
func IsAdmin(groups []string, adminGroup string) bool {
	target := strings.ToLower(strings.TrimSpace(adminGroup))
	if target == "" {
		target = "admin"
	}
	for _, g := range groups {
		if g == target {
			return true
		}
	}
	return false
}

// CanWrite reports whether the user may perform mutations. For now write access
// equals admin; any other authenticated user is read-only.
func CanWrite(groups []string, adminGroup string) bool {
	return IsAdmin(groups, adminGroup)
}

// PermissionsFor resolves the permission set for a user's groups.
func PermissionsFor(groups []string, adminGroup string) Permissions {
	admin := IsAdmin(groups, adminGroup)
	return Permissions{Admin: admin, CanWrite: admin}
}
