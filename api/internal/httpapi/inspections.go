package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/report"
	"github.com/ncl/mooring-api/internal/store"
)

type inspLogBody struct {
	ConditionStatus string `json:"conditionStatus" enum:"Good,Monitor,Action"`
	InspectedBy     string `json:"inspectedBy,omitempty"`
	InspectedAt     string `json:"inspectedAt,omitempty" format:"date-time"`
	Notes           string `json:"notes,omitempty"`
}

type inspIngestBody struct {
	SerialNumber    string `json:"serialNumber" minLength:"1"`
	ExternalID      string `json:"externalId,omitempty"`
	ConditionStatus string `json:"conditionStatus" enum:"Good,Monitor,Action"`
	InspectedBy     string `json:"inspectedBy,omitempty"`
	InspectedAt     string `json:"inspectedAt,omitempty" format:"date-time"`
	Notes           string `json:"notes,omitempty"`
}

// inspParseTime parses an optional RFC3339 timestamp; empty yields nil (defaults to now in the store).
func inspParseTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("inspected_at must be RFC3339 date-time")
	}
	return &t, nil
}

// registerInspections wires the inspections slice: manual logging, idempotent API ingest,
// per-line listing, vessel logbook, and CSV/PDF condition reports.
func registerInspections(api huma.API, s *Server) {
	tag := []string{"inspections"}

	huma.Register(api, huma.Operation{
		OperationID: "insp-log", Method: http.MethodPost, Path: "/lines/{id}/inspections",
		Summary: "Log a manual inspection for a line", Tags: tag,
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body inspLogBody
	}) (*struct{ Body store.Inspection }, error) {
		at, err := inspParseTime(in.Body.InspectedAt)
		if err != nil {
			return nil, err
		}
		insp, err := s.Store.LogInspection(ctx, in.ID, store.InspInput{
			ConditionStatus: in.Body.ConditionStatus,
			InspectedBy:     in.Body.InspectedBy,
			Notes:           in.Body.Notes,
			InspectedAt:     at,
		})
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Inspection }{Body: insp}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "insp-ingest", Method: http.MethodPost, Path: "/inspections/ingest",
		Summary: "Ingest an inspection from the third-party API (idempotent by external_id)", Tags: tag,
	}, func(ctx context.Context, in *struct {
		Body inspIngestBody
	}) (*struct {
		Body struct {
			Inspection store.Inspection `json:"inspection"`
			Created    bool             `json:"created"`
		}
	}, error) {
		at, err := inspParseTime(in.Body.InspectedAt)
		if err != nil {
			return nil, err
		}
		insp, created, err := s.Store.IngestInspection(ctx, in.Body.SerialNumber, in.Body.ExternalID,
			in.Body.ConditionStatus, in.Body.InspectedBy, in.Body.Notes, at)
		if err != nil {
			return nil, mapErr(err)
		}
		out := &struct {
			Body struct {
				Inspection store.Inspection `json:"inspection"`
				Created    bool             `json:"created"`
			}
		}{}
		out.Body.Inspection = insp
		out.Body.Created = created
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "insp-list", Method: http.MethodGet, Path: "/lines/{id}/inspections",
		Summary: "List inspections for a line", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{ Body []store.Inspection }, error) {
		items, err := s.Store.ListInspections(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body []store.Inspection }{Body: items}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "insp-logbook", Method: http.MethodGet, Path: "/inspections/logbook",
		Summary: "Chronological inspection logbook for a vessel", Tags: tag,
	}, func(ctx context.Context, in *struct {
		VesselID string `query:"vesselId"`
		Limit    int    `query:"limit"`
	}) (*struct{ Body []store.InspLogbookEntry }, error) {
		items, err := s.Store.Logbook(ctx, s.vessel(in.VesselID), in.Limit)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body []store.InspLogbookEntry }{Body: items}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "insp-report", Method: http.MethodGet, Path: "/reports/condition",
		Summary: "Download a condition report (CSV or PDF)", Tags: tag,
	}, func(ctx context.Context, in *struct {
		VesselID string `query:"vesselId"`
		Format   string `query:"format" enum:"pdf,csv"`
	}) (*struct {
		ContentType string `header:"Content-Type"`
		Disposition string `header:"Content-Disposition"`
		Body        []byte
	}, error) {
		vesselName, rows, err := s.Store.ConditionReport(ctx, s.vessel(in.VesselID))
		if err != nil {
			return nil, mapErr(err)
		}
		out := &struct {
			ContentType string `header:"Content-Type"`
			Disposition string `header:"Content-Disposition"`
			Body        []byte
		}{}
		if in.Format == "pdf" {
			b, err := report.PDF(vesselName, rows)
			if err != nil {
				return nil, huma.Error500InternalServerError("could not render PDF", err)
			}
			out.ContentType = "application/pdf"
			out.Disposition = `attachment; filename="condition-report.pdf"`
			out.Body = b
		} else {
			out.ContentType = "text/csv"
			out.Disposition = `attachment; filename="condition-report.csv"`
			out.Body = report.CSV(rows)
		}
		return out, nil
	})
}
