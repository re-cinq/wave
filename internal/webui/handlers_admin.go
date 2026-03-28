package webui

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
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
func (s *Server) handleAPIAdminCredentials(w http.ResponseWriter, _ *http.Request) {
	keys := []string{
		"ANTHROPIC_API_KEY",
		"GH_TOKEN",
		"OPENAI_API_KEY",
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GITLAB_TOKEN",
	}

	resp := make(adminCredentialsResponse, len(keys))
	for _, key := range keys {
		resp[key] = os.Getenv(key) != ""
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
