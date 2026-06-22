package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5"
)

// APIKey is the safe, listable view of an api_key row. The hash and plaintext are
// never exposed here; KeyPrefix is the only part of the key shown after creation.
type APIKey struct {
	ID         string     `json:"id"`
	UserID     string     `json:"userId"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"keyPrefix"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

// NewAPIKey carries the one-time plaintext returned at creation, in addition to the
// stored metadata. PlainKey is shown to the operator exactly once and never persisted.
type NewAPIKey struct {
	APIKey
	PlainKey string `json:"plainKey"`
}

// generateKey mints a fresh key: 32 random bytes, base64url, prefixed "mlm_".
// Returns the full plaintext, a short display prefix, and the hex sha-256 hash.
func generateKey() (full, prefix, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", "", err
	}
	full = "mlm_" + base64.RawURLEncoding.EncodeToString(buf)
	prefix = full[:12] // "mlm_" + 8 chars — display only
	hash = hashKey(full)
	return full, prefix, hash, nil
}

func hashKey(full string) string {
	sum := sha256.Sum256([]byte(full))
	return hex.EncodeToString(sum[:])
}

const apiKeyCols = `id, user_id, name, key_prefix, last_used_at, created_at, revoked_at`

func scanAPIKey(row pgx.Row) (APIKey, error) {
	var k APIKey
	if err := row.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.CreatedAt, &k.RevokedAt); err != nil {
		return APIKey{}, err
	}
	return k, nil
}

func (s *Store) ListAPIKeys(ctx context.Context, userID string) ([]APIKey, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+apiKeyCols+` FROM api_key WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// CreateAPIKey mints a key for a user and returns the plaintext exactly once.
func (s *Store) CreateAPIKey(ctx context.Context, userID, name string) (NewAPIKey, error) {
	full, prefix, hash, err := generateKey()
	if err != nil {
		return NewAPIKey{}, err
	}
	row := s.Pool.QueryRow(ctx, `
INSERT INTO api_key (id, user_id, name, key_hash, key_prefix)
VALUES ($1,$2,$3,$4,$5)
RETURNING `+apiKeyCols,
		newID(), userID, name, hash, prefix)
	k, err := scanAPIKey(row)
	if err != nil {
		return NewAPIKey{}, err
	}
	return NewAPIKey{APIKey: k, PlainKey: full}, nil
}

// RevokeAPIKey marks a key revoked. Returns pgx.ErrNoRows if it doesn't exist or is
// already revoked.
func (s *Store) RevokeAPIKey(ctx context.Context, id string) error {
	ct, err := s.Pool.Exec(ctx,
		`UPDATE api_key SET revoked_at=now() WHERE id=$1 AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// AuthenticateAPIKey resolves a presented plaintext key to its owning user. The key is
// matched by hash through the unique index (an equality lookup, so a byte-by-byte timing
// side-channel isn't reachable — constant-time compare isn't needed here). Only active,
// non-revoked keys belonging to active users authenticate. Returns pgx.ErrNoRows on miss.
func (s *Store) AuthenticateAPIKey(ctx context.Context, plain string) (AuthUser, error) {
	hash := hashKey(plain)
	var u AuthUser
	err := s.Pool.QueryRow(ctx, `
SELECT u.id, u.name, u.email, u.role, COALESCE(u.vessel_id::text,'')
FROM api_key k JOIN app_user u ON u.id = k.user_id
WHERE k.key_hash=$1 AND k.revoked_at IS NULL AND u.active = true`, hash).
		Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.VesselID)
	if err != nil {
		return AuthUser{}, err
	}
	// Best-effort last-used stamp; never fail auth on a bookkeeping write.
	_, _ = s.Pool.Exec(ctx, `UPDATE api_key SET last_used_at=now() WHERE key_hash=$1`, hash)
	return u, nil
}
