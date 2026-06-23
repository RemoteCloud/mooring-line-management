// Package seed loads demo data for Norwegian Luna so both deployments are runnable
// with realistic content (spec deliverable 6).
package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LunaVesselID is fixed so the onboard Docker deployment (VESSEL_ID env) matches.
const LunaVesselID = "00000000-0000-0000-0000-0000000000aa"

func id() string { return uuid.Must(uuid.NewV7()).String() }

func dptr(s string) *time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return &t
}

// Run seeds the database. If reset is true it wipes existing demo data first.
// Idempotent: if the Luna vessel already exists and reset is false, it does nothing.
func Run(ctx context.Context, pool *pgxpool.Pool, reset bool) error {
	if reset {
		if _, err := pool.Exec(ctx, `DELETE FROM vessel WHERE id=$1`, LunaVesselID); err != nil {
			return err
		}
		// catalogue is global; clear and reseed for a clean demo
		for _, t := range []string{"product", "maker", "line_type"} {
			if _, err := pool.Exec(ctx, "DELETE FROM "+t); err != nil {
				return err
			}
		}
	}

	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM vessel WHERE id=$1)`, LunaVesselID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil // already seeded
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// --- catalogue ---
	makerSamson, makerLankhorst := id(), id()
	exec(ctx, tx, &err, `INSERT INTO maker (id,name) VALUES ($1,'Samson'),($2,'Lankhorst')`,
		makerSamson, makerLankhorst)

	ltMain, ltTail, ltLashing := id(), id(), id()
	exec(ctx, tx, &err, `INSERT INTO line_type (id,name,description) VALUES
		($1,'Main Line','Primary mooring line'),
		($2,'Tail','Energy-absorbing tail'),
		($3,'Lashing','Securing lashing')`, ltMain, ltTail, ltLashing)

	prodMain, prodMain2, prodTail, prodLash := id(), id(), id(), id()
	exec(ctx, tx, &err, `INSERT INTO product
		(id,maker_id,line_type_id,product_name,construction_type,default_length,can_be_turned) VALUES
		($1,$5,$9,'Turbo-75 8-strand','8-strand',220,true),
		($2,$6,$9,'Blue Ocean 12-strand','12-strand',200,true),
		($3,$7,$10,'Grommet Tail','double-braid',11,false),
		($4,$8,$11,'Dyneema Lashing','single-braid',8,false)`,
		prodMain, prodMain2, prodTail, prodLash,
		makerSamson, makerLankhorst, makerSamson, makerLankhorst,
		ltMain, ltTail, ltLashing)
	if err != nil {
		return err
	}

	// --- vessel + layout ---
	exec(ctx, tx, &err, `INSERT INTO vessel (id,name,imo) VALUES ($1,'Norwegian Luna','9999991')`, LunaVesselID)

	type winchDef struct {
		label       string
		station     string
		x, y        float64
		orientation int
		drums       int
		drive       string
	}
	winches := []winchDef{
		{"FWD-1", "fwd", 0.22, 0.40, -45, 3, "electric"},
		{"FWD-2", "fwd", 0.50, 0.26, 0, 3, "hydraulic"},
		{"FWD-3", "fwd", 0.78, 0.40, 45, 3, "electric"},
		{"AFT-1", "aft", 0.22, 0.62, 45, 3, "hydraulic"},
		{"AFT-2", "aft", 0.50, 0.74, 0, 3, "electric"},
		{"AFT-3", "aft", 0.78, 0.62, -45, 3, "hydraulic"},
	}
	var drumIDs []string // flattened, in winch order
	for _, w := range winches {
		wid := id()
		exec(ctx, tx, &err, `INSERT INTO winch_location
			(id,vessel_id,label,station,x,y,orientation,drum_count,drive_type) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			wid, LunaVesselID, w.label, w.station, w.x, w.y, w.orientation, w.drums, w.drive)
		for i := 1; i <= w.drums; i++ {
			did := id()
			exec(ctx, tx, &err, `INSERT INTO drum (id,winch_id,idx) VALUES ($1,$2,$3)`, did, wid, i)
			drumIDs = append(drumIDs, did)
		}
	}
	storageFwd, storageAft := id(), id()
	exec(ctx, tx, &err, `INSERT INTO storage_location (id,vessel_id,label,station,x,y) VALUES
		($1,$3,'Store FWD','fwd',0.50,0.72),
		($2,$3,'Store AFT','aft',0.50,0.30)`, storageFwd, storageAft, LunaVesselID)
	if err != nil {
		return err
	}

	// --- lines: one per drum (18 active) ---
	conds := []string{"Good", "Good", "Good", "Monitor", "Good", "Good",
		"Good", "Action", "Good", "Monitor", "Good", "Good",
		"Good", "Good", "Monitor", "Good", "Action", "Good"}
	products := []string{prodMain, prodMain2}
	var firstLineID string
	lineIDs := make([]string, len(drumIDs))
	for i, did := range drumIDs {
		lid := id()
		lineIDs[i] = lid
		if i == 0 {
			firstLineID = lid
		}
		cond := conds[i%len(conds)]
		side := "A"
		if i%2 == 1 {
			side = "B"
		}
		prod := products[i%len(products)]
		serial := fmt.Sprintf("LUNA-ML-%03d", i+1)
		exec(ctx, tx, &err, `INSERT INTO mooring_line
			(id,vessel_id,product_id,name,tag_number,certificate_number,serial_number,
			 lifecycle_status,length,manufacture_date,installation_date,can_be_turned,
			 current_side,side_a_change_date,side_a_accumulated_age_days,side_a_condition,
			 side_b_change_date,side_b_accumulated_age_days,side_b_condition,
			 current_condition_status,certificate_ref,current_drum_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'active',$8,$9,$10,true,
			        $11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
			lid, LunaVesselID, prod, fmt.Sprintf("Mooring Line %d", i+1),
			fmt.Sprintf("TAG-%03d", i+1), fmt.Sprintf("CERT-%05d", 1000+i), serial,
			210.0, dptr("2023-03-15"), dptr("2023-06-01"),
			side, dptr("2025-12-01"), 120+i*3, cond,
			dptr("2025-06-01"), 90+i*2, "Good",
			cond, "certs/"+serial+".pdf", did)
	}
	if err != nil {
		return err
	}

	// components on the first line: a tail + a lashing
	exec(ctx, tx, &err, `INSERT INTO mooring_line
		(id,vessel_id,product_id,name,serial_number,certificate_number,lifecycle_status,
		 length,manufacture_date,can_be_turned,current_side,current_condition_status,parent_line_id)
		VALUES
		($1,$3,$4,'Tail (ML1)','LUNA-TL-001','CERT-TL-001','active',11,$6,false,'n/a','Good',$2),
		($7,$3,$5,'Lashing (ML1)','LUNA-LS-001','CERT-LS-001','active',8,$6,false,'n/a','Good',$2)`,
		id(), firstLineID, LunaVesselID, prodTail, prodLash, dptr("2023-03-15"), id())
	if err != nil {
		return err
	}

	// spares in storage (3) + 1 ordered (no location)
	spareSQL := `INSERT INTO mooring_line
		(id,vessel_id,product_id,name,serial_number,certificate_number,lifecycle_status,
		 length,manufacture_date,installation_date,can_be_turned,current_side,
		 current_condition_status,current_storage_id)
		VALUES ($1,$2,$3,$4,$5,$6,'spare',$7,$8,$8,true,'n/a',$9,$10)`
	mfg := dptr("2024-01-10")
	exec(ctx, tx, &err, spareSQL, id(), LunaVesselID, prodMain, "Spare A", "LUNA-ML-019", "CERT-02001", 210.0, mfg, "Good", storageFwd)
	exec(ctx, tx, &err, spareSQL, id(), LunaVesselID, prodMain, "Spare B", "LUNA-ML-020", "CERT-02002", 210.0, mfg, "Monitor", storageFwd)
	exec(ctx, tx, &err, spareSQL, id(), LunaVesselID, prodMain2, "Spare C", "LUNA-ML-021", "CERT-02003", 200.0, mfg, "Good", storageAft)
	if err != nil {
		return err
	}

	exec(ctx, tx, &err, `INSERT INTO mooring_line
		(id,vessel_id,product_id,name,serial_number,certificate_number,lifecycle_status,
		 length,can_be_turned,current_side,current_condition_status)
		VALUES ($1,$2,$3,'Replacement (on order)','LUNA-ML-022','CERT-02004','ordered',220,true,'n/a','Good')`,
		id(), LunaVesselID, prodMain)
	if err != nil {
		return err
	}

	// --- inspection history: spread across the last 5 months so the dashboard
	// trend, logbook and per-line Inspections tab have real data ---
	inspSQL := `INSERT INTO inspection
		(id,line_id,vessel_id,inspected_at,inspected_by,source,condition_status,notes)
		VALUES ($1,$2,$3, now() - make_interval(days => $4), $5,'manual',$6,$7)`
	inspectors := []string{"A. Berg", "M. Olsen", "K. Haugen", "T. Nilsen"}
	for i, lid := range lineIDs {
		cond := conds[i%len(conds)]
		// two historical inspections + one recent for every line
		for j, ageDays := range []int{132, 71, 12} {
			c := "Good"
			if j == 2 {
				c = cond // most-recent inspection matches the line's current status
			} else if cond == "Action" && j == 1 {
				c = "Monitor"
			}
			note := ""
			if c == "Action" {
				note = "Visible abrasion — schedule replacement."
			} else if c == "Monitor" {
				note = "Minor wear, keep under observation."
			}
			exec(ctx, tx, &err, inspSQL, id(), lid, LunaVesselID,
				ageDays-i%7, inspectors[(i+j)%len(inspectors)], c, note)
		}
	}
	if err != nil {
		return err
	}

	// --- a couple of turn events on the first two lines ---
	turnSQL := `INSERT INTO turn_event (id,line_id,event_type,event_date,side_after,note,origin)
		VALUES ($1,$2,'turn', (current_date - make_interval(days => $3))::date, $4,$5,'onboard')`
	exec(ctx, tx, &err, turnSQL, id(), lineIDs[0], 188, "B", "Routine end-for-end turn.")
	exec(ctx, tx, &err, turnSQL, id(), lineIDs[1], 64, "A", "Turned after inspection.")
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// exec runs a statement, recording the first error so callers can check once.
func exec(ctx context.Context, tx pgx.Tx, errp *error, sql string, args ...any) {
	if *errp != nil {
		return
	}
	if _, e := tx.Exec(ctx, sql, args...); e != nil {
		*errp = fmt.Errorf("seed exec: %w", e)
	}
}
