package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
)

// WithStaticSPA fronts the API mux with the built web bundle so a single process serves
// both. It replicates web/nginx.conf: /api/* is stripped and routed to the API; the docs
// and OpenAPI spec are served by the API at their own paths; everything else is a static
// file with index.html SPA fallback. Used only when WEB_DIR is set (the combined image);
// in local dev WEB_DIR is empty and Vite serves the web separately.
func WithStaticSPA(apiMux http.Handler, webDir string) http.Handler {
	files := http.FileServer(http.Dir(webDir))
	mux := http.NewServeMux()

	// API under /api: strip the prefix to reach the root-registered huma routes
	// (e.g. /api/vessels -> /vessels), matching nginx `proxy_pass http://api:8080/`.
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))

	// Docs + spec are served by the API at these exact paths (no prefix strip).
	for _, p := range []string{"/docs", "/docs/", "/swagger", "/swagger/", "/openapi.json", "/openapi.yaml"} {
		mux.Handle(p, apiMux)
	}

	// Everything else: static files with SPA fallback to index.html.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rel := filepath.Clean(r.URL.Path)
		full := filepath.Join(webDir, rel)
		if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
	})

	return mux
}
