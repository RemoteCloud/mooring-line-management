package httpapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type HealthOutput struct {
	Body struct {
		Status   string `json:"status" example:"ok" doc:"Liveness status"`
		Scope    string `json:"scope" example:"onboard" doc:"Deployment scope"`
		VesselID string `json:"vessel_id,omitempty" doc:"Configured vessel (onboard only)"`
		DB       string `json:"db" example:"ok" doc:"Database connectivity"`
	}
}

func registerHealth(api huma.API, s *Server) {
	huma.Register(api, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check",
		Tags:        []string{"system"},
	}, func(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
		out := &HealthOutput{}
		out.Body.Status = "ok"
		out.Body.Scope = string(s.Cfg.Scope)
		out.Body.VesselID = s.Cfg.VesselID
		out.Body.DB = "ok"
		if s.Store != nil && s.Store.Pool != nil {
			if err := s.Store.Pool.Ping(ctx); err != nil {
				out.Body.DB = "down"
			}
		} else {
			out.Body.DB = "unconfigured"
		}
		return out, nil
	})
}
