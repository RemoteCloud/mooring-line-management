package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// User is an app_user record. Under key-only auth there is no password; password_hash
// is defaulted in the DB and never set here.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"` // admin | vessel_user | readonly
	VesselID  string    `json:"vesselId,omitempty"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
}

// AuthUser is the minimal identity the auth middleware resolves from an API key and
// stashes in the request context for handlers.
type AuthUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	VesselID string `json:"vesselId,omitempty"`
}

const userCols = `id, email, name, role, COALESCE(vessel_id::text,''), active, created_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.VesselID, &u.Active, &u.CreatedAt); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+userCols+` FROM app_user ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CreateUser inserts a user. password_hash is omitted (DB default ”); origin defaults
// to 'shore' per the 0001 schema.
func (s *Store) CreateUser(ctx context.Context, email, name, role, vesselID string) (User, error) {
	row := s.Pool.QueryRow(ctx, `
INSERT INTO app_user (id, email, name, role, vessel_id)
VALUES ($1,$2,$3,$4,$5)
RETURNING `+userCols,
		newID(), email, name, role, nullUUID(vesselID))
	return scanUser(row)
}

// UpdateUser patches active and/or role. nil pointers leave the field unchanged.
func (s *Store) UpdateUser(ctx context.Context, id string, active *bool, role *string) (User, error) {
	row := s.Pool.QueryRow(ctx, `
UPDATE app_user SET
  active = COALESCE($2, active),
  role   = COALESCE($3, role),
  updated_at = now()
WHERE id = $1
RETURNING `+userCols,
		id, active, role)
	return scanUser(row)
}
