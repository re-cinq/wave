//go:build webui

package webui

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/state"
)

// testTemplates creates minimal stub templates for handler tests.
// Each page gets its own template set with a "templates/layout.html" entry
// that the handlers execute, matching the clone-per-page production approach.
func testTemplates(t *testing.T) map[string]*template.Template {
	t.Helper()
	funcMap := template.FuncMap{
		"statusClass":    statusClass,
		"formatDuration": formatDuration,
		"formatTime":     formatTime,
	}
	pages := map[string]string{
		"templates/runs.html":       `<html><body>{{range .Runs}}<div>{{.RunID}}</div>{{end}}</body></html>`,
		"templates/run_detail.html": `<html><body><div>{{.Run.RunID}}</div></body></html>`,
		"templates/personas.html":   `<html><body>{{range .Personas}}<div>{{.Name}}</div>{{end}}</body></html>`,
		"templates/pipelines.html":  `<html><body>{{range .Pipelines}}<div>{{.Name}}</div>{{end}}</body></html>`,
	}
	result := make(map[string]*template.Template, len(pages))
	for name, body := range pages {
		tmpl := template.Must(template.New("templates/layout.html").Funcs(funcMap).Parse(body))
		result[name] = tmpl
	}
	return result
}

// testServer creates a test server with a temporary database.
func testServer(t *testing.T) (*Server, state.StateStore) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// Create the database with the RW store
	rwStore, err := state.NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create state store: %v", err)
	}

	roStore, err := state.NewReadOnlyStateStore(dbPath)
	if err != nil {
		rwStore.Close()
		t.Fatalf("failed to create read-only state store: %v", err)
	}

	tmpl := testTemplates(t)

	srv := &Server{
		store:      roStore,
		rwStore:    rwStore,
		templates:  tmpl,
		broker:     NewSSEBroker(),
		bind:       "127.0.0.1",
		port:       0,
		activeRuns: make(map[string]context.CancelFunc),
	}

	t.Cleanup(func() {
		roStore.Close()
		rwStore.Close()
	})

	return srv, rwStore
}

func TestHandleAPIRuns_Empty(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIRuns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp RunListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(resp.Runs))
	}
	if resp.HasMore {
		t.Error("expected HasMore to be false")
	}
}

func TestHandleAPIRuns_WithData(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create test runs
	for i := 0; i < 3; i++ {
		_, err := rwStore.CreateRun("test-pipeline", "input")
		if err != nil {
			t.Fatalf("failed to create run: %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/runs", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIRuns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp RunListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Runs) != 3 {
		t.Errorf("expected 3 runs, got %d", len(resp.Runs))
	}
}

func TestHandleAPIRuns_StatusFilter(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	rwStore.UpdateRunStatus(runID, "completed", "", 100)
	rwStore.CreateRun("test-pipeline", "input2") // pending

	req := httptest.NewRequest("GET", "/api/runs?status=completed", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIRuns(rec, req)

	var resp RunListResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Runs) != 1 {
		t.Errorf("expected 1 completed run, got %d", len(resp.Runs))
	}
}

func TestHandleAPIRuns_Pagination(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create 30 runs (more than default page size of 25)
	for i := 0; i < 30; i++ {
		rwStore.CreateRun("test-pipeline", "input")
	}

	req := httptest.NewRequest("GET", "/api/runs", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIRuns(rec, req)

	var resp RunListResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Runs) != 25 {
		t.Errorf("expected 25 runs (page size), got %d", len(resp.Runs))
	}
	if !resp.HasMore {
		t.Error("expected HasMore to be true")
	}
	if resp.NextCursor == "" {
		t.Error("expected NextCursor to be set")
	}
}

func TestHandleAPIRunDetail(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "test input")

	req := httptest.NewRequest("GET", "/api/runs/"+runID, nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleAPIRunDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp RunDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Run.RunID != runID {
		t.Errorf("expected run ID %q, got %q", runID, resp.Run.RunID)
	}
	if resp.Run.PipelineName != "test-pipeline" {
		t.Errorf("expected pipeline name 'test-pipeline', got %q", resp.Run.PipelineName)
	}
}

func TestHandleAPIRunDetail_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleAPIRunDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleStartPipeline_MissingPipeline(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`{"input":"test input"}`)
	req := httptest.NewRequest("POST", "/api/pipelines/nonexistent/start", body)
	req.SetPathValue("name", "nonexistent")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleStartPipeline(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing pipeline, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleStartPipeline_WithPipeline(t *testing.T) {
	srv, _ := testServer(t)

	// Create a minimal pipeline YAML in a temp location
	tmpDir := t.TempDir()
	pipelineYAML := `kind: Pipeline
metadata:
  name: test-pipeline
steps:
  - id: step1
    persona: navigator
    exec:
      prompt: "test"
`
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	os.MkdirAll(pipelineDir, 0o755)
	os.WriteFile(filepath.Join(pipelineDir, "test-pipeline.yaml"), []byte(pipelineYAML), 0o644)

	// Change to temp dir so loadPipelineYAML finds the file
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	body := strings.NewReader(`{"input":"test input"}`)
	req := httptest.NewRequest("POST", "/api/pipelines/test-pipeline/start", body)
	req.SetPathValue("name", "test-pipeline")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleStartPipeline(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp StartPipelineResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.PipelineName != "test-pipeline" {
		t.Errorf("expected pipeline name 'test-pipeline', got %q", resp.PipelineName)
	}
	if resp.RunID == "" {
		t.Error("expected non-empty run ID")
	}
	if resp.Status != "running" {
		t.Errorf("expected status 'running', got %q", resp.Status)
	}
}

func TestHandleCancelRun(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	rwStore.UpdateRunStatus(runID, "running", "step1", 0)

	body := strings.NewReader(`{"force": false}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/cancel", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleCancelRun(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCancelRun_NotCancellable(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	rwStore.UpdateRunStatus(runID, "completed", "", 100)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/cancel", body)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleCancelRun(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for completed run, got %d", rec.Code)
	}
}

func TestHandleRetryRun(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	rwStore.UpdateRunStatus(runID, "failed", "", 0)

	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/retry", nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleRetryRun(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp RetryRunResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.OriginalRunID != runID {
		t.Errorf("expected original run ID %q, got %q", runID, resp.OriginalRunID)
	}
	if resp.PipelineName != "test-pipeline" {
		t.Errorf("expected pipeline name 'test-pipeline', got %q", resp.PipelineName)
	}
}

func TestHandleRetryRun_NotRetryable(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	rwStore.UpdateRunStatus(runID, "running", "step1", 0)

	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/retry", nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleRetryRun(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for running run, got %d", rec.Code)
	}
}

func TestHandleAPIPersonas_NoManifest(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/personas", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPersonas(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp PersonaListResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	// No manifest means no personas - should return empty list, not error
	if resp.Personas != nil && len(resp.Personas) > 0 {
		t.Errorf("expected empty personas, got %d", len(resp.Personas))
	}
}

func TestHandleRunsPage(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/runs", nil)
	rec := httptest.NewRecorder()
	srv.handleRunsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}
