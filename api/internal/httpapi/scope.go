package httpapi

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/config"
)

// ScopeMiddleware enforces deployment scope. Onboard deployments serve exactly
// one vessel: any request that names a different vessel_id (path or query) is
// rejected. Shore is fleet-wide and imposes no such restriction here.
//
// Feature handlers should still resolve the effective vessel via EffectiveVesselID
// so onboard requests that omit vessel_id default to the configured vessel.
func ScopeMiddleware(api huma.API, cfg *config.Config) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if cfg.IsOnboard() {
			if v := requestedVesselID(ctx); v != "" && v != cfg.VesselID {
				huma.WriteErr(api, ctx, 403,
					"this onboard deployment serves vessel "+cfg.VesselID+" only")
				return
			}
		}
		next(ctx)
	}
}

// requestedVesselID pulls a vessel id from the path param or query string, if present.
func requestedVesselID(ctx huma.Context) string {
	if v := ctx.Param("vessel_id"); v != "" {
		return v
	}
	return ctx.Query("vessel_id")
}

// EffectiveVesselID returns the vessel a request targets: the configured vessel
// onboard (ignoring client input), or the explicitly requested vessel on shore.
func EffectiveVesselID(cfg *config.Config, requested string) string {
	if cfg.IsOnboard() {
		return cfg.VesselID
	}
	return requested
}
