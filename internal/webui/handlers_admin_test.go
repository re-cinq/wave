package webui

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAPIAdminConfig(t *testing.T) {
	srv, _ := testServer(t)
	srv.runtime.scheduler = NewScheduler(3)

	req := httptest.NewRequest("GET", "/api/admin/config", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIAdminConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp adminConfigResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.MaxConcurrency != 3 {
		t.Errorf("expected max_concurrency=3, got %d", resp.MaxConcurrency)
	}
	if resp.Address != "127.0.0.1:0" {
		t.Errorf("expected address 127.0.0.1:0, got %s", resp.Address)
	}
	// testServer doesn't set authMode (it skips NewServer's resolution logic),
	// so expect the zero value. Production always resolves to "none" or "bearer".
	if resp.AuthMode != "" {
		t.Errorf("expected auth_mode to be empty in test, got %s", resp.AuthMode)
	}
	if resp.WorkspaceRoot == "" {
		t.Error("expected non-empty workspace_root")
	}
	if resp.AdapterBinaries == nil {
		t.Error("expected non-nil adapter_binaries")
	}
	// All known adapters should be present
	for _, name := range []string{"claude", "codex", "opencode", "gemini"} {
		if _, ok := resp.AdapterBinaries[name]; !ok {
			t.Errorf("missing adapter binary entry for %s", name)
		}
	}
}

func TestHandleAPIAdminConfig_NilScheduler(t *testing.T) {
	srv, _ := testServer(t)
	srv.runtime.scheduler = nil

	req := httptest.NewRequest("GET", "/api/admin/config", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIAdminConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp adminConfigResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.MaxConcurrency != 5 {
		t.Errorf("expected default max_concurrency=5 when scheduler is nil, got %d", resp.MaxConcurrency)
	}
}

func TestHandleAPIAdminCredentials(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/admin/credentials", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIAdminCredentials(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp adminCredentialsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Only set credentials should appear — unset keys are excluded to avoid
	// exposing the full credential surface in shared/multi-user servers.
	for key, val := range resp {
		if !val {
			t.Errorf("unset credential key %q should not appear in response", key)
		}
	}

	// Set a test env var and verify it appears
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	req2 := httptest.NewRequest("GET", "/api/admin/credentials", nil)
	rec2 := httptest.NewRecorder()
	srv.handleAPIAdminCredentials(rec2, req2)

	var resp2 adminCredentialsResponse
	if err := json.NewDecoder(rec2.Body).Decode(&resp2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if v, ok := resp2["ANTHROPIC_API_KEY"]; !ok || !v {
		t.Errorf("expected ANTHROPIC_API_KEY to be present and true when set")
	}
}

func TestHandleAPIEmergencyStop_NoRunning(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("POST", "/api/admin/emergency-stop", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIEmergencyStop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp emergencyStopResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Cancelled != 0 {
		t.Errorf("expected 0 cancelled, got %d", resp.Cancelled)
	}
}

func TestHandleAPIEmergencyStop_WithRunning(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create a run and mark it as running
	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}
	if err := rwStore.UpdateRunStatus(runID, "running", "", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/admin/emergency-stop", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIEmergencyStop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp emergencyStopResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Cancelled != 1 {
		t.Errorf("expected 1 cancelled, got %d", resp.Cancelled)
	}
	if len(resp.RunIDs) != 1 || resp.RunIDs[0] != runID {
		t.Errorf("expected run_ids=[%s], got %v", runID, resp.RunIDs)
	}
}

func TestHandleAdminPage(t *testing.T) {
	srv, _ := testServer(t)
	// Add admin template stub
	srv.assets.templates["templates/admin.html"] = testAdminTemplate(t)

	req := httptest.NewRequest("GET", "/admin", nil)
	rec := httptest.NewRecorder()
	srv.handleAdminPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("expected text/html content type, got %s", ct)
	}
}

func testAdminTemplate(t *testing.T) *template.Template {
	t.Helper()
	return template.Must(template.New("templates/layout.html").Parse(`<html><body>Admin</body></html>`))
}
