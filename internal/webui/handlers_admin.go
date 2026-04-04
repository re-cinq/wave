package webui

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// --- Page Handler ---

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		ActivePage string
	}{
		ActivePage: "admin",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/admin.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// --- API Endpoints ---

// adminConfigResponse is the JSON response for GET /api/admin/config.
type adminConfigResponse struct {
	Address         string            `json:"address"`
	MaxConcurrency  int               `json:"max_concurrency"`
	AdapterBinaries map[string]string `json:"adapter_binaries"`
	WorkspaceRoot   string            `json:"workspace_root"`
	AuthMode        string            `json:"auth_mode"`
}

// handleAPIAdminConfig handles GET /api/admin/config.
func (s *Server) handleAPIAdminConfig(w http.ResponseWriter, _ *http.Request) {
	addr := fmt.Sprintf("%s:%d", s.bind, s.port)

	maxConcurrency := 5
	if s.scheduler != nil {
		maxConcurrency = s.scheduler.MaxConcurrency()
	}

	binaries := map[string]string{}
	for _, name := range []string{"claude", "codex", "opencode", "gemini"} {
		if p, err := exec.LookPath(name); err == nil {
			binaries[name] = p
		} else {
			binaries[name] = "not found"
		}
	}

	wsRoot := ".wave/workspaces"
	if s.manifest != nil && s.manifest.Runtime.WorkspaceRoot != "" {
		wsRoot = s.manifest.Runtime.WorkspaceRoot
	}

	resp := adminConfigResponse{
		Address:         addr,
		MaxConcurrency:  maxConcurrency,
		AdapterBinaries: binaries,
		WorkspaceRoot:   wsRoot,
		AuthMode:        string(s.authMode),
	}

	writeJSON(w, http.StatusOK, resp)
}

// adminCredentialsResponse is the JSON response for GET /api/admin/credentials.
type adminCredentialsResponse map[string]bool

// handleAPIAdminCredentials handles GET /api/admin/credentials.
// Only returns keys that are actually set, to avoid exposing the full credential surface.
func (s *Server) handleAPIAdminCredentials(w http.ResponseWriter, _ *http.Request) {
	allKeys := []string{
		"ANTHROPIC_API_KEY",
		"GH_TOKEN",
		"GITLAB_TOKEN",
		"GITEA_TOKEN",
		"CODEBERG_TOKEN",
		"BITBUCKET_TOKEN",
		"OPENAI_API_KEY",
		"GOOGLE_APPLICATION_CREDENTIALS",
	}

	resp := make(adminCredentialsResponse)
	for _, key := range allKeys {
		if os.Getenv(key) != "" {
			resp[key] = true
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// emergencyStopResponse is the JSON response for POST /api/admin/emergency-stop.
type emergencyStopResponse struct {
	Cancelled int      `json:"cancelled"`
	RunIDs    []string `json:"run_ids"`
}

// handleAPIEmergencyStop handles POST /api/admin/emergency-stop.
func (s *Server) handleAPIEmergencyStop(w http.ResponseWriter, _ *http.Request) {
	runs, err := s.store.GetRunningRuns()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get running runs: "+err.Error())
		return
	}

	var cancelledIDs []string
	for _, run := range runs {
		// Cancel the goroutine context if active
		s.mu.Lock()
		if cancelFn, ok := s.activeRuns[run.RunID]; ok {
			cancelFn()
		}
		s.mu.Unlock()

		if err := s.rwStore.RequestCancellation(run.RunID, true); err != nil {
			continue // best-effort: skip runs that fail to cancel
		}
		cancelledIDs = append(cancelledIDs, run.RunID)
	}

	resp := emergencyStopResponse{
		Cancelled: len(cancelledIDs),
		RunIDs:    cancelledIDs,
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Pipeline Enable/Disable ---

// pipelineToggleResponse is the JSON response for pipeline enable/disable.
type pipelineToggleResponse struct {
	Name     string `json:"name"`
	Disabled bool   `json:"disabled"`
}

// handleDisablePipeline handles POST /api/admin/pipelines/{name}/disable.
func (s *Server) handleDisablePipeline(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "missing pipeline name")
		return
	}

	s.mu.Lock()
	s.disabledPipelines[name] = true
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, pipelineToggleResponse{Name: name, Disabled: true})
}

// handleEnablePipeline handles POST /api/admin/pipelines/{name}/enable.
func (s *Server) handleEnablePipeline(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "missing pipeline name")
		return
	}

	s.mu.Lock()
	delete(s.disabledPipelines, name)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, pipelineToggleResponse{Name: name, Disabled: false})
}

// isPipelineDisabled checks whether a pipeline is currently disabled.
func (s *Server) isPipelineDisabled(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.disabledPipelines[name]
}

// --- Audit Log ---

// auditEventResponse is a single entry in the audit log API response.
type auditEventResponse struct {
	ID        int64  `json:"id"`
	RunID     string `json:"run_id"`
	Timestamp string `json:"timestamp"`
	State     string `json:"state"`
	StepID    string `json:"step_id,omitempty"`
	Persona   string `json:"persona,omitempty"`
	Message   string `json:"message,omitempty"`
}

// auditLogResponse is the JSON response for GET /api/admin/audit.
type auditLogResponse struct {
	Events []auditEventResponse `json:"events"`
	Total  int                  `json:"total"`
}

// auditEventStates defines which event types appear in the audit log.
var auditEventStates = []string{
	"run_start",
	"run_completed",
	"run_failed",
	"step_failed",
	"gate_requested",
}

// handleAPIAdminAudit handles GET /api/admin/audit.
func (s *Server) handleAPIAdminAudit(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	events, err := s.store.GetAuditEvents(auditEventStates, limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to query audit events: "+err.Error())
		return
	}

	resp := auditLogResponse{
		Events: make([]auditEventResponse, 0, len(events)),
		Total:  len(events),
	}

	for _, ev := range events {
		resp.Events = append(resp.Events, auditEventResponse{
			ID:        ev.ID,
			RunID:     ev.RunID,
			Timestamp: ev.Timestamp.Format(time.RFC3339),
			State:     ev.State,
			StepID:    ev.StepID,
			Persona:   ev.Persona,
			Message:   ev.Message,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// getDisabledPipelineSet returns a snapshot of currently disabled pipeline names.
func (s *Server) getDisabledPipelineSet() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make(map[string]bool, len(s.disabledPipelines))
	for k, v := range s.disabledPipelines {
		cp[k] = v
	}
	return cp
}
