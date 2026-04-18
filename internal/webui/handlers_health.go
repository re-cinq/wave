package webui

import (
	"net/http"

	"github.com/recinq/wave/internal/tui"
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
	provider := tui.NewDefaultHealthDataProvider(s.manifest, s.store, ".agents/pipelines")

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

func healthStatusString(s tui.HealthCheckStatus) string {
	switch s {
	case tui.HealthCheckOK:
		return "ok"
	case tui.HealthCheckWarn:
		return "warn"
	case tui.HealthCheckErr:
		return "error"
	default:
		return "unknown"
	}
}
