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

	cfg := huma.DefaultConfig("Mooring Line Management API", "0.2.0")
	cfg.Info.Description = apiDescription
	cfg.Info.Contact = &huma.Contact{
		Name: "Mooring Line Management",
		URL:  "https://mooring.operationcentric.com",
	}
	// Document the base path so generated clients and the "Authorize" flow target /api.
	cfg.Servers = []*huma.Server{
		{URL: "/api", Description: "Same-origin (the web app proxies here)"},
		{URL: "https://mooring.operationcentric.com/api", Description: "Hosted deployment"},
	}
	// Document the API-key auth so the spec is self-describing and the docs UIs show
	// an Authorize button. Keys are sent as `Authorization: Bearer <key>` (also accepted
	// as `X-API-Key: <key>` or `?api_key=<key>` for downloads).
	if cfg.Components == nil {
		cfg.Components = &huma.Components{}
	}
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"apiKey": {
			Type:        "http",
			Scheme:      "bearer",
			Description: "API key issued per user. Send as `Authorization: Bearer <key>`. `X-API-Key: <key>` and `?api_key=<key>` are also accepted.",
		},
	}
	cfg.Security = []map[string][]string{{"apiKey": {}}}
	// Disable Huma's unpkg-CDN docs page; we serve a self-hosted, offline-safe one
	// (registerDocs) instead. /openapi.json|yaml are still served by Huma.
	cfg.DocsPath = ""
	api := humago.New(mux, cfg)

	registerDocs(mux)
	registerSwagger(mux)

	// Cross-cutting middleware, in order: authenticate the API key first (so scope and
	// handlers see the authenticated user), then enforce deployment scope.
	api.UseMiddleware(AuthMiddleware(api, s))
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
	registerWebhooks(api, s)
	registerUsers(api, s)

	return mux, api
}

// apiDescription is the OpenAPI info.description: an at-a-glance guide for external
// developers integrating against the API.
const apiDescription = `Track mooring lines across their life: where each line sits on deck, its
identity and certification, side-tracking and turning, inspections, condition rollups, and
document/photo evidence.

**Deployment scope.** The same API runs in two scopes, set by configuration, not by the client:
- *onboard* — one vessel; the deployment ignores any other vessel id you send and serves its own.
- *shore* — the whole fleet; vessel id selects which vessel.

**Authentication.** Every endpoint except ` + "`/health`" + ` requires an API key. Send it as
` + "`Authorization: Bearer <key>`" + ` (preferred), ` + "`X-API-Key: <key>`" + `, or ` + "`?api_key=<key>`" + `
(handy for file downloads). Keys are issued per user.

**Condition.** Line/winch/storage condition is one of ` + "`Good`" + `, ` + "`Monitor`" + `, or ` + "`Action`" + `.

**Integrating an inspection provider.**
1. Push assessments with ` + "`POST /inspections/ingest`" + ` — look the line up by ` + "`serialNumber`" + `,
   and pass an ` + "`externalId`" + ` so retries are idempotent.
2. Comment on an existing inspection with ` + "`POST /inspections/{id}/feedback`" + `.
3. Subscribe to changes with webhooks (` + "`/webhooks`" + `); deliveries are HMAC-SHA256 signed.

**Errors.** Failures return a JSON problem document (` + "`status`" + `, ` + "`title`" + `, ` + "`detail`" + `, and a
field-level ` + "`errors`" + ` array on validation problems).`
