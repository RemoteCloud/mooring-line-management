package httpapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/store"
)

func registerVessel(api huma.API, s *Server) {
	tag := []string{"vessel"}

	huma.Register(api, huma.Operation{
		OperationID: "list-vessels", Method: http.MethodGet, Path: "/vessels",
		Summary: "List vessels", Tags: tag,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.Vessel }, error) {
		v, err := s.Store.ListVessels(ctx)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body []store.Vessel }{Body: v}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-vessel", Method: http.MethodPost, Path: "/vessels",
		Summary: "Create vessel", Tags: tag, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		Body struct {
			Name string `json:"name" minLength:"1"`
			IMO  string `json:"imo,omitempty"`
		}
	}) (*struct{ Body store.Vessel }, error) {
		v, err := s.Store.CreateVessel(ctx, in.Body.Name, in.Body.IMO)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Vessel }{Body: v}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-vessel", Method: http.MethodGet, Path: "/vessels/{vessel_id}",
		Summary: "Get vessel", Tags: tag,
	}, func(ctx context.Context, in *struct {
		VesselID string `path:"vessel_id" format:"uuid"`
	}) (*struct{ Body store.Vessel }, error) {
		v, err := s.Store.GetVessel(ctx, s.vessel(in.VesselID))
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Vessel }{Body: v}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-layout", Method: http.MethodGet, Path: "/vessels/{vessel_id}/layout",
		Summary: "Get deck layout (winches, drums, storage with worst-case status)", Tags: tag,
	}, func(ctx context.Context, in *struct {
		VesselID string `path:"vessel_id" format:"uuid"`
	}) (*struct{ Body store.Layout }, error) {
		l, err := s.Store.GetLayout(ctx, s.vessel(in.VesselID))
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Layout }{Body: l}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "save-layout", Method: http.MethodPut, Path: "/vessels/{vessel_id}/layout",
		Summary: "Replace deck layout (staged edit save)", Tags: tag,
	}, func(ctx context.Context, in *struct {
		VesselID string `path:"vessel_id" format:"uuid"`
		Body     struct {
			Winches []winchBody   `json:"winches"`
			Storage []storageBody `json:"storage"`
		}
	}) (*struct{ Body store.Layout }, error) {
		vid := s.vessel(in.VesselID)
		input := store.SaveLayoutInput{}
		for _, w := range in.Body.Winches {
			input.Winches = append(input.Winches, store.WinchInput{
				ID: w.ID, Label: w.Label, Station: w.Station, X: w.X, Y: w.Y,
				Orientation: w.Orientation, DrumCount: w.DrumCount,
			})
		}
		for _, st := range in.Body.Storage {
			input.Storage = append(input.Storage, store.StorageInput{
				ID: st.ID, Label: st.Label, Station: st.Station, X: st.X, Y: st.Y,
			})
		}
		if err := s.Store.SaveLayout(ctx, vid, input); err != nil {
			return nil, mapErr(err)
		}
		l, err := s.Store.GetLayout(ctx, vid)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Layout }{Body: l}, nil
	})
}

type winchBody struct {
	ID          string  `json:"id,omitempty"`
	Label       string  `json:"label" minLength:"1"`
	Station     string  `json:"station" enum:"fwd,aft"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Orientation int     `json:"orientation" enum:"0,45,-45,90,-90"`
	DrumCount   int     `json:"drum_count" minimum:"1" maximum:"6"`
}

type storageBody struct {
	ID      string  `json:"id,omitempty"`
	Label   string  `json:"label" minLength:"1"`
	Station string  `json:"station" enum:"fwd,aft"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
}
