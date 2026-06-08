package httpapi

import (
	"embed"
	"net/http"
	"strings"
)

// Stoplight Elements assets are vendored (pinned @9.0.15) and embedded so the API
// reference renders fully offline — onboard tablets have no internet, and a CDN
// dependency would leave /docs blank. Served at /docs/assets/* and referenced by
// the /docs page below (no crossorigin/SRI, since they're same-origin local files).
//
//go:embed docsassets/web-components.min.js docsassets/styles.min.css
var docsAssets embed.FS

// docsPage is the Stoplight Elements host page. Same markup Huma emits, but pointing
// at the embedded assets and the local spec instead of unpkg.
const docsPage = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="referrer" content="no-referrer">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Mooring Line Management API Reference</title>
    <link rel="stylesheet" href="/docs/assets/styles.min.css">
    <script src="/docs/assets/web-components.min.js"></script>
  </head>
  <body style="height: 100vh;">
    <elements-api
      apiDescriptionUrl="/openapi.yaml"
      router="hash"
      layout="sidebar"
      tryItCredentialsPolicy="same-origin"
    ></elements-api>
  </body>
</html>`

// registerDocs wires the self-hosted API reference onto the raw mux (Huma's own
// CDN-based docs route is disabled via cfg.DocsPath="" in NewAPI).
func registerDocs(mux *http.ServeMux) {
	mux.HandleFunc("/docs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(docsPage))
	})

	mux.HandleFunc("/docs/assets/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/docs/assets/")
		data, err := docsAssets.ReadFile("docsassets/" + name)
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
