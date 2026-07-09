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
	ConditionStatus string `json:"conditionStatus" enum:"Good,Monitor,Action" doc:"Assessed condition of the line" example:"Monitor"`
	InspectedBy     string `json:"inspectedBy,omitempty" doc:"Who performed the inspection" example:"Bosun A. Hansen"`
	InspectedAt     string `json:"inspectedAt,omitempty" format:"date-time" doc:"When the inspection happened (RFC3339); defaults to now"`
	Notes           string `json:"notes,omitempty" example:"Light surface wear, no core damage."`
}

type inspIngestBody struct {
	SerialNumber    string `json:"serialNumber" minLength:"1" doc:"Serial of the line being inspected; used to resolve it (vessel-wide unique)" example:"NCL-LUNA-0142"`
	ExternalID      string `json:"externalId,omitempty" doc:"Caller-supplied idempotency key; replaying the same value returns the existing inspection with created=false" example:"acme-insp-88231"`
	ConditionStatus string `json:"conditionStatus" enum:"Good,Monitor,Action" doc:"Assessed condition of the line" example:"Action"`
	InspectedBy     string `json:"inspectedBy,omitempty" doc:"Person or system that performed the inspection" example:"Acme Rope Survey"`
	InspectedAt     string `json:"inspectedAt,omitempty" format:"date-time" doc:"When the inspection happened (RFC3339); defaults to now"`
	Notes           string `json:"notes,omitempty"`
}

type inspFeedbackBody struct {
	Status          string `json:"status" enum:"acknowledged,disputed,resolved,comment" doc:"How the reviewer dispositioned the inspection" example:"acknowledged"`
	ExternalID      string `json:"externalId,omitempty" doc:"Caller-supplied idempotency key; replaying the same value returns the existing feedback with created=false" example:"acme-rev-5512"`
	Author          string `json:"author,omitempty" doc:"Person or system giving the feedback" example:"Acme Rope Survey"`
	ConditionStatus string `json:"conditionStatus,omitempty" enum:"Good,Monitor,Action" doc:"Optional condition the reviewer suggests for the line" example:"Monitor"`
	Notes           string `json:"notes,omitempty" example:"Sheath abrasion confirmed at 12m; schedule re-inspection."`
	CreatedAt       string `json:"createdAt,omitempty" format:"date-time" doc:"When the feedback was made; defaults to now"`
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
		Summary: "Log a manual inspection for a line",
		Description: "Records a condition assessment against a line and updates the line's current condition. " +
			"Use this for crew/manual entries; third-party systems should use `POST /inspections/ingest`. " +
			"Fires the `inspection.logged` webhook event.",
		Tags:          tag,
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnprocessableEntity, http.StatusNotFound},
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
		Summary: "Ingest an inspection from a third-party system",
		Description: "The inbound integration endpoint for external inspection providers. Resolves the line by " +
			"`serialNumber`, records the assessment, and updates the line's current condition. Idempotent by " +
			"`externalId`: replaying the same key returns the stored inspection with `created:false`. Returns " +
			"`{ inspection, created }`. Fires `inspection.logged` only on first insert.",
		Tags:   tag,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusNotFound},
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
		Summary:     "List inspections for a line",
		Description: "Returns a line's inspections, most recent first.",
		Tags:        tag,
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
		Summary: "Chronological inspection logbook for a vessel",
		Description: "Returns inspections across a vessel (newest first), each enriched with the line's name and serial. " +
			"On a shore deployment, omit `vesselId` for a fleet-wide log.",
		Tags: tag,
	}, func(ctx context.Context, in *struct {
		VesselID string `query:"vesselId" doc:"Vessel to scope to; ignored onboard (always the configured vessel)"`
		Limit    int    `query:"limit" doc:"Max entries to return (default 100)"`
	}) (*struct{ Body []store.InspLogbookEntry }, error) {
		items, err := s.Store.Logbook(ctx, s.vessel(in.VesselID), in.Limit)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body []store.InspLogbookEntry }{Body: items}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "insp-report", Method: http.MethodGet, Path: "/reports/condition",
		Summary: "Download a condition report (CSV or PDF)",
		Description: "Returns a downloadable report of every top-level line's latest condition. " +
			"`format=csv` (default) or `format=pdf`. For programmatic reads prefer the JSON list endpoints.",
		Tags: tag,
	}, func(ctx context.Context, in *struct {
		VesselID string `query:"vesselId" doc:"Vessel to scope to; ignored onboard"`
		Format   string `query:"format" enum:"pdf,csv" doc:"Output format; defaults to csv"`
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

	huma.Register(api, huma.Operation{
		OperationID: "insp-feedback", Method: http.MethodPost, Path: "/inspections/{id}/feedback",
		Summary: "Attach feedback to an inspection",
		Description: "Records a follow-up assessment or acknowledgement on an existing inspection — " +
			"the channel a third-party reviewer uses after consuming an `inspection.logged` event. " +
			"Idempotent by `externalId` (replaying the same key returns the stored feedback with `created:false`). " +
			"Feedback is an annotation: it does not change the line's condition. Fires the `inspection.feedback` webhook event.",
		Tags:   tag,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid" doc:"The inspection being commented on"`
		Body inspFeedbackBody
	}) (*struct {
		Body struct {
			Feedback store.InspectionFeedback `json:"feedback"`
			Created  bool                     `json:"created" doc:"False when this externalId was already recorded"`
		}
	}, error) {
		at, err := inspParseTime(in.Body.CreatedAt)
		if err != nil {
			return nil, err
		}
		fb, created, err := s.Store.CreateFeedback(ctx, in.ID, store.FeedbackInput{
			ExternalID:      in.Body.ExternalID,
			Author:          in.Body.Author,
			Status:          in.Body.Status,
			ConditionStatus: in.Body.ConditionStatus,
			Notes:           in.Body.Notes,
			CreatedAt:       at,
		})
		if err != nil {
			return nil, mapErr(err)
		}
		out := &struct {
			Body struct {
				Feedback store.InspectionFeedback `json:"feedback"`
				Created  bool                     `json:"created" doc:"False when this externalId was already recorded"`
			}
		}{}
		out.Body.Feedback = fb
		out.Body.Created = created
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "insp-feedback-list", Method: http.MethodGet, Path: "/inspections/{id}/feedback",
		Summary:     "List feedback on an inspection",
		Description: "Returns every feedback entry attached to an inspection, oldest first. " +
			"An unknown inspection id yields an empty list.",
		Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{ Body []store.InspectionFeedback }, error) {
		items, err := s.Store.ListFeedback(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body []store.InspectionFeedback }{Body: items}, nil
	})
}
