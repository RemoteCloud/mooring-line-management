package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/store"
)

func registerLines(api huma.API, s *Server) {
	tag := []string{"lines"}

	huma.Register(api, huma.Operation{
		OperationID: "list-lines", Method: http.MethodGet, Path: "/vessels/{vesselId}/lines",
		Summary: "List mooring lines (filterable, searchable, paginated)",
		Description: "Returns the vessel's top-level mooring lines as compact rows, with `items` and a `total` " +
			"count for paging. Combine filters freely; `q` matches name/serial/tag.",
		Tags: tag,
	}, func(ctx context.Context, in *struct {
		VesselID   string `path:"vesselId" format:"uuid"`
		LineTypeID string `query:"lineTypeId" format:"uuid" doc:"Filter to one line type (see GET /line-types)"`
		Condition  string `query:"condition" enum:"Good,Monitor,Action" doc:"Filter by current condition"`
		Placement  string `query:"placement" enum:"installed,spare" doc:"installed = on a drum; spare = not currently rigged"`
		Q          string `query:"q" doc:"Free-text search over name, serial, and tag"`
		Limit      int    `query:"limit" doc:"Page size (default server value)"`
		Offset     int    `query:"offset" doc:"Rows to skip for paging"`
	}) (*struct {
		Body struct {
			Items []store.LineRow `json:"items"`
			Total int             `json:"total"`
		}
	}, error) {
		rows, total, err := s.Store.ListLines(ctx, s.vessel(in.VesselID), store.LineFilter{
			LineTypeID: in.LineTypeID, Condition: in.Condition, Placement: in.Placement,
			Q: in.Q, Limit: in.Limit, Offset: in.Offset,
		})
		if err != nil {
			return nil, mapErr(err)
		}
		out := &struct {
			Body struct {
				Items []store.LineRow `json:"items"`
				Total int             `json:"total"`
			}
		}{}
		out.Body.Items = rows
		out.Body.Total = total
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "register-line", Method: http.MethodPost, Path: "/vessels/{vesselId}/lines",
		Summary: "Register a mooring line",
		Description: "Creates a mooring line from a catalogue product. `serialNumber` must be unique on the vessel. " +
			"Use `lifecycleStatus=ordered` to pre-register a line before it arrives. Fires `line.registered`.",
		Tags:          tag,
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *struct {
		VesselID string `path:"vesselId" format:"uuid"`
		Body     lineBody
	}) (*struct{ Body store.Line }, error) {
		l, err := s.Store.CreateLine(ctx, s.vessel(in.VesselID), in.Body.toInput())
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Line }{Body: l}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-line", Method: http.MethodGet, Path: "/lines/{id}",
		Summary:     "Get the full rope record",
		Description: "Returns a line's complete record: identity, certification, specs (inherited from its product), location, side-tracking ages, and condition.",
		Tags:        tag,
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{ Body store.Line }, error) {
		l, err := s.Store.GetLine(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Line }{Body: l}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "move-line", Method: http.MethodPost, Path: "/lines/{id}/move",
		Summary: "Move a line to a drum or to storage",
		Description: "Relocates a line. Provide exactly one of `toDrumId` or `toStorageId`; sending both or neither " +
			"is rejected. Moving onto an occupied drum is a conflict. Fires `line.moved`.",
		Tags:   tag,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusConflict, http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body struct {
			ToDrumID    string `json:"toDrumId,omitempty" format:"uuid" doc:"Destination drum; mutually exclusive with toStorageId"`
			ToStorageID string `json:"toStorageId,omitempty" format:"uuid" doc:"Destination storage area; mutually exclusive with toDrumId"`
		}
	}) (*struct{ Body store.Line }, error) {
		l, err := s.Store.MoveLine(ctx, in.ID, in.Body.ToDrumID, in.Body.ToStorageID)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Line }{Body: l}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "add-component", Method: http.MethodPost, Path: "/lines/{id}/components",
		Summary: "Add a component (tail/lashing) to a line",
		Description: "Registers a sub-line (such as a tail or lashing) under a parent line, sharing its vessel. " +
			"Same body as register-line.",
		Tags:          tag,
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnprocessableEntity, http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body lineBody
	}) (*struct{ Body store.Line }, error) {
		parent, err := s.Store.GetLine(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		l, err := s.Store.AddComponent(ctx, in.ID, parent.VesselID, in.Body.toInput())
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Line }{Body: l}, nil
	})
}

type lineBody struct {
	ProductID         string   `json:"productId" format:"uuid" doc:"Catalogue product this line is built from (see GET /products)"`
	Name              string   `json:"name" minLength:"1" doc:"Operational name/position" example:"Fwd Spring 1"`
	SerialNumber      string   `json:"serialNumber" minLength:"1" doc:"Manufacturer serial; unique per vessel" example:"NCL-LUNA-0142"`
	TagNumber         string   `json:"tagNumber,omitempty" doc:"Physical asset tag, if any"`
	CertificateNumber string   `json:"certificateNumber,omitempty" doc:"Certificate reference"`
	LifecycleStatus   string   `json:"lifecycleStatus,omitempty" enum:"ordered,active,spare,retired" doc:"ordered = pre-registered before arrival; active = in service; spare = held; retired = withdrawn"`
	Length            *float64 `json:"length,omitempty" doc:"Length in metres; defaults from the product"`
	ManufactureDate   string   `json:"manufactureDate,omitempty" format:"date"`
	InstallationDate  string   `json:"installationDate,omitempty" format:"date"`
	CurrentSide       string   `json:"currentSide,omitempty" enum:"A,B,n/a" doc:"Which end is rigged for turnable lines; n/a if not turnable"`
}

func (b lineBody) toInput() store.NewLineInput {
	return store.NewLineInput{
		ProductID: b.ProductID, Name: b.Name, SerialNumber: b.SerialNumber,
		TagNumber: b.TagNumber, CertificateNumber: b.CertificateNumber,
		LifecycleStatus: b.LifecycleStatus, Length: b.Length,
		ManufactureDate:  parseDate(b.ManufactureDate),
		InstallationDate: parseDate(b.InstallationDate),
		CurrentSide:      b.CurrentSide,
	}
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}
