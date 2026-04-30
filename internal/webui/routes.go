package webui

import (
	"net/http"
)

// registerRoutes sets up all HTTP routes on the provided mux.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Static assets
	mux.Handle("GET /static/", staticHandler())

	// Dashboard pages (HTML)
	mux.HandleFunc("GET /{$}", s.handleRoot)
	mux.HandleFunc("GET /runs", s.handleRunsPage)
	mux.HandleFunc("GET /runs/{id}", s.handleRunDetailPage)

	mux.HandleFunc("GET /work", s.handleWorkBoard)
	mux.HandleFunc("GET /work/{forge}/{owner}/{repo}/{number}", s.handleWorkItemDetail)

	mux.HandleFunc("GET /pipelines", s.handlePipelinesPage)
	mux.HandleFunc("GET /pipelines/{name}", s.handlePipelineDetailPage)
	mux.HandleFunc("GET /personas", s.handlePersonasPage)
	mux.HandleFunc("GET /personas/{name}", s.handlePersonaDetailPage)
	mux.HandleFunc("GET /contracts", s.handleContractsPage)
	mux.HandleFunc("GET /contracts/{name}", s.handleContractDetailPage)
	mux.HandleFunc("GET /skills", s.handleSkillsPage)
	mux.HandleFunc("GET /skills/{name}", s.handleSkillDetailPage)
	mux.HandleFunc("GET /compose", s.handleComposePage)
	mux.HandleFunc("GET /issues", s.handleIssuesPage)
	mux.HandleFunc("GET /issues/{number}", s.handleIssueDetailPage)
	mux.HandleFunc("GET /prs", s.handlePRsPage)
	mux.HandleFunc("GET /prs/{number}", s.handlePRDetailPage)
	mux.HandleFunc("GET /health", s.handleHealthPage)
	mux.HandleFunc("GET /onboard", s.handleOnboardPage)
	mux.HandleFunc("GET /onboard/{id}", s.handleOnboardPage)
	mux.HandleFunc("GET /onboard/{id}/stream", s.handleOnboardStream)
	mux.HandleFunc("POST /onboard/{id}/answer", s.handleOnboardAnswer)
	// Retros is optional — registered via build tag. See features_retros.go.
	mux.HandleFunc("GET /compare", s.handleComparePage)
	// Analytics and Webhooks are optional — registered via build tags.
	// See features_analytics.go and features_webhooks.go.
	mux.HandleFunc("GET /admin", s.handleAdminPage)

	// Evolution proposal approval gate (#1613)
	mux.HandleFunc("GET /proposals", s.handleProposalsPage)
	mux.HandleFunc("GET /proposals/{id}", s.handleProposalDetailPage)
	mux.HandleFunc("POST /proposals/{id}/approve", s.handleProposalApprove)
	mux.HandleFunc("POST /proposals/{id}/reject", s.handleProposalReject)
	mux.HandleFunc("POST /pipelines/{pipelineName}/rollback", s.handleProposalRollback)

	// API endpoints (JSON)
	mux.HandleFunc("GET /api/runs", s.handleAPIRuns)
	mux.HandleFunc("GET /api/runs/export", s.handleExportRuns)
	mux.HandleFunc("POST /api/runs", s.handleSubmitRun)
	mux.HandleFunc("GET /api/pipelines", s.handleAPIPipelines)
	mux.HandleFunc("GET /api/adapters", s.handleAPIAdapters)
	mux.HandleFunc("GET /api/models", s.handleAPIModels)
	mux.HandleFunc("GET /api/runs/{id}", s.handleAPIRunDetail)
	mux.HandleFunc("GET /api/runs/{id}/children", s.handleAPIRunChildren)
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
	mux.HandleFunc("POST /api/skills/{name}/run-install", s.handleAPISkillRunInstall)
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
	mux.HandleFunc("POST /work/{forge}/{owner}/{repo}/{number}/dispatch", s.handleWorkDispatch)
	mux.HandleFunc("GET /api/prs", s.handleAPIPRs)
	mux.HandleFunc("POST /api/prs/{number}/review", s.handlePRReview)
	mux.HandleFunc("POST /api/prs/start", s.handleAPIStartFromPR)
	mux.HandleFunc("POST /api/cache/refresh", s.handleAPICacheRefresh)
	mux.HandleFunc("GET /api/health", s.handleAPIHealth)
	mux.HandleFunc("GET /api/attention", s.handleAttentionSummary)
	mux.HandleFunc("GET /api/attention/events", s.handleAttentionSSE)
	mux.HandleFunc("GET /api/compare", s.handleAPICompare)
	// Retrospective API — see features_retros.go

	// Admin API
	mux.HandleFunc("GET /api/admin/config", s.handleAPIAdminConfig)
	mux.HandleFunc("GET /api/admin/credentials", s.handleAPIAdminCredentials)
	mux.HandleFunc("POST /api/admin/emergency-stop", s.handleAPIEmergencyStop)
	mux.HandleFunc("POST /api/admin/pipelines/{name}/disable", s.handleDisablePipeline)
	mux.HandleFunc("POST /api/admin/pipelines/{name}/enable", s.handleEnablePipeline)
	mux.HandleFunc("GET /api/admin/audit", s.handleAPIAdminAudit)

	// Optional feature routes (analytics, webhooks, etc.)
	for _, fn := range s.assets.features.routeFns {
		fn(s, mux)
	}

	// Catch-all 404 for unmatched routes
	mux.HandleFunc("/", s.handleNotFound)
}

// handleNotFound renders the 404 page for unmatched routes.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	tmpl := s.assets.templates["templates/notfound.html"]
	if tmpl != nil {
		_ = tmpl.ExecuteTemplate(w, "templates/layout.html", nil)
	}
}
