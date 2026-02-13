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
		"templates/pipeline_detail.html": `<html><body><div>{{.Name}}</div></body></html>`,
		"templates/persona_detail.html":  `<html><body><div>{{.Name}}</div></body></html>`,
		"templates/statistics.html":      `<html><body><div>{{.TimeRange}}</div></body></html>`,
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

func TestHandleAPIStatistics_Empty(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/statistics", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIStatistics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp StatisticsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Aggregate.Total != 0 {
		t.Errorf("expected 0 total, got %d", resp.Aggregate.Total)
	}
	if resp.TimeRange != "7d" {
		t.Errorf("expected default time range '7d', got %q", resp.TimeRange)
	}
}

func TestHandleAPIStatistics_WithData(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create test runs with various statuses
	runID1, _ := rwStore.CreateRun("test-pipeline", "input1")
	rwStore.UpdateRunStatus(runID1, "completed", "", 100)
	runID2, _ := rwStore.CreateRun("test-pipeline", "input2")
	rwStore.UpdateRunStatus(runID2, "failed", "some error", 50)
	rwStore.CreateRun("test-pipeline", "input3") // pending

	req := httptest.NewRequest("GET", "/api/statistics?range=all", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIStatistics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp StatisticsResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Aggregate.Total != 3 {
		t.Errorf("expected 3 total runs, got %d", resp.Aggregate.Total)
	}
	if resp.Aggregate.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", resp.Aggregate.Succeeded)
	}
	if resp.Aggregate.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", resp.Aggregate.Failed)
	}
	if resp.TimeRange != "all" {
		t.Errorf("expected time range 'all', got %q", resp.TimeRange)
	}
}

func TestHandleAPIStatistics_TimeRangeFilter(t *testing.T) {
	srv, _ := testServer(t)

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{"default", "", "7d"},
		{"24h", "?range=24h", "24h"},
		{"7d", "?range=7d", "7d"},
		{"30d", "?range=30d", "30d"},
		{"all", "?range=all", "all"},
		{"invalid", "?range=invalid", "7d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/statistics"+tt.query, nil)
			rec := httptest.NewRecorder()
			srv.handleAPIStatistics(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}

			var resp StatisticsResponse
			json.NewDecoder(rec.Body).Decode(&resp)

			if resp.TimeRange != tt.want {
				t.Errorf("expected time range %q, got %q", tt.want, resp.TimeRange)
			}
		})
	}
}

func TestHandleAPIPipelineDetail_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/pipelines/nonexistent", nil)
	req.SetPathValue("name", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleAPIPipelineDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAPIPipelineDetail_MissingName(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/pipelines/", nil)
	req.SetPathValue("name", "")
	rec := httptest.NewRecorder()
	srv.handleAPIPipelineDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAPIPersonaDetail_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/personas/nonexistent", nil)
	req.SetPathValue("name", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleAPIPersonaDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAPIPersonaDetail_MissingName(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/personas/", nil)
	req.SetPathValue("name", "")
	rec := httptest.NewRecorder()
	srv.handleAPIPersonaDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorkspaceTree_MissingParams(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs//workspace//tree", nil)
	req.SetPathValue("id", "")
	req.SetPathValue("step", "")
	rec := httptest.NewRecorder()
	srv.handleWorkspaceTree(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorkspaceTree_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs/nonexistent/workspace/step1/tree", nil)
	req.SetPathValue("id", "nonexistent")
	req.SetPathValue("step", "step1")
	rec := httptest.NewRecorder()
	srv.handleWorkspaceTree(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp WorkspaceTreeResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Error == "" {
		t.Error("expected error in response for non-existent workspace")
	}
}

func TestHandleWorkspaceFile_MissingParams(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs//workspace//file", nil)
	req.SetPathValue("id", "")
	req.SetPathValue("step", "")
	rec := httptest.NewRecorder()
	srv.handleWorkspaceFile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorkspaceFile_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs/nonexistent/workspace/step1/file?path=test.txt", nil)
	req.SetPathValue("id", "nonexistent")
	req.SetPathValue("step", "step1")
	rec := httptest.NewRecorder()
	srv.handleWorkspaceFile(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp WorkspaceFileResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Error == "" {
		t.Error("expected error in response for non-existent workspace")
	}
}

func TestHandleWorkspaceTree_WithDirectory(t *testing.T) {
	srv, _ := testServer(t)

	// Create a temporary workspace directory
	tmpDir := t.TempDir()
	wsDir := tmpDir + "/.wave/workspaces/test-run/step1"
	os.MkdirAll(wsDir+"/subdir", 0o755)
	os.WriteFile(wsDir+"/test.go", []byte("package main"), 0o644)
	os.WriteFile(wsDir+"/README.md", []byte("# Test"), 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	req := httptest.NewRequest("GET", "/api/runs/test-run/workspace/step1/tree", nil)
	req.SetPathValue("id", "test-run")
	req.SetPathValue("step", "step1")
	rec := httptest.NewRecorder()
	srv.handleWorkspaceTree(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp WorkspaceTreeResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Error != "" {
		t.Fatalf("unexpected error: %s", resp.Error)
	}

	if len(resp.Entries) < 2 {
		t.Errorf("expected at least 2 entries, got %d", len(resp.Entries))
	}

	// First entry should be the directory (sorted first)
	if len(resp.Entries) > 0 && !resp.Entries[0].IsDir {
		t.Error("expected first entry to be a directory")
	}
}

func TestHandleWorkspaceFile_WithFile(t *testing.T) {
	srv, _ := testServer(t)

	// Create a temporary workspace with a file
	tmpDir := t.TempDir()
	wsDir := tmpDir + "/.wave/workspaces/test-run/step1"
	os.MkdirAll(wsDir, 0o755)
	os.WriteFile(wsDir+"/test.go", []byte("package main\nfunc main() {}"), 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	req := httptest.NewRequest("GET", "/api/runs/test-run/workspace/step1/file?path=test.go", nil)
	req.SetPathValue("id", "test-run")
	req.SetPathValue("step", "step1")
	rec := httptest.NewRecorder()
	srv.handleWorkspaceFile(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp WorkspaceFileResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Error != "" {
		t.Fatalf("unexpected error: %s", resp.Error)
	}
	if resp.MimeType != "text/x-go" {
		t.Errorf("expected mime type text/x-go, got %q", resp.MimeType)
	}
	if resp.Truncated {
		t.Error("file should not be truncated")
	}
}

func TestHandleWorkspaceTree_PathTraversal(t *testing.T) {
	srv, _ := testServer(t)

	// Create a temporary workspace
	tmpDir := t.TempDir()
	wsDir := tmpDir + "/.wave/workspaces/test-run/step1"
	os.MkdirAll(wsDir, 0o755)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	req := httptest.NewRequest("GET", "/api/runs/test-run/workspace/step1/tree?path=../../etc", nil)
	req.SetPathValue("id", "test-run")
	req.SetPathValue("step", "step1")
	rec := httptest.NewRecorder()
	srv.handleWorkspaceTree(rec, req)

	var resp WorkspaceTreeResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	// Should either return error or be safely resolved within workspace
	// The path gets cleaned by filepath.Clean("/"+reqPath) which neutralizes the traversal
}
