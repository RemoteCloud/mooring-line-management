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
		OperationID: "list-lines", Method: http.MethodGet, Path: "/vessels/{vessel_id}/lines",
		Summary: "List mooring lines (filterable, searchable, paginated)", Tags: tag,
	}, func(ctx context.Context, in *struct {
		VesselID   string `path:"vessel_id" format:"uuid"`
		LineTypeID string `query:"line_type_id"`
		Condition  string `query:"condition" enum:"Good,Monitor,Action"`
		Placement  string `query:"placement" enum:"installed,spare"`
		Q          string `query:"q"`
		Limit      int    `query:"limit"`
		Offset     int    `query:"offset"`
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
		OperationID: "register-line", Method: http.MethodPost, Path: "/vessels/{vessel_id}/lines",
		Summary: "Register a mooring line (supports lifecycle_status=ordered)", Tags: tag,
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		VesselID string `path:"vessel_id" format:"uuid"`
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
		Summary: "Get full rope record", Tags: tag,
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
		Summary: "Move a line to a drum or to storage", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body struct {
			ToDrumID    string `json:"to_drum_id,omitempty"`
			ToStorageID string `json:"to_storage_id,omitempty"`
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
		Summary: "Add a component (tail/lashing) to a line", Tags: tag,
		DefaultStatus: http.StatusCreated,
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
	ProductID         string   `json:"product_id" format:"uuid"`
	Name              string   `json:"name" minLength:"1"`
	SerialNumber      string   `json:"serial_number" minLength:"1"`
	TagNumber         string   `json:"tag_number,omitempty"`
	CertificateNumber string   `json:"certificate_number,omitempty"`
	LifecycleStatus   string   `json:"lifecycle_status,omitempty" enum:"ordered,active,spare,retired"`
	Length            *float64 `json:"length,omitempty"`
	ManufactureDate   string   `json:"manufacture_date,omitempty" format:"date"`
	InstallationDate  string   `json:"installation_date,omitempty" format:"date"`
	CurrentSide       string   `json:"current_side,omitempty" enum:"A,B,n/a"`
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
