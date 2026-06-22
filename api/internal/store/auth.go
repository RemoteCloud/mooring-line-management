package store

import (
	"context"
	"encoding/json"
	"time"
)

// --- OIDC flow (short-lived auth-code state) ------------------------------

// CreateFlow persists the state for an in-flight login.
func (s *Store) CreateFlow(ctx context.Context, f OIDCFlow) error {
	returnTo := f.ReturnTo
	if returnTo == "" {
		returnTo = "/"
	}
	_, err := s.Pool.Exec(ctx, `
INSERT INTO oidc_flow (state, code_verifier, nonce, return_to)
VALUES ($1,$2,$3,$4)`,
		f.State, f.CodeVerifier, f.Nonce, returnTo)
	return err
}

// TakeFlow atomically fetches and deletes a flow row by state. Returns pgx.ErrNoRows
// if it does not exist.
func (s *Store) TakeFlow(ctx context.Context, state string) (OIDCFlow, error) {
	var f OIDCFlow
	err := s.Pool.QueryRow(ctx, `
DELETE FROM oidc_flow WHERE state = $1
RETURNING state, code_verifier, nonce, return_to, created_at`, state).
		Scan(&f.State, &f.CodeVerifier, &f.Nonce, &f.ReturnTo, &f.CreatedAt)
	return f, err
}

// DeleteExpiredFlows removes flow rows older than the cutoff (housekeeping).
func (s *Store) DeleteExpiredFlows(ctx context.Context, olderThan time.Time) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM oidc_flow WHERE created_at < $1`, olderThan)
	return err
}

// --- Users ----------------------------------------------------------------

// UpsertUserByOIDC inserts or updates an app_user keyed by oidc_sub, returning the
// stored user. Groups are persisted as a JSON array.
func (s *Store) UpsertUserByOIDC(ctx context.Context, sub, email, name, positionID, positionName string, groups []string, isAdmin bool) (User, error) {
	groupsJSON, err := json.Marshal(groups)
	if err != nil {
		return User{}, err
	}
	id := newID()
	var u User
	var groupsRaw []byte
	err = s.Pool.QueryRow(ctx, `
INSERT INTO app_user (id, oidc_sub, email, name, role, groups, position_id, position_name, is_admin, active, origin, last_login_at)
VALUES ($1, $2, $3, $4, NULL, $5, NULLIF($6,''), NULLIF($7,''), $8, true, 'oidc', now())
ON CONFLICT (oidc_sub) DO UPDATE SET
    email         = EXCLUDED.email,
    name          = EXCLUDED.name,
    groups        = EXCLUDED.groups,
    position_id   = EXCLUDED.position_id,
    position_name = EXCLUDED.position_name,
    is_admin      = EXCLUDED.is_admin,
    active        = true,
    last_login_at = now(),
    updated_at    = now()
RETURNING id, COALESCE(email,''), COALESCE(name,''), COALESCE(oidc_sub,''),
          COALESCE(groups,'[]'), COALESCE(position_id,''), COALESCE(position_name,''), is_admin, last_login_at`,
		id, sub, email, name, string(groupsJSON), positionID, positionName, isAdmin).
		Scan(&u.ID, &u.Email, &u.Name, &u.OIDCSub, &groupsRaw, &u.PositionID, &u.PositionName, &u.IsAdmin, &u.LastLoginAt)
	if err != nil {
		return User{}, err
	}
	_ = json.Unmarshal(groupsRaw, &u.Groups)
	return u, nil
}

// GetUser fetches a user by id.
func (s *Store) GetUser(ctx context.Context, id string) (User, error) {
	var u User
	var groupsRaw []byte
	err := s.Pool.QueryRow(ctx, `
SELECT id, COALESCE(email,''), COALESCE(name,''), COALESCE(oidc_sub,''),
       COALESCE(groups,'[]'), COALESCE(position_id,''), COALESCE(position_name,''), is_admin, last_login_at
FROM app_user WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &u.Name, &u.OIDCSub, &groupsRaw, &u.PositionID, &u.PositionName, &u.IsAdmin, &u.LastLoginAt)
	if err != nil {
		return User{}, err
	}
	_ = json.Unmarshal(groupsRaw, &u.Groups)
	return u, nil
}

// --- Sessions -------------------------------------------------------------

