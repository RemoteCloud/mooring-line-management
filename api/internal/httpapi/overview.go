package httpapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/store"
)

// registerOverview wires the dashboard vessel-overview endpoint. DEFINE-only:
// the orchestrator adds the registerOverview(api, s) call to NewAPI.
func registerOverview(api huma.API, s *Server) {
	huma.Register(api, huma.Operation{
		OperationID: "over-get", Method: http.MethodGet,
		Path:    "/vessels/{vessel_id}/overview",
		Summary: "Vessel dashboard overview (KPIs, condition, attention, trend)",
		Tags:    []string{"dashboard"},
	}, func(ctx context.Context, in *struct {
		VesselID string `path:"vessel_id" format:"uuid"`
	}) (*struct{ Body store.Overview }, error) {
		ov, err := s.Store.Overview(ctx, s.vessel(in.VesselID))
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Overview }{Body: ov}, nil
	})
}
