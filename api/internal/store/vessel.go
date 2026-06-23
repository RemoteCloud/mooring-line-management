package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ncl/mooring-api/internal/domain"
)

// ErrOccupied is returned when a layout change would remove a winch/drum/storage
// that still holds a line.
var ErrOccupied = errors.New("position still holds a line")

func (s *Store) ListVessels(ctx context.Context) ([]Vessel, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name, COALESCE(imo,'') FROM vessel ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Vessel
	for rows.Next() {
		var v Vessel
		if err := rows.Scan(&v.ID, &v.Name, &v.IMO); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) GetVessel(ctx context.Context, id string) (Vessel, error) {
	var v Vessel
	err := s.Pool.QueryRow(ctx, `SELECT id, name, COALESCE(imo,'') FROM vessel WHERE id=$1`, id).
		Scan(&v.ID, &v.Name, &v.IMO)
	return v, err
}

func (s *Store) CreateVessel(ctx context.Context, name, imo string) (Vessel, error) {
	v := Vessel{ID: newID(), Name: name, IMO: imo}
	_, err := s.Pool.Exec(ctx, `INSERT INTO vessel (id, name, imo) VALUES ($1,$2,NULLIF($3,''))`,
		v.ID, name, imo)
	return v, err
}

// GetLayout assembles winches (with drums + per-drum line counts + worst status),
// and storage (with line counts + worst status) for a vessel.
func (s *Store) GetLayout(ctx context.Context, vesselID string) (Layout, error) {
	out := Layout{VesselID: vesselID, Winches: []Winch{}, Storage: []Storage{}}

	// winches
	wr, err := s.Pool.Query(ctx, `
SELECT id, label, station, x, y, orientation, drum_count, drive_type, label_auto, swl, break_load
FROM winch_location WHERE vessel_id=$1 ORDER BY station, label`, vesselID)
	if err != nil {
		return out, err
	}
	winchByID := map[string]*Winch{}
	for wr.Next() {
		var w Winch
		if err := wr.Scan(&w.ID, &w.Label, &w.Station, &w.X, &w.Y, &w.Orientation, &w.DrumCount, &w.DriveType, &w.LabelAuto, &w.SWL, &w.BreakLoad); err != nil {
			wr.Close()
			return out, err
		}
		w.Drums = []Drum{}
		out.Winches = append(out.Winches, w)
	}
	wr.Close()
	for i := range out.Winches {
		winchByID[out.Winches[i].ID] = &out.Winches[i]
	}

	// drums
	dr, err := s.Pool.Query(ctx, `
SELECT d.id, d.winch_id, d.idx
FROM drum d JOIN winch_location w ON w.id=d.winch_id
WHERE w.vessel_id=$1 ORDER BY d.idx`, vesselID)
	if err != nil {
		return out, err
	}
	drumWinch := map[string]string{}
	for dr.Next() {
		var id, winchID string
		var idx int
		if err := dr.Scan(&id, &winchID, &idx); err != nil {
			dr.Close()
			return out, err
		}
		if w := winchByID[winchID]; w != nil {
			w.Drums = append(w.Drums, Drum{ID: id, Idx: idx})
		}
		drumWinch[id] = winchID
	}
	dr.Close()

	// storage (on-map ordered first, then off-map text areas — both by label)
	sr, err := s.Pool.Query(ctx, `
SELECT id, label, COALESCE(station,''), on_map, x, y FROM storage_location
WHERE vessel_id=$1 ORDER BY on_map DESC, station, label`, vesselID)
	if err != nil {
		return out, err
	}
	storageByID := map[string]*Storage{}
	for sr.Next() {
		var st Storage
		if err := sr.Scan(&st.ID, &st.Label, &st.Station, &st.OnMap, &st.X, &st.Y); err != nil {
			sr.Close()
			return out, err
		}
		out.Storage = append(out.Storage, st)
	}
	sr.Close()
	for i := range out.Storage {
		storageByID[out.Storage[i].ID] = &out.Storage[i]
	}

	// lines with location + condition, to roll up counts and worst status
	lr, err := s.Pool.Query(ctx, `
SELECT current_drum_id, current_storage_id, COALESCE(current_condition_status,'')
FROM mooring_line
WHERE vessel_id=$1 AND parent_line_id IS NULL`, vesselID)
	if err != nil {
		return out, err
	}
	defer lr.Close()
	drumCount := map[string]int{}
	winchConds := map[string][]domain.Condition{}
	for lr.Next() {
		var drumID, storageID *string
		var cond string
		if err := lr.Scan(&drumID, &storageID, &cond); err != nil {
			return out, err
		}
		c := domain.Condition(cond)
		if drumID != nil {
			drumCount[*drumID]++
			if w := drumWinch[*drumID]; w != "" {
				winchConds[w] = append(winchConds[w], c)
			}
		}
		if storageID != nil {
			if st := storageByID[*storageID]; st != nil {
				st.LineCount++
				st.WorstStatus = string(domain.Worst(domain.Condition(st.WorstStatus), c))
			}
		}
	}
	for i := range out.Winches {
		w := &out.Winches[i]
		for j := range w.Drums {
			w.Drums[j].LineCount = drumCount[w.Drums[j].ID]
		}
		w.WorstStatus = string(domain.Worst(winchConds[w.ID]...))
	}
	return out, nil
}

// Layout save -------------------------------------------------------------

type WinchInput struct {
	ID          string
	Label       string
	Station     string
	X, Y        float64
	Orientation int
	DrumCount   int
	DriveType   string
	LabelAuto   bool
	SWL         *float64
	BreakLoad   *float64
}

type StorageInput struct {
	ID      string
	Label   string
	Station string // "" for off-map areas (stored as NULL)
	OnMap   bool
	X, Y    float64
}

type SaveLayoutInput struct {
	Winches []WinchInput
	Storage []StorageInput
}

// SaveLayout applies a staged layout edit transactionally: upsert winches/storage,
// adjust drums to match drum_count, and remove anything dropped — refusing to remove
// a position that still holds a line.
func (s *Store) SaveLayout(ctx context.Context, vesselID string, in SaveLayoutInput) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	keepWinch := map[string]bool{}
	for _, w := range in.Winches {
		if w.DriveType == "" {
			w.DriveType = "electric"
		}
		id := w.ID
		if id == "" {
			id = newID()
			if _, err := tx.Exec(ctx, `
INSERT INTO winch_location (id, vessel_id, label, station, x, y, orientation, drum_count, drive_type, label_auto, swl, break_load)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
				id, vesselID, w.Label, w.Station, w.X, w.Y, w.Orientation, w.DrumCount, w.DriveType, w.LabelAuto, w.SWL, w.BreakLoad); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(ctx, `
UPDATE winch_location SET label=$2, station=$3, x=$4, y=$5, orientation=$6, drum_count=$7, drive_type=$8, label_auto=$9, swl=$11, break_load=$12
WHERE id=$1 AND vessel_id=$10`,
				id, w.Label, w.Station, w.X, w.Y, w.Orientation, w.DrumCount, w.DriveType, w.LabelAuto, vesselID, w.SWL, w.BreakLoad); err != nil {
				return err
			}
		}
		if err := adjustDrums(ctx, tx, id, w.DrumCount); err != nil {
			return err
		}
		keepWinch[id] = true
	}

	// remove winches not kept (guard occupied)
	existing, err := idsForVessel(ctx, tx, `SELECT id FROM winch_location WHERE vessel_id=$1`, vesselID)
	if err != nil {
		return err
	}
	for _, id := range existing {
		if keepWinch[id] {
			continue
		}
		var occupied bool
		if err := tx.QueryRow(ctx, `
SELECT EXISTS(SELECT 1 FROM mooring_line ml JOIN drum d ON d.id=ml.current_drum_id WHERE d.winch_id=$1)`,
			id).Scan(&occupied); err != nil {
			return err
		}
		if occupied {
			return fmt.Errorf("winch %s: %w", id, ErrOccupied)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM winch_location WHERE id=$1`, id); err != nil {
			return err
		}
	}

	// storage upsert + removal
	keepStorage := map[string]bool{}
	for _, st := range in.Storage {
		id := st.ID
		if id == "" {
			id = newID()
			if _, err := tx.Exec(ctx, `
INSERT INTO storage_location (id, vessel_id, label, station, on_map, x, y) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				id, vesselID, st.Label, nullStr(st.Station), st.OnMap, st.X, st.Y); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(ctx, `
UPDATE storage_location SET label=$2, station=$3, on_map=$4, x=$5, y=$6 WHERE id=$1 AND vessel_id=$7`,
				id, st.Label, nullStr(st.Station), st.OnMap, st.X, st.Y, vesselID); err != nil {
				return err
			}
		}
		keepStorage[id] = true
	}
	existingStorage, err := idsForVessel(ctx, tx, `SELECT id FROM storage_location WHERE vessel_id=$1`, vesselID)
	if err != nil {
		return err
	}
	for _, id := range existingStorage {
		if keepStorage[id] {
			continue
		}
		var occupied bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM mooring_line WHERE current_storage_id=$1)`, id).Scan(&occupied); err != nil {
			return err
		}
		if occupied {
			return fmt.Errorf("storage %s: %w", id, ErrOccupied)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM storage_location WHERE id=$1`, id); err != nil {
			return err
		}
	}

	if err := writeOutbox(ctx, tx, vesselID, "layout", vesselID, "layout.updated", map[string]any{"vesselId": vesselID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// adjustDrums brings a winch's drums in line with target count: add missing idxs,
// remove surplus idxs (only when empty).
func adjustDrums(ctx context.Context, tx pgx.Tx, winchID string, target int) error {
	existing := map[int]string{}
	rows, err := tx.Query(ctx, `SELECT id, idx FROM drum WHERE winch_id=$1`, winchID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		var idx int
		if err := rows.Scan(&id, &idx); err != nil {
			rows.Close()
			return err
		}
		existing[idx] = id
	}
	rows.Close()

	for idx := 1; idx <= target; idx++ {
		if _, ok := existing[idx]; !ok {
			if _, err := tx.Exec(ctx, `INSERT INTO drum (id, winch_id, idx) VALUES ($1,$2,$3)`,
				newID(), winchID, idx); err != nil {
				return err
			}
		}
	}
	for idx, id := range existing {
		if idx <= target {
			continue
		}
		var occupied bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM mooring_line WHERE current_drum_id=$1)`, id).Scan(&occupied); err != nil {
			return err
		}
		if occupied {
			return fmt.Errorf("drum %d: %w", idx, ErrOccupied)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM drum WHERE id=$1`, id); err != nil {
			return err
		}
	}
	return nil
}

func idsForVessel(ctx context.Context, tx pgx.Tx, sql, vesselID string) ([]string, error) {
	rows, err := tx.Query(ctx, sql, vesselID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
