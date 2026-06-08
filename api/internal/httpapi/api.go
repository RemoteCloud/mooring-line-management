// Package httpapi wires the Huma API (code-first, emits OpenAPI 3.1) onto an http mux.
package httpapi

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/ncl/mooring-api/internal/config"
	"github.com/ncl/mooring-api/internal/store"
)

// Server holds shared dependencies passed to each feature's route registration.
type Server struct {
	Cfg   *config.Config
	Store *store.Store
}

// vessel resolves the effective vessel for a request: the configured vessel onboard
// (ignoring client input), or the requested vessel on shore.
func (s *Server) vessel(requested string) string {
	return EffectiveVesselID(s.Cfg, requested)
}

// NewAPI builds the mux + Huma API and registers all routes.
func NewAPI(s *Server) (http.Handler, huma.API) {
	mux := http.NewServeMux()

	cfg := huma.DefaultConfig("Mooring Line Management API", "0.1.0")
	cfg.Info.Description = "Fleet-wide mooring line management. Same API runs onboard (single vessel) and shore (fleet); scope is configuration."
	// Disable Huma's unpkg-CDN docs page; we serve a self-hosted, offline-safe one
	// (registerDocs) instead. /openapi.json|yaml are still served by Huma.
	cfg.DocsPath = ""
	api := humago.New(mux, cfg)

	registerDocs(mux)

	// Cross-cutting: scope guard runs before feature handlers (registered as middleware).
	api.UseMiddleware(ScopeMiddleware(api, s.Cfg))

	// Feature route registration (one per slice) goes here.
	registerHealth(api, s)
	registerCatalogue(api, s)
	registerVessel(api, s)
	registerLines(api, s)
	registerTurn(api, s)
	registerInspections(api, s)
	registerFiles(api, s)
	registerOverview(api, s)

	return mux, api
}
