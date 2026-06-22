package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ncl/mooring-api/internal/domain"
)

// ErrDrumOccupied is returned when moving a line onto a drum that already holds one.
var ErrDrumOccupied = errors.New("drum already holds a line")

// ErrInvalidMoveTarget is returned when a move request names neither or both
// destinations, or a destination that belongs to a different vessel.
var ErrInvalidMoveTarget = errors.New("invalid move target")

// validateMoveRequest enforces exactly-one destination (drum XOR storage). Pure
// and DB-free so it is unit-testable without a database.
func validateMoveRequest(toDrumID, toStorageID string) error {
	if (toDrumID == "") == (toStorageID == "") {
		return ErrInvalidMoveTarget // both empty or both set
	}
	return nil
}

type LineFilter struct {
	LineTypeID string
	Condition  string
	Placement  string // installed | spare | "" (all)
	Q          string
	Limit      int
	Offset     int
}

// rowSelect lists the columns shared by the register row and the full record.
const rowSelect = `
SELECT ml.id, ml.name, ml.serial_number, COALESCE(ml.tag_number,''),
       COALESCE(ml.certificate_number,''), ml.lifecycle_status,
       p.product_name, m.name, lt.name,
       COALESCE(ml.current_condition_status,''), COALESCE(ml.current_side,''),
       CASE WHEN ml.current_drum_id IS NOT NULL THEN w.label || ' · D' || d.idx
            WHEN ml.current_storage_id IS NOT NULL THEN st.label
            ELSE '—' END AS location_label,
       (ml.current_drum_id IS NOT NULL) AS installed,
       ml.installation_date, ml.manufacture_date,
       ml.current_drum_id, ml.current_storage_id
FROM mooring_line ml
JOIN product p ON p.id = ml.product_id
JOIN maker m ON m.id = p.maker_id
JOIN line_type lt ON lt.id = p.line_type_id
LEFT JOIN drum d ON d.id = ml.current_drum_id
LEFT JOIN winch_location w ON w.id = d.winch_id
LEFT JOIN storage_location st ON st.id = ml.current_storage_id`

func scanRow(row pgx.Row, now time.Time) (LineRow, error) {
	var r LineRow
	var inst, mfg *time.Time
	err := row.Scan(&r.ID, &r.Name, &r.SerialNumber, &r.TagNumber, &r.CertificateNumber,
		&r.LifecycleStatus, &r.ProductName, &r.MakerName, &r.LineTypeName,
		&r.CurrentConditionStatus, &r.CurrentSide, &r.LocationLabel, &r.Installed, &inst, &mfg,
		&r.CurrentDrumID, &r.CurrentStorageID)
	if err != nil {
		return r, err
	}
	r.InstallAgeDays = domain.AgeDays(inst, now)
	r.BuildAgeDays = domain.AgeDays(mfg, now)
	r.NextInspectionDue = domain.NextInspectionDue(inst)
	return r, nil
}

