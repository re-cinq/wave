//go:build webui

package webui

import (
	"net/http"
)

// registerRoutes sets up all HTTP routes on the provided mux.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Static assets
	mux.Handle("GET /static/", staticHandler())

	// Dashboard pages (HTML)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/runs", http.StatusFound)
	})
	mux.HandleFunc("GET /runs", s.handleRunsPage)
	mux.HandleFunc("GET /runs/{id}", s.handleRunDetailPage)

	mux.HandleFunc("GET /pipelines", s.handlePipelinesPage)
	mux.HandleFunc("GET /personas", s.handlePersonasPage)

	// API endpoints (JSON)
	mux.HandleFunc("GET /api/runs", s.handleAPIRuns)
	mux.HandleFunc("GET /api/pipelines", s.handleAPIPipelines)
	mux.HandleFunc("GET /api/runs/{id}", s.handleAPIRunDetail)
	mux.HandleFunc("POST /api/pipelines/{name}/start", s.handleStartPipeline)
	mux.HandleFunc("POST /api/runs/{id}/cancel", s.handleCancelRun)
	mux.HandleFunc("POST /api/runs/{id}/retry", s.handleRetryRun)
	mux.HandleFunc("GET /api/personas", s.handleAPIPersonas)
	mux.HandleFunc("GET /api/runs/{id}/artifacts/{step}/{name}", s.handleArtifact)
	mux.HandleFunc("GET /api/runs/{id}/events", s.handleSSE)
}
