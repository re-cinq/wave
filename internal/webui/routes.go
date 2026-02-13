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
	mux.HandleFunc("GET /pipelines/{name}", s.handlePipelineDetailPage)
	mux.HandleFunc("GET /personas", s.handlePersonasPage)
	mux.HandleFunc("GET /personas/{name}", s.handlePersonaDetailPage)
	mux.HandleFunc("GET /statistics", s.handleStatisticsPage)

	// API endpoints (JSON)
	mux.HandleFunc("GET /api/runs", s.handleAPIRuns)
	mux.HandleFunc("GET /api/pipelines", s.handleAPIPipelines)
	mux.HandleFunc("GET /api/runs/{id}", s.handleAPIRunDetail)
	mux.HandleFunc("POST /api/pipelines/{name}/start", s.handleStartPipeline)
	mux.HandleFunc("POST /api/runs/{id}/cancel", s.handleCancelRun)
	mux.HandleFunc("POST /api/runs/{id}/retry", s.handleRetryRun)
	mux.HandleFunc("GET /api/personas", s.handleAPIPersonas)
	mux.HandleFunc("GET /api/pipelines/{name}", s.handleAPIPipelineDetail)
	mux.HandleFunc("GET /api/personas/{name}", s.handleAPIPersonaDetail)
	mux.HandleFunc("GET /api/statistics", s.handleAPIStatistics)
	mux.HandleFunc("GET /api/runs/{id}/artifacts/{step}/{name}", s.handleArtifact)
	mux.HandleFunc("GET /api/runs/{id}/workspace/{step}/tree", s.handleWorkspaceTree)
	mux.HandleFunc("GET /api/runs/{id}/workspace/{step}/file", s.handleWorkspaceFile)
	mux.HandleFunc("GET /api/runs/{id}/events", s.handleSSE)
}
