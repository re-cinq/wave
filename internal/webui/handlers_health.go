package webui

import (
	"net/http"

	"github.com/recinq/wave/internal/health"
)

// handleHealthPage handles GET /health - serves the HTML health checks page.
func (s *Server) handleHealthPage(w http.ResponseWriter, r *http.Request) {
	healthData := s.getHealthListData()

	data := struct {
		ActivePage string
		HealthListResponse
	}{
		ActivePage:         "health",
		HealthListResponse: healthData,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/health.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIHealth handles GET /api/health - returns health check results as JSON.
func (s *Server) handleAPIHealth(w http.ResponseWriter, r *http.Request) {
	data := s.getHealthListData()
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) getHealthListData() HealthListResponse {
	provider := health.NewDefaultDataProvider(s.manifest, s.store, ".agents/pipelines")

	var checks []HealthCheckResult
	for _, name := range provider.CheckNames() {
		result := provider.RunCheck(name)
		checks = append(checks, HealthCheckResult{
			Name:    result.Name,
			Status:  healthStatusString(result.Status),
			Message: result.Message,
			Details: result.Details,
		})
	}

	return HealthListResponse{Checks: checks}
}

func healthStatusString(s health.CheckStatus) string {
	switch s {
	case health.StatusOK:
		return "ok"
	case health.StatusWarn:
		return "warn"
	case health.StatusErr:
		return "error"
	default:
		return "unknown"
	}
}
