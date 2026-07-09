package store

import "context"

// bootstrapAdminID is a fixed UUID for the seeded admin so re-runs upsert the same row.
const bootstrapAdminID = "00000000-0000-0000-0000-0000000000ad"

// EnsureBootstrapAdmin guarantees a usable admin credential exists so the operator is
// never locked out. Idempotent across restarts.
//
//   - A fixed admin user (admin@local) is upserted and kept active.
//   - If bootstrapKey is set (ADMIN_BOOTSTRAP_KEY) it is registered as that admin's key
//     (insert-if-absent by hash) so the same key always works; returns "" (nothing to log).
//   - If bootstrapKey is empty and the admin has no active key, a random key is minted and
//     its plaintext returned for one-time logging; returns "" when a key already exists.
func (s *Store) EnsureBootstrapAdmin(ctx context.Context, bootstrapKey string) (string, error) {
	if _, err := s.Pool.Exec(ctx, `
INSERT INTO app_user (id, email, name, role, active)
VALUES ($1, 'admin@local', 'Bootstrap Admin', 'admin', true)
ON CONFLICT (id) DO UPDATE SET active = true, role = 'admin'`, bootstrapAdminID); err != nil {
		return "", err
	}

	if bootstrapKey != "" {
		prefix := bootstrapKey
		if len(prefix) > 12 {
			prefix = prefix[:12]
		}
		_, err := s.Pool.Exec(ctx, `
INSERT INTO api_key (id, user_id, name, key_hash, key_prefix)
VALUES ($1, $2, 'bootstrap (env)', $3, $4)
ON CONFLICT (key_hash) DO NOTHING`,
			newID(), bootstrapAdminID, hashKey(bootstrapKey), prefix)
		return "", err
	}

	var n int
	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM api_key WHERE user_id=$1 AND revoked_at IS NULL`, bootstrapAdminID).
		Scan(&n); err != nil {
		return "", err
	}
	if n > 0 {
		return "", nil
	}
	nk, err := s.CreateAPIKey(ctx, bootstrapAdminID, "bootstrap (generated)")
	if err != nil {
		return "", err
	}
	return nk.PlainKey, nil
}
