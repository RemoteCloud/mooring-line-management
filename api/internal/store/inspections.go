package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Inspection is one logged or ingested condition assessment of a mooring line.
type Inspection struct {
	ID              string    `json:"id"`
	LineID          string    `json:"lineId"`
	VesselID        string    `json:"vesselId"`
	InspectedAt     time.Time `json:"inspectedAt" doc:"When the inspection was performed"`
	InspectedBy     string    `json:"inspectedBy,omitempty" doc:"Person or system that performed it"`
	Source          string    `json:"source" doc:"How it was recorded" enum:"manual,api"`
	ExternalID      string    `json:"externalId,omitempty" doc:"Caller-supplied idempotency key (api source only)"`
	ConditionStatus string    `json:"conditionStatus" doc:"Assessed condition" enum:"Good,Monitor,Action"`
	Notes           string    `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

// InspLogbookEntry is an inspection enriched with its line's name and serial,
// for the chronological logbook view across a vessel.
type InspLogbookEntry struct {
	Inspection
	LineName     string `json:"lineName"`
	SerialNumber string `json:"serialNumber"`
}

// InspInput carries the manual-logging fields. InspectedAt is optional (defaults to now).
type InspInput struct {
	ConditionStatus string
	InspectedBy     string
	Notes           string
	InspectedAt     *time.Time
}

// InspectionFeedback is a follow-up assessment or acknowledgement attached to an
// existing inspection, typically by a third-party system that consumed it.
type InspectionFeedback struct {
	ID              string    `json:"id"`
	InspectionID    string    `json:"inspectionId"`
	ExternalID      string    `json:"externalId,omitempty" doc:"Caller-supplied idempotency key; a repeat with the same value is ignored"`
	Source          string    `json:"source" doc:"Where the feedback came from" enum:"api,manual"`
	Author          string    `json:"author,omitempty" doc:"Person or system that gave the feedback"`
	Status          string    `json:"status" doc:"Feedback disposition" enum:"acknowledged,disputed,resolved,comment"`
	ConditionStatus string    `json:"conditionStatus,omitempty" doc:"Optional condition the reviewer suggests" enum:"Good,Monitor,Action"`
	Notes           string    `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

// FeedbackInput carries the fields for creating feedback. CreatedAt is optional (defaults to now).
type FeedbackInput struct {
	ExternalID      string
	Author          string
	Status          string
	ConditionStatus string
	Notes           string
	CreatedAt       *time.Time
}

// InspReportRow is one line's latest condition for the condition report.
type InspReportRow struct {
	LineName        string
	SerialNumber    string
	ConditionStatus string
	LastInspected   string
}

const inspSelect = `
SELECT id, line_id, vessel_id, inspected_at, COALESCE(inspected_by,''), source,
       COALESCE(external_id,''), condition_status, COALESCE(notes,''), created_at
FROM inspection`

func inspScan(row pgx.Row) (Inspection, error) {
	var i Inspection
	err := row.Scan(&i.ID, &i.LineID, &i.VesselID, &i.InspectedAt, &i.InspectedBy, &i.Source,
		&i.ExternalID, &i.ConditionStatus, &i.Notes, &i.CreatedAt)
	return i, err
}

// LogInspection records a manual inspection, updates the line's current condition,
// and emits an outbox event — all in one transaction.
func (s *Store) LogInspection(ctx context.Context, lineID string, in InspInput) (Inspection, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Inspection{}, err
	}
	defer tx.Rollback(ctx)

	var vesselID string
	if err := tx.QueryRow(ctx, `SELECT vessel_id FROM mooring_line WHERE id=$1`, lineID).Scan(&vesselID); err != nil {
		return Inspection{}, err
	}

	id := newID()
	_, err = tx.Exec(ctx, `
INSERT INTO inspection (id, line_id, vessel_id, inspected_at, inspected_by, source, condition_status, notes)
VALUES ($1,$2,$3,COALESCE($4, now()),$5,'manual',$6,$7)`,
		id, lineID, vesselID, in.InspectedAt, nullStr(in.InspectedBy), in.ConditionStatus, nullStr(in.Notes))
	if err != nil {
		return Inspection{}, mapPgError(err)
	}

	if _, err := tx.Exec(ctx, `UPDATE mooring_line SET current_condition_status=$2 WHERE id=$1`,
		lineID, in.ConditionStatus); err != nil {
		return Inspection{}, err
	}

	if err := writeOutbox(ctx, tx, vesselID, "inspection", id, "inspection.logged",
		map[string]any{"id": id, "lineId": lineID, "conditionStatus": in.ConditionStatus}); err != nil {
		return Inspection{}, err
	}

	insp, err := inspScan(tx.QueryRow(ctx, inspSelect+` WHERE id=$1`, id))
	if err != nil {
		return Inspection{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Inspection{}, err
	}
	return insp, nil
}

// IngestInspection idempotently records an inspection arriving from the third-party API,
// keyed by external_id. Returns created=false (and the existing row) on a duplicate.
// A line that cannot be resolved by serial number surfaces pgx.ErrNoRows for a 404.
func (s *Store) IngestInspection(ctx context.Context, serial, externalID, conditionStatus, inspectedBy, notes string, inspectedAt *time.Time) (Inspection, bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Inspection{}, false, err
	}
	defer tx.Rollback(ctx)

	var lineID, vesselID string
	err = tx.QueryRow(ctx, `SELECT id, vessel_id FROM mooring_line WHERE serial_number=$1`, serial).
		Scan(&lineID, &vesselID)
	if err != nil {
		// pgx.ErrNoRows propagates so the handler returns 404.
		return Inspection{}, false, err
	}

	id := newID()
	var newRowID string
	err = tx.QueryRow(ctx, `
INSERT INTO inspection (id, line_id, vessel_id, inspected_at, inspected_by, source, external_id, condition_status, notes)
VALUES ($1,$2,$3,COALESCE($4, now()),$5,'api',$6,$7,$8)
ON CONFLICT (external_id) WHERE external_id IS NOT NULL DO NOTHING
RETURNING id`,
		id, lineID, vesselID, inspectedAt, nullStr(inspectedBy), nullStr(externalID), conditionStatus, nullStr(notes)).
		Scan(&newRowID)

	created := false
	switch {
	case err == nil:
		created = true
	case errors.Is(err, pgx.ErrNoRows):
		// Duplicate external_id — ON CONFLICT DO NOTHING returned no row.
		created = false
	default:
		return Inspection{}, false, mapPgError(err)
	}

	if created {
		if _, err := tx.Exec(ctx, `UPDATE mooring_line SET current_condition_status=$2 WHERE id=$1`,
			lineID, conditionStatus); err != nil {
			return Inspection{}, false, err
		}
		if err := writeOutbox(ctx, tx, vesselID, "inspection", newRowID, "inspection.logged",
			map[string]any{"id": newRowID, "lineId": lineID, "conditionStatus": conditionStatus, "source": "api"}); err != nil {
			return Inspection{}, false, err
		}
	}

	var insp Inspection
	if created {
		insp, err = inspScan(tx.QueryRow(ctx, inspSelect+` WHERE id=$1`, newRowID))
	} else {
		insp, err = inspScan(tx.QueryRow(ctx, inspSelect+` WHERE external_id=$1`, externalID))
	}
	if err != nil {
		return Inspection{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Inspection{}, false, err
	}
	return insp, created, nil
}

const feedbackSelect = `
SELECT id, inspection_id, COALESCE(external_id,''), source, COALESCE(author,''),
       status, COALESCE(condition_status,''), COALESCE(notes,''), created_at
FROM inspection_feedback`

func feedbackScan(row pgx.Row) (InspectionFeedback, error) {
	var f InspectionFeedback
	err := row.Scan(&f.ID, &f.InspectionID, &f.ExternalID, &f.Source, &f.Author,
		&f.Status, &f.ConditionStatus, &f.Notes, &f.CreatedAt)
	return f, err
}

// CreateFeedback idempotently attaches feedback to an existing inspection, keyed by
// external_id. Returns created=false (and the existing row) on a duplicate external_id.
// An inspection that does not exist surfaces pgx.ErrNoRows for a 404.
func (s *Store) CreateFeedback(ctx context.Context, inspectionID string, in FeedbackInput) (InspectionFeedback, bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return InspectionFeedback{}, false, err
	}
	defer tx.Rollback(ctx)

	var vesselID string
	if err := tx.QueryRow(ctx, `SELECT vessel_id FROM inspection WHERE id=$1`, inspectionID).Scan(&vesselID); err != nil {
		// pgx.ErrNoRows propagates so the handler returns 404.
		return InspectionFeedback{}, false, err
	}

	id := newID()
	var newRowID string
	err = tx.QueryRow(ctx, `
INSERT INTO inspection_feedback (id, inspection_id, external_id, source, author, status, condition_status, notes, created_at)
VALUES ($1,$2,$3,'api',$4,$5,$6,$7,COALESCE($8, now()))
ON CONFLICT (external_id) WHERE external_id IS NOT NULL DO NOTHING
RETURNING id`,
		id, inspectionID, nullStr(in.ExternalID), nullStr(in.Author), in.Status,
		nullStr(in.ConditionStatus), nullStr(in.Notes), in.CreatedAt).Scan(&newRowID)

	created := false
	switch {
	case err == nil:
		created = true
	case errors.Is(err, pgx.ErrNoRows):
		// Duplicate external_id — ON CONFLICT DO NOTHING returned no row.
		created = false
	default:
		return InspectionFeedback{}, false, mapPgError(err)
	}

	if created {
		if err := writeOutbox(ctx, tx, vesselID, "inspection", newRowID, "inspection.feedback",
			map[string]any{"id": newRowID, "inspectionId": inspectionID, "status": in.Status}); err != nil {
			return InspectionFeedback{}, false, err
		}
	}

	var f InspectionFeedback
	if created {
		f, err = feedbackScan(tx.QueryRow(ctx, feedbackSelect+` WHERE id=$1`, newRowID))
	} else {
		f, err = feedbackScan(tx.QueryRow(ctx, feedbackSelect+` WHERE external_id=$1`, in.ExternalID))
	}
	if err != nil {
		return InspectionFeedback{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return InspectionFeedback{}, false, err
	}
	return f, created, nil
}

// ListFeedback returns the feedback attached to an inspection, oldest first.
func (s *Store) ListFeedback(ctx context.Context, inspectionID string) ([]InspectionFeedback, error) {
	rows, err := s.Pool.Query(ctx, feedbackSelect+` WHERE inspection_id=$1 ORDER BY created_at`, inspectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []InspectionFeedback{}
	for rows.Next() {
		f, err := feedbackScan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ListInspections returns a line's inspections, most recent first.
func (s *Store) ListInspections(ctx context.Context, lineID string) ([]Inspection, error) {
	rows, err := s.Pool.Query(ctx, inspSelect+` WHERE line_id=$1 ORDER BY inspected_at DESC`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Inspection{}
	for rows.Next() {
		i, err := inspScan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// Logbook returns inspections (newest first) joined with line name and serial.
// An empty vesselID is fleet-wide (shore with no vessel filter); onboard always
// passes its configured vessel.
func (s *Store) Logbook(ctx context.Context, vesselID string, limit int) ([]InspLogbookEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.Pool.Query(ctx, `
SELECT i.id, i.line_id, i.vessel_id, i.inspected_at, COALESCE(i.inspected_by,''), i.source,
       COALESCE(i.external_id,''), i.condition_status, COALESCE(i.notes,''), i.created_at,
       ml.name, ml.serial_number
FROM inspection i
JOIN mooring_line ml ON ml.id = i.line_id
WHERE ($1::uuid IS NULL OR i.vessel_id = $1::uuid)
ORDER BY i.inspected_at DESC
LIMIT $2`, nullStr(vesselID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []InspLogbookEntry{}
	for rows.Next() {
		var e InspLogbookEntry
		if err := rows.Scan(&e.ID, &e.LineID, &e.VesselID, &e.InspectedAt, &e.InspectedBy, &e.Source,
			&e.ExternalID, &e.ConditionStatus, &e.Notes, &e.CreatedAt, &e.LineName, &e.SerialNumber); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ConditionReport returns, for each top-level line of a vessel, its latest inspection
// condition (falling back to the line's current_condition_status), for CSV/PDF export.
func (s *Store) ConditionReport(ctx context.Context, vesselID string) (string, []InspReportRow, error) {
	vesselName := "Fleet"
	if vesselID != "" {
		if err := s.Pool.QueryRow(ctx, `SELECT name FROM vessel WHERE id=$1`, vesselID).Scan(&vesselName); err != nil {
			return "", nil, err
		}
	}

	rows, err := s.Pool.Query(ctx, `
SELECT ml.name, ml.serial_number,
       COALESCE(latest.condition_status, ml.current_condition_status, '') AS condition_status,
       latest.inspected_at
FROM mooring_line ml
LEFT JOIN LATERAL (
    SELECT i.condition_status, i.inspected_at
    FROM inspection i
    WHERE i.line_id = ml.id
    ORDER BY i.inspected_at DESC
    LIMIT 1
) latest ON true
WHERE ($1::uuid IS NULL OR ml.vessel_id = $1::uuid) AND ml.parent_line_id IS NULL
ORDER BY ml.name`, nullStr(vesselID))
	if err != nil {
		return vesselName, nil, err
	}
	defer rows.Close()

	out := []InspReportRow{}
	for rows.Next() {
		var r InspReportRow
		var inspectedAt *time.Time
		if err := rows.Scan(&r.LineName, &r.SerialNumber, &r.ConditionStatus, &inspectedAt); err != nil {
			return vesselName, nil, err
		}
		if inspectedAt != nil {
			r.LastInspected = inspectedAt.Format("2006-01-02")
		} else {
			r.LastInspected = "—"
		}
		out = append(out, r)
	}
	return vesselName, out, rows.Err()
}