// CreateSession stores a new server-side session with encrypted tokens.
func (s *Store) CreateSession(ctx context.Context, sess AuthSession) error {
	_, err := s.Pool.Exec(ctx, `
INSERT INTO auth_session (sid, user_id, access_token_enc, refresh_token_enc, id_token_enc, access_expires_at)
VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),NULLIF($5,''),$6)`,
		sess.SID, sess.UserID, sess.AccessTokenEnc, sess.RefreshTokenEnc, sess.IDTokenEnc, sess.AccessExpiresAt)
	return err
}

// GetSession fetches a session row by sid (pgx.ErrNoRows if absent).
func (s *Store) GetSession(ctx context.Context, sid string) (AuthSession, error) {
	var a AuthSession
	var access, refresh, idtok *string
	err := s.Pool.QueryRow(ctx, `
SELECT sid, user_id, access_token_enc, refresh_token_enc, id_token_enc,
       access_expires_at, created_at, last_seen_at
FROM auth_session WHERE sid = $1`, sid).
		Scan(&a.SID, &a.UserID, &access, &refresh, &idtok, &a.AccessExpiresAt, &a.CreatedAt, &a.LastSeenAt)
	if err != nil {
		return AuthSession{}, err
	}
	if access != nil {
		a.AccessTokenEnc = *access
	}
	if refresh != nil {
		a.RefreshTokenEnc = *refresh
	}
	if idtok != nil {
		a.IDTokenEnc = *idtok
	}
	return a, nil
}

// TouchSession updates last_seen_at for liveness tracking.
func (s *Store) TouchSession(ctx context.Context, sid string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE auth_session SET last_seen_at = now() WHERE sid = $1`, sid)
	return err
}

// DeleteSession removes a session (logout).
func (s *Store) DeleteSession(ctx context.Context, sid string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM auth_session WHERE sid = $1`, sid)
	return err
}

// --- Group access control -------------------------------------------------

// ListGroupAccess returns all group access grants, ordered by group id.
func (s *Store) ListGroupAccess(ctx context.Context) ([]GroupAccess, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT group_id, level, COALESCE(label,''), COALESCE(updated_by,''), updated_at
FROM group_access
ORDER BY group_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GroupAccess
	for rows.Next() {
		var g GroupAccess
		if err := rows.Scan(&g.GroupID, &g.Level, &g.Label, &g.UpdatedBy, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// GrantsMap returns a groupId -> level map used by permission resolution.
func (s *Store) GrantsMap(ctx context.Context) (map[string]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT group_id, level FROM group_access`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, level string
		if err := rows.Scan(&id, &level); err != nil {
			return nil, err
		}
		out[id] = level
	}
	return out, rows.Err()
}

// UpsertGroupAccess inserts or updates a grant for a group id.
func (s *Store) UpsertGroupAccess(ctx context.Context, groupID, level, label, updatedBy string) (GroupAccess, error) {
	var g GroupAccess
	err := s.Pool.QueryRow(ctx, `
INSERT INTO group_access (group_id, level, label, updated_by, updated_at)
VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), now())
ON CONFLICT (group_id) DO UPDATE SET
    level      = EXCLUDED.level,
    label      = EXCLUDED.label,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
RETURNING group_id, level, COALESCE(label,''), COALESCE(updated_by,''), updated_at`,
		groupID, level, label, updatedBy).
		Scan(&g.GroupID, &g.Level, &g.Label, &g.UpdatedBy, &g.UpdatedAt)
	if err != nil {
		return GroupAccess{}, err
	}
	return g, nil
}

// DeleteGroupAccess removes a grant (reverting the group to denied).
func (s *Store) DeleteGroupAccess(ctx context.Context, groupID string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM group_access WHERE group_id = $1`, groupID)
	return err
}

// GroupsSeen returns the distinct group ids observed across all users' groups
// (a JSON array stored as text), with a count of users having each. Lets the
// admin UI discover which group GUIDs actually exist. NULL/empty groups are
// guarded against.
func (s *Store) GroupsSeen(ctx context.Context) (map[string]int, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT g, count(*)
FROM app_user, jsonb_array_elements_text(NULLIF(groups,'')::jsonb) AS g
WHERE groups IS NOT NULL AND groups <> '' AND groups <> '[]'
GROUP BY g`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}
