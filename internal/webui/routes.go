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
	mux.HandleFunc("GET /contracts", s.handleContractsPage)
	mux.HandleFunc("GET /contracts/{name}", s.handleContractDetailPage)
	mux.HandleFunc("GET /skills", s.handleSkillsPage)
	mux.HandleFunc("GET /compose", s.handleComposePage)
	mux.HandleFunc("GET /issues", s.handleIssuesPage)
	mux.HandleFunc("GET /issues/{number}", s.handleIssueDetailPage)
	mux.HandleFunc("GET /prs", s.handlePRsPage)
	mux.HandleFunc("GET /prs/{number}", s.handlePRDetailPage)
	mux.HandleFunc("GET /health", s.handleHealthPage)
	// Ontology is optional — registered via build tag. See features_ontology.go.
	mux.HandleFunc("GET /retros", s.handleRetrosPage)
	mux.HandleFunc("GET /compare", s.handleComparePage)
	// Analytics and Webhooks are optional — registered via build tags.
	// See features_analytics.go and features_webhooks.go.
	mux.HandleFunc("GET /admin", s.handleAdminPage)

	// API endpoints (JSON)
	mux.HandleFunc("GET /api/runs", s.handleAPIRuns)
	mux.HandleFunc("GET /api/runs/export", s.handleExportRuns)
	mux.HandleFunc("POST /api/runs", s.handleSubmitRun)
	mux.HandleFunc("GET /api/pipelines", s.handleAPIPipelines)
	mux.HandleFunc("GET /api/adapters", s.handleAPIAdapters)
	mux.HandleFunc("GET /api/runs/{id}", s.handleAPIRunDetail)
	mux.HandleFunc("GET /api/runs/{id}/logs", s.handleRunLogs)
	mux.HandleFunc("POST /api/pipelines/{name}/start", s.handleStartPipeline)
	mux.HandleFunc("POST /api/runs/{id}/cancel", s.handleCancelRun)
	mux.HandleFunc("POST /api/runs/{id}/retry", s.handleRetryRun)
	mux.HandleFunc("POST /api/runs/{id}/resume", s.handleResumeRun)
	mux.HandleFunc("POST /api/runs/{id}/fork", s.handleForkRun)
	mux.HandleFunc("POST /api/runs/{id}/rewind", s.handleRewindRun)
	mux.HandleFunc("GET /api/runs/{id}/fork-points", s.handleForkPoints)
	mux.HandleFunc("POST /api/runs/{id}/gates/{step}/approve", s.handleGateApprove)
	mux.HandleFunc("GET /api/personas", s.handleAPIPersonas)
	mux.HandleFunc("GET /api/contracts", s.handleAPIContracts)
	mux.HandleFunc("GET /api/skills", s.handleAPISkills)
	mux.HandleFunc("POST /api/skills/{name}/install", s.handleAPISkillInstall)
	mux.HandleFunc("GET /api/contracts/{name}", s.handleAPIContractDetail)
	mux.HandleFunc("GET /api/compose", s.handleAPICompose)
	mux.HandleFunc("GET /api/pipelines/info", s.handleAPIPipelineInfo)
	mux.HandleFunc("GET /api/pipelines/{name}", s.handleAPIPipelineDetail)
	mux.HandleFunc("GET /api/runs/{id}/artifacts/{step}/{name}", s.handleArtifact)
	mux.HandleFunc("GET /api/runs/{id}/step-events", s.handleAPIStepEvents)
	mux.HandleFunc("GET /api/runs/{id}/diff", s.handleAPIDiffSummary)
	mux.HandleFunc("GET /api/runs/{id}/diff/{path...}", s.handleAPIDiffFile)
	mux.HandleFunc("GET /api/runs/{id}/events", s.handleSSE)
	mux.HandleFunc("GET /api/issues", s.handleAPIIssues)
	mux.HandleFunc("POST /api/issues/start", s.handleAPIStartFromIssue)
	mux.HandleFunc("GET /api/prs", s.handleAPIPRs)
	mux.HandleFunc("POST /api/prs/{number}/review", s.handlePRReview)
	mux.HandleFunc("POST /api/cache/refresh", s.handleAPICacheRefresh)
	mux.HandleFunc("GET /api/health", s.handleAPIHealth)
	// api/ontology — see features_ontology.go
	mux.HandleFunc("GET /api/compare", s.handleAPICompare)
	// Retrospective API
	mux.HandleFunc("GET /api/retros", s.handleAPIRetros)
	mux.HandleFunc("GET /api/retros/{id}", s.handleAPIRetroDetail)
	mux.HandleFunc("POST /api/retros/{id}/narrate", s.handleNarrateRetro)

	// Admin API
	mux.HandleFunc("GET /api/admin/config", s.handleAPIAdminConfig)
	mux.HandleFunc("GET /api/admin/credentials", s.handleAPIAdminCredentials)
	mux.HandleFunc("POST /api/admin/emergency-stop", s.handleAPIEmergencyStop)
	mux.HandleFunc("POST /api/admin/pipelines/{name}/disable", s.handleDisablePipeline)
	mux.HandleFunc("POST /api/admin/pipelines/{name}/enable", s.handleEnablePipeline)
	mux.HandleFunc("GET /api/admin/audit", s.handleAPIAdminAudit)

	// Optional feature routes (analytics, webhooks, etc.)
	for _, fn := range featureRoutes {
		fn(s, mux)
	}

	// Catch-all 404 for unmatched routes
	mux.HandleFunc("/", s.handleNotFound)
}

// handleNotFound renders the 404 page for unmatched routes.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	tmpl := s.templates["templates/notfound.html"]
	if tmpl != nil {
		_ = tmpl.ExecuteTemplate(w, "templates/layout.html", nil)
	}
}
