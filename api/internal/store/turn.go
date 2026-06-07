package store

import (
	"context"
	"errors"
	"time"
)

// ErrNotTurnable is returned when a line cannot be turned (not turnable by type,
// or not currently installed on a definite side).
var ErrNotTurnable = errors.New("line is not turnable")

// TurnLine flips an installed line to its other side. It freezes the accumulated
// age of the side leaving service, sets the change date of the side entering
// service to today, records a turn_event, and emits a line.turned outbox event.
func (s *Store) TurnLine(ctx context.Context, lineID, note string) (Line, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Line{}, err
	}
	defer tx.Rollback(ctx)

	var (
		vesselID    string
		canBeTurned bool
		currentSide string
		saChange    *time.Time
		sbChange    *time.Time
		install     *time.Time
	)
	err = tx.QueryRow(ctx, `
SELECT vessel_id, can_be_turned, COALESCE(current_side,''),
       side_a_change_date, side_b_change_date, installation_date
FROM mooring_line WHERE id=$1`, lineID).Scan(
		&vesselID, &canBeTurned, &currentSide, &saChange, &sbChange, &install)
	if err != nil {
		return Line{}, err
	}

	if !canBeTurned || (currentSide != "A" && currentSide != "B") {
		return Line{}, ErrNotTurnable
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	// elapsed returns clamped days between the side's reference date and today.
	elapsed := func(ref *time.Time) int {
		base := ref
		if base == nil {
			base = install
		}
		if base == nil {
			return 0
		}
		d := int(today.Sub(base.UTC().Truncate(24*time.Hour)).Hours() / 24)
		if d < 0 {
			d = 0
		}
		return d
	}

	var newSide string
	if currentSide == "A" {
		// Freeze side A, side B enters service today.
		newSide = "B"
		_, err = tx.Exec(ctx, `
UPDATE mooring_line
SET current_side = $2,
    side_a_accumulated_age_days = side_a_accumulated_age_days + $3,
    side_b_change_date = $4,
    updated_at = now()
WHERE id = $1`, lineID, newSide, elapsed(saChange), today)
	} else {
		// Freeze side B, side A enters service today.
		newSide = "A"
		_, err = tx.Exec(ctx, `
UPDATE mooring_line
SET current_side = $2,
    side_b_accumulated_age_days = side_b_accumulated_age_days + $3,
    side_a_change_date = $4,
    updated_at = now()
WHERE id = $1`, lineID, newSide, elapsed(sbChange), today)
	}
	if err != nil {
		return Line{}, mapPgError(err)
	}

	_, err = tx.Exec(ctx, `
INSERT INTO turn_event
  (id, line_id, event_type, event_date, side_after, note, origin)
VALUES ($1,$2,'turn',$3,$4,$5,'onboard')`,
		newID(), lineID, today, newSide, nullStr(note))
	if err != nil {
		return Line{}, mapPgError(err)
	}

	if err := writeOutbox(ctx, tx, vesselID, "mooring_line", lineID, "line.turned",
		map[string]any{"id": lineID, "side": newSide}); err != nil {
		return Line{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Line{}, err
	}
	return s.GetLine(ctx, lineID)
}
