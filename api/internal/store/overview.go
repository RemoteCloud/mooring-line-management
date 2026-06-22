package store

import (
	"context"
	"time"
)

// OverTrendPoint is a single month in the 5-month inspection trend.
type OverTrendPoint struct {
	Month       string `json:"month"`
	Inspections int    `json:"inspections"`
	Action      int    `json:"action"`
}

// OverAttentionItem is a top-level line in Monitor/Action condition.
type OverAttentionItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	SerialNumber    string `json:"serialNumber"`
	ConditionStatus string `json:"conditionStatus"`
	LocationLabel   string `json:"locationLabel"`
}

// OverRecentInspection is one of the most recent inspections for a vessel.
type OverRecentInspection struct {
	LineName        string    `json:"lineName"`
	ConditionStatus string    `json:"conditionStatus"`
	InspectedAt     time.Time `json:"inspectedAt"`
}

// Overview is the dashboard payload for a vessel.
type Overview struct {
	ActiveLines       int                    `json:"activeLines"`
	Spares            int                    `json:"spares"`
	Good              int                    `json:"good"`
	Monitor           int                    `json:"monitor"`
	Action            int                    `json:"action"`
	NeedingAttention  int                    `json:"needingAttention"`
	InspectionsDue    int                    `json:"inspectionsDue"`
	AvgInstallAgeDays int                    `json:"avgInstallAgeDays"`
	Attention         []OverAttentionItem    `json:"attention"`
	RecentInspections []OverRecentInspection `json:"recentInspections"`
	Trend             []OverTrendPoint       `json:"trend"`
}

// Overview assembles the dashboard summary for a single vessel. Only top-level
// lines (parent_line_id IS NULL) are counted throughout.
func (s *Store) Overview(ctx context.Context, vesselID string) (Overview, error) {
	var o Overview

	// Scalar KPIs in one round trip. All scoped to top-level lines.
	err := s.Pool.QueryRow(ctx, `
SELECT
  count(*) FILTER (WHERE current_drum_id IS NOT NULL)                           AS active,
  count(*) FILTER (WHERE current_drum_id IS NULL)                               AS spares,
  count(*) FILTER (WHERE current_condition_status = 'Good')                     AS good,
  count(*) FILTER (WHERE current_condition_status = 'Monitor')                  AS monitor,
  count(*) FILTER (WHERE current_condition_status = 'Action')                   AS action,
  COALESCE(ROUND(AVG(current_date - installation_date)
    FILTER (WHERE current_drum_id IS NOT NULL AND installation_date IS NOT NULL)), 0)::int AS avg_age
FROM mooring_line
WHERE vessel_id = $1 AND parent_line_id IS NULL`, vesselID).
		Scan(&o.ActiveLines, &o.Spares, &o.Good, &o.Monitor, &o.Action, &o.AvgInstallAgeDays)
	if err != nil {
		return Overview{}, err
	}
	o.NeedingAttention = o.Monitor + o.Action

	// Inspections due: top-level lines with no inspection in the last 30 days, and
	// either at least one prior inspection or an installation older than 30 days.
	err = s.Pool.QueryRow(ctx, `
SELECT count(*)
FROM mooring_line ml
WHERE ml.vessel_id = $1 AND ml.parent_line_id IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM inspection i
    WHERE i.line_id = ml.id AND i.inspected_at >= now() - interval '30 days')
  AND ( EXISTS (SELECT 1 FROM inspection i WHERE i.line_id = ml.id)
        OR ml.installation_date <= current_date - 30 )`, vesselID).
		Scan(&o.InspectionsDue)
	if err != nil {
		return Overview{}, err
	}

	// Attention: top-level Monitor/Action lines, Action first, then name.
	o.Attention = []OverAttentionItem{}
	arows, err := s.Pool.Query(ctx, `
SELECT ml.id, ml.name, ml.serial_number, ml.current_condition_status,
       CASE WHEN ml.current_drum_id IS NOT NULL THEN w.label || ' · D' || d.idx
            WHEN ml.current_storage_id IS NOT NULL THEN st.label
            ELSE '—' END AS location_label
FROM mooring_line ml
LEFT JOIN drum d ON d.id = ml.current_drum_id
LEFT JOIN winch_location w ON w.id = d.winch_id
LEFT JOIN storage_location st ON st.id = ml.current_storage_id
WHERE ml.vessel_id = $1 AND ml.parent_line_id IS NULL
  AND ml.current_condition_status IN ('Monitor','Action')
ORDER BY CASE ml.current_condition_status WHEN 'Action' THEN 0 ELSE 1 END, ml.name
LIMIT 12`, vesselID)
	if err != nil {
		return Overview{}, err
	}
	defer arows.Close()
	for arows.Next() {
		var it OverAttentionItem
		if err := arows.Scan(&it.ID, &it.Name, &it.SerialNumber, &it.ConditionStatus, &it.LocationLabel); err != nil {
			return Overview{}, err
		}
		o.Attention = append(o.Attention, it)
	}
	if err := arows.Err(); err != nil {
		return Overview{}, err
	}

	// Recent inspections: latest 8 for the vessel.
	o.RecentInspections = []OverRecentInspection{}
	rrows, err := s.Pool.Query(ctx, `
SELECT ml.name, COALESCE(i.condition_status,''), i.inspected_at
FROM inspection i
JOIN mooring_line ml ON ml.id = i.line_id
WHERE i.vessel_id = $1
ORDER BY i.inspected_at DESC
LIMIT 8`, vesselID)
	if err != nil {
		return Overview{}, err
	}
	defer rrows.Close()
	for rrows.Next() {
		var ri OverRecentInspection
		if err := rrows.Scan(&ri.LineName, &ri.ConditionStatus, &ri.InspectedAt); err != nil {
			return Overview{}, err
		}
		o.RecentInspections = append(o.RecentInspections, ri)
	}
	if err := rrows.Err(); err != nil {
		return Overview{}, err
	}

	// Trend: last 5 calendar months. Query grouped counts, then fill gaps in Go so
	// every month appears in order oldest -> newest (even zero months).
	counts := map[string]OverTrendPoint{}
	trows, err := s.Pool.Query(ctx, `
SELECT to_char(inspected_at, 'YYYY-MM') AS month,
       count(*)::int AS inspections,
       count(*) FILTER (WHERE condition_status = 'Action')::int AS action
FROM inspection
WHERE vessel_id = $1
  AND inspected_at >= date_trunc('month', now()) - interval '4 months'
GROUP BY 1`, vesselID)
	if err != nil {
		return Overview{}, err
	}
	defer trows.Close()
	for trows.Next() {
		var p OverTrendPoint
		if err := trows.Scan(&p.Month, &p.Inspections, &p.Action); err != nil {
			return Overview{}, err
		}
		counts[p.Month] = p
	}
	if err := trows.Err(); err != nil {
		return Overview{}, err
	}

	now := time.Now().UTC()
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	o.Trend = make([]OverTrendPoint, 0, 5)
	for i := 4; i >= 0; i-- {
		m := first.AddDate(0, -i, 0)
		key := m.Format("2006-01")
		if p, ok := counts[key]; ok {
			p.Month = key
			o.Trend = append(o.Trend, p)
		} else {
			o.Trend = append(o.Trend, OverTrendPoint{Month: key})
		}
	}

	return o, nil
}