func (s *Store) ListLines(ctx context.Context, vesselID string, f LineFilter) ([]LineRow, int, error) {
	now := time.Now().UTC()
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 200
	}
	args := []any{vesselID, nullUUID(f.LineTypeID), nullStr(f.Condition), f.Placement, f.Q, f.Limit, f.Offset}
	where := `
WHERE ml.vessel_id = $1 AND ml.parent_line_id IS NULL
  AND ($2::uuid IS NULL OR p.line_type_id = $2)
  AND ($3::text IS NULL OR ml.current_condition_status = $3)
  AND ($4 = '' OR ($4 = 'installed' AND ml.current_drum_id IS NOT NULL)
              OR ($4 = 'spare' AND ml.current_drum_id IS NULL))
  AND ($5 = '' OR ml.name ILIKE '%'||$5||'%' OR ml.serial_number ILIKE '%'||$5||'%'
              OR p.product_name ILIKE '%'||$5||'%'
              OR COALESCE(w.label, st.label, '') ILIKE '%'||$5||'%')`

	rows, err := s.Pool.Query(ctx, rowSelect+where+` ORDER BY ml.name LIMIT $6 OFFSET $7`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []LineRow{}
	for rows.Next() {
		r, err := scanRow(rows, now)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// total count for the same filter (without pagination)
	var total int
	countArgs := []any{vesselID, nullUUID(f.LineTypeID), nullStr(f.Condition), f.Placement, f.Q}
	countSQL := `
SELECT count(*) FROM mooring_line ml
JOIN product p ON p.id = ml.product_id
LEFT JOIN drum d ON d.id = ml.current_drum_id
LEFT JOIN winch_location w ON w.id = d.winch_id
LEFT JOIN storage_location st ON st.id = ml.current_storage_id` + where
	if err := s.Pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// GetLine returns the full record including side ages, location ids and components.
func (s *Store) GetLine(ctx context.Context, id string) (Line, error) {
	now := time.Now().UTC()
	l, err := s.scanFullLine(ctx, id, now)
	if err != nil {
		return Line{}, err
	}
	// components
	cr, err := s.Pool.Query(ctx, `SELECT id FROM mooring_line WHERE parent_line_id=$1 ORDER BY name`, id)
	if err != nil {
		return l, err
	}
	var compIDs []string
	for cr.Next() {
		var cid string
		if err := cr.Scan(&cid); err != nil {
			cr.Close()
			return l, err
		}
		compIDs = append(compIDs, cid)
	}
	cr.Close()
	for _, cid := range compIDs {
		c, err := s.scanFullLine(ctx, cid, now)
		if err != nil {
			return l, err
		}
		l.Components = append(l.Components, c)
	}
	return l, nil
}

func (s *Store) scanFullLine(ctx context.Context, id string, now time.Time) (Line, error) {
	var l Line
	var inst, mfg, saCD, sbCD *time.Time
	var saAcc, sbAcc int
	err := s.Pool.QueryRow(ctx, `
SELECT ml.id, ml.name, ml.serial_number, COALESCE(ml.tag_number,''),
       COALESCE(ml.certificate_number,''), ml.lifecycle_status,
       p.product_name, m.name, lt.name,
       COALESCE(ml.current_condition_status,''), COALESCE(ml.current_side,''),
       CASE WHEN ml.current_drum_id IS NOT NULL THEN w.label || ' · D' || d.idx
            WHEN ml.current_storage_id IS NOT NULL THEN st.label
            ELSE '—' END,
       (ml.current_drum_id IS NOT NULL),
       ml.installation_date, ml.manufacture_date,
       ml.vessel_id, ml.product_id, COALESCE(p.construction_type,''), p.swl, p.break_load, ml.length,
       ml.can_be_turned, COALESCE(ml.certificate_ref,''),
       ml.side_a_change_date, ml.side_a_accumulated_age_days, COALESCE(ml.side_a_condition,''),
       ml.side_b_change_date, ml.side_b_accumulated_age_days, COALESCE(ml.side_b_condition,''),
       ml.current_drum_id, ml.current_storage_id, ml.parent_line_id
FROM mooring_line ml
JOIN product p ON p.id = ml.product_id
JOIN maker m ON m.id = p.maker_id
JOIN line_type lt ON lt.id = p.line_type_id
LEFT JOIN drum d ON d.id = ml.current_drum_id
LEFT JOIN winch_location w ON w.id = d.winch_id
LEFT JOIN storage_location st ON st.id = ml.current_storage_id
WHERE ml.id=$1`, id).Scan(
		&l.ID, &l.Name, &l.SerialNumber, &l.TagNumber, &l.CertificateNumber, &l.LifecycleStatus,
		&l.ProductName, &l.MakerName, &l.LineTypeName,
		&l.CurrentConditionStatus, &l.CurrentSide, &l.LocationLabel, &l.Installed,
		&inst, &mfg, &l.VesselID, &l.ProductID, &l.ConstructionType, &l.SWL, &l.BreakLoad, &l.Length,
		&l.CanBeTurned, &l.CertificateRef,
		&saCD, &saAcc, &l.SideACondition,
		&sbCD, &sbAcc, &l.SideBCondition,
		&l.CurrentDrumID, &l.CurrentStorageID, &l.ParentLineID)
	if err != nil {
		return l, err
	}
	l.InstallationDate, l.ManufactureDate = inst, mfg
	l.InstallAgeDays = domain.AgeDays(inst, now)
	l.BuildAgeDays = domain.AgeDays(mfg, now)
	l.NextInspectionDue = domain.NextInspectionDue(inst)
	l.SideAChangeDate, l.SideBChangeDate = saCD, sbCD
	l.SideAAgeDays = domain.LiveSideAge(saAcc, saCD, l.CurrentSide == "A", now)
	l.SideBAgeDays = domain.LiveSideAge(sbAcc, sbCD, l.CurrentSide == "B", now)
	active := l.SideAAgeDays
	if l.CurrentSide == "B" {
		active = l.SideBAgeDays
	}
	l.TurnDue = domain.TurnDue(l.CanBeTurned, active)
	l.Components = []Line{}
	return l, nil
}

// CreateLine registers a line or component. can_be_turned defaults from the product
// unless the line is non-reversible by type.
func (s *Store) CreateLine(ctx context.Context, vesselID string, in NewLineInput) (Line, error) {
	id := newID()
	if in.LifecycleStatus == "" {
		in.LifecycleStatus = "active"
	}
	side := in.CurrentSide
	if side == "" {
		side = "n/a"
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Line{}, err
	}
	defer tx.Rollback(ctx)

	// can_be_turned defaults from product
	var canTurn bool
	if err := tx.QueryRow(ctx, `SELECT can_be_turned FROM product WHERE id=$1`, in.ProductID).Scan(&canTurn); err != nil {
		return Line{}, err
	}

	_, err = tx.Exec(ctx, `
INSERT INTO mooring_line
  (id, vessel_id, product_id, name, tag_number, certificate_number, serial_number,
   lifecycle_status, length, manufacture_date, installation_date, can_be_turned,
   current_side, parent_line_id)
VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),$7,$8,$9,$10,$11,$12,$13,$14)`,
		id, vesselID, in.ProductID, in.Name, in.TagNumber, in.CertificateNumber, in.SerialNumber,
		in.LifecycleStatus, in.Length, in.ManufactureDate, in.InstallationDate, canTurn,
		side, in.ParentLineID)
	if err != nil {
		return Line{}, mapPgError(err)
	}
	if err := writeOutbox(ctx, tx, vesselID, "mooring_line", id, "line.registered",
		map[string]any{"id": id, "serial": in.SerialNumber}); err != nil {
		return Line{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Line{}, err
	}
	return s.GetLine(ctx, id)
}

// MoveLine relocates a line to a drum or to storage (exactly one). Enforces
// one-line-per-drum via the partial unique index.
func (s *Store) MoveLine(ctx context.Context, id, toDrumID, toStorageID string) (Line, error) {
	if err := validateMoveRequest(toDrumID, toStorageID); err != nil {
		return Line{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Line{}, err
	}
	defer tx.Rollback(ctx)

	var vesselID string
	if err := tx.QueryRow(ctx, `SELECT vessel_id FROM mooring_line WHERE id=$1`, id).Scan(&vesselID); err != nil {
		return Line{}, err
	}

	// The destination must belong to the same vessel as the line. Onboard is a
	// single vessel so this is unreachable there; it guards the shore/fleet case.
	var targetVessel string
	if toDrumID != "" {
		err = tx.QueryRow(ctx, `SELECT w.vessel_id FROM drum d JOIN winch_location w ON w.id=d.winch_id WHERE d.id=$1`, toDrumID).Scan(&targetVessel)
	} else {
		err = tx.QueryRow(ctx, `SELECT vessel_id FROM storage_location WHERE id=$1`, toStorageID).Scan(&targetVessel)
	}
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && targetVessel != vesselID) {
		return Line{}, ErrInvalidMoveTarget
	}
	if err != nil {
		return Line{}, err
	}

	_, err = tx.Exec(ctx, `
UPDATE mooring_line
SET current_drum_id = $2, current_storage_id = $3, updated_at = now(),
    lifecycle_status = CASE WHEN $2::uuid IS NOT NULL THEN 'active'
                            WHEN $3::uuid IS NOT NULL THEN 'spare'
                            ELSE lifecycle_status END
WHERE id = $1`, id, nullUUID(toDrumID), nullUUID(toStorageID))
	if err != nil {
		return Line{}, mapPgError(err)
	}
	if err := writeOutbox(ctx, tx, vesselID, "mooring_line", id, "line.moved",
		map[string]any{"id": id, "drumId": toDrumID, "storageId": toStorageID}); err != nil {
		return Line{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Line{}, err
	}
	return s.GetLine(ctx, id)
}

func (s *Store) AddComponent(ctx context.Context, parentID, vesselID string, in NewLineInput) (Line, error) {
	in.ParentLineID = &parentID
	return s.CreateLine(ctx, vesselID, in)
}

// mapPgError translates unique-violation on the drum index into ErrDrumOccupied.
func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "mooring_line_drum_key" {
			return ErrDrumOccupied
		}
	}
	return err
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
