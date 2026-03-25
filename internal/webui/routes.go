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
	mux.HandleFunc("GET /contracts", s.handleContractsPage)
	mux.HandleFunc("GET /skills", s.handleSkillsPage)
	mux.HandleFunc("GET /compose", s.handleComposePage)
	mux.HandleFunc("GET /issues", s.handleIssuesPage)
	mux.HandleFunc("GET /prs", s.handlePRsPage)
	mux.HandleFunc("GET /health", s.handleHealthPage)

	// API endpoints (JSON)
	mux.HandleFunc("GET /api/runs", s.handleAPIRuns)
	mux.HandleFunc("GET /api/pipelines", s.handleAPIPipelines)
	mux.HandleFunc("GET /api/runs/{id}", s.handleAPIRunDetail)
	mux.HandleFunc("POST /api/pipelines/{name}/start", s.handleStartPipeline)
	mux.HandleFunc("POST /api/runs/{id}/cancel", s.handleCancelRun)
	mux.HandleFunc("POST /api/runs/{id}/retry", s.handleRetryRun)
	mux.HandleFunc("POST /api/runs/{id}/resume", s.handleResumeRun)
	mux.HandleFunc("GET /api/personas", s.handleAPIPersonas)
	mux.HandleFunc("GET /api/contracts", s.handleAPIContracts)
	mux.HandleFunc("GET /api/skills", s.handleAPISkills)
	mux.HandleFunc("GET /api/contracts/{name}", s.handleAPIContractDetail)
	mux.HandleFunc("GET /api/compose", s.handleAPICompose)
	mux.HandleFunc("GET /api/pipelines/info", s.handleAPIPipelineInfo)
	mux.HandleFunc("GET /api/runs/{id}/artifacts/{step}/{name}", s.handleArtifact)
	mux.HandleFunc("GET /api/runs/{id}/step-events", s.handleAPIStepEvents)
	mux.HandleFunc("GET /api/runs/{id}/diff", s.handleAPIDiffSummary)
	mux.HandleFunc("GET /api/runs/{id}/diff/{path...}", s.handleAPIDiffFile)
	mux.HandleFunc("GET /api/runs/{id}/events", s.handleSSE)
	mux.HandleFunc("GET /api/issues", s.handleAPIIssues)
	mux.HandleFunc("POST /api/issues/start", s.handleAPIStartFromIssue)
	mux.HandleFunc("GET /api/prs", s.handleAPIPRs)
	mux.HandleFunc("GET /api/health", s.handleAPIHealth)

	// Catch-all 404 for unmatched routes
	mux.HandleFunc("/", s.handleNotFound)
}

// handleNotFound renders the 404 page for unmatched routes.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	tmpl := s.templates["templates/notfound.html"]
	if tmpl != nil {
		tmpl.ExecuteTemplate(w, "templates/layout.html", nil)
	}
}
