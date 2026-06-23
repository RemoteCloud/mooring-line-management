package httpapi

import (
	"embed"
	"net/http"
	"strings"
)

// Swagger UI is vendored (pinned swagger-ui-dist@5.17.14) and embedded so it renders
// offline like the Stoplight page. Served at /swagger (UI) + /swagger/assets/*.
//
//go:embed swaggerassets/swagger-ui.css swaggerassets/swagger-ui-bundle.js swaggerassets/swagger-ui-standalone-preset.js
var swaggerAssets embed.FS

const swaggerPage = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Mooring Line Management API — Swagger UI</title>
    <link rel="stylesheet" href="/swagger/assets/swagger-ui.css">
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="/swagger/assets/swagger-ui-bundle.js"></script>
    <script src="/swagger/assets/swagger-ui-standalone-preset.js"></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: "/openapi.json",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
        layout: "StandaloneLayout",
      });
    </script>
  </body>
</html>`

// registerSwagger wires the self-hosted Swagger UI onto the raw mux.
func registerSwagger(mux *http.ServeMux) {
	mux.HandleFunc("/swagger", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerPage))
	})

	mux.HandleFunc("/swagger/assets/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/swagger/assets/")
		data, err := swaggerAssets.ReadFile("swaggerassets/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		switch {
		case strings.HasSuffix(name, ".js"):
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		case strings.HasSuffix(name, ".css"):
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		}
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(data)
	})
}
