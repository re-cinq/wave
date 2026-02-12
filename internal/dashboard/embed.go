package dashboard

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static
var staticFS embed.FS

// staticHandler returns an http.Handler that serves the embedded static files.
// It serves index.html for any path that doesn't match a static file (SPA routing).
func staticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("failed to create sub filesystem: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Try to serve the file directly
		if path != "/" && !strings.HasSuffix(path, "/") {
			// Check if the file exists in the embedded FS
			if f, err := sub.Open(strings.TrimPrefix(path, "/")); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// For SPA routing: serve index.html for any unmatched path
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
