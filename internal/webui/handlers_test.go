package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/manifest"
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
		"formatTokens":   formatTokensFunc,
		"csrfToken":      func() string { return "test-csrf-token" },
		"add":            func(a, b int) int { return a + b },
		"subtract":       func(a, b int) int { return a - b },
		"hasPrefix":      strings.HasPrefix,
		"pluralize": func(n int, singular, plural string) string {
			if n == 1 {
				return singular
			}
			return plural
		},
	}
	pages := map[string]string{
		"templates/runs.html":           `<html><body><nav>{{if eq .ActivePage "runs"}}<a class="nav-link-active">Runs</a>{{end}}</nav>{{range .Runs}}<div>{{.RunID}}</div>{{end}}</body></html>`,
		"templates/run_detail.html":     `<html><body><nav>{{if eq .ActivePage "runs"}}<a class="nav-link-active">Runs</a>{{end}}</nav><div>{{.Run.RunID}}</div></body></html>`,
		"templates/personas.html":       `<html><body>{{range .Personas}}<div>{{.Name}}</div>{{end}}</body></html>`,
		"templates/pipelines.html":      `<html><body>{{range .Pipelines}}<div>{{.Name}}</div>{{end}}</body></html>`,
		"templates/contracts.html":      `<html><body>{{range .Contracts}}<div>{{.Name}}</div>{{end}}</body></html>`,
		"templates/skills.html":         `<html><body>{{range .Skills}}<div>{{.Name}}</div>{{end}}</body></html>`,
		"templates/compose.html":        `<html><body>{{range .Pipelines}}<div>{{.Name}}</div>{{end}}</body></html>`,
		"templates/issues.html":         `<html><body><span class="filter">{{.FilterState}}</span>{{range .Issues}}<div>#{{.Number}} {{.Title}}</div>{{end}}{{if .Message}}<p>{{.Message}}</p>{{end}}</body></html>`,
		"templates/prs.html":            `<html><body><span class="filter">{{.FilterState}}</span>{{range .PullRequests}}<div>#{{.Number}} {{.Title}}</div>{{end}}{{if .Message}}<p>{{.Message}}</p>{{end}}</body></html>`,
		"templates/health.html":         `<html><body>{{range .Checks}}<div>{{.Name}}: {{.Status}}</div>{{end}}</body></html>`,
		"templates/ontology.html":       `<html><body>{{if .HasOntology}}<div>{{.Telos}}</div>{{range .Contexts}}<div>{{.Name}}</div>{{end}}{{end}}</body></html>`,
		"templates/notfound.html":       `<html><body>Page not found</body></html>`,
		"templates/compare.html":        `<html><body><nav>nav</nav>{{if .Error}}<div class="alert alert-error">{{.Error}}</div>{{end}}{{if .ShowSelector}}<select id="compare-left-select" name="left"></select><select id="compare-right-select" name="right"></select>{{end}}{{if not .ShowSelector}}{{.Left.RunID}} vs {{.Right.RunID}}{{range .Rows}}<tr>{{.StepID}}</tr>{{end}}{{end}}</body></html>`,
		"templates/analytics.html":      `<html><body><h1>Token Usage Analytics</h1><div>{{formatTokens .Analytics.TotalTokens}}</div><div>{{.Analytics.TotalRuns}} {{pluralize .Analytics.TotalRuns "run" "runs"}}</div></body></html>`,
		"templates/retros.html":         `<html><body>{{if .HasData}}<div>retros</div>{{end}}</body></html>`,
		"templates/webhook_detail.html": `<html><body><div>{{.Webhook.Name}}</div>{{range .Deliveries}}<div>{{.Event}}</div>{{end}}</body></html>`,
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
		store:             roStore,
		rwStore:           rwStore,
		templates:         tmpl,
		broker:            NewSSEBroker(),
		bind:              "127.0.0.1",
		port:              0,
		activeRuns:        make(map[string]context.CancelFunc),
		disabledPipelines: make(map[string]bool),
		gateRegistry:      NewGateRegistry(),
		csrfToken:         "test-csrf-token",
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
	if err := rwStore.UpdateRunStatus(runID, "completed", "", 100); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}
	if _, err := rwStore.CreateRun("test-pipeline", "input2"); err != nil { // pending
		t.Fatalf("failed to create run: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs?status=completed", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIRuns(rec, req)

	var resp RunListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Runs) != 1 {
		t.Errorf("expected 1 completed run, got %d", len(resp.Runs))
	}
}

func TestHandleAPIRuns_Pagination(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create 30 runs (more than default page size of 25)
	for i := 0; i < 30; i++ {
		if _, err := rwStore.CreateRun("test-pipeline", "input"); err != nil {
			t.Fatalf("failed to create run: %v", err)
		}
	}

	// Verify all rows are visible through the read-only store before testing
	// pagination. SQLite WAL mode can have brief visibility delays between
	// separate connections.
	runs, err := srv.store.ListRuns(state.ListRunsOptions{Limit: 50})
	if err != nil {
		t.Fatalf("failed to verify test data: %v", err)
	}
	if len(runs) != 30 {
		t.Fatalf("expected 30 runs to be visible through roStore, got %d (WAL sync issue)", len(runs))
	}

	req := httptest.NewRequest("GET", "/api/runs", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIRuns(rec, req)

	var resp RunListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

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
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pipelineDir, "test-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}

	// Change to temp dir so loadPipelineYAML finds the file
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

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
	if err := rwStore.UpdateRunStatus(runID, "running", "step1", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

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
	if err := rwStore.UpdateRunStatus(runID, "completed", "", 100); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

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

	// Create pipeline YAML so retry can load it
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
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pipelineDir, "test-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "failed", "", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/retry", nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleRetryRun(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp RetryRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.OriginalRunID != runID {
		t.Errorf("expected original run ID %q, got %q", runID, resp.OriginalRunID)
	}
	if resp.PipelineName != "test-pipeline" {
		t.Errorf("expected pipeline name 'test-pipeline', got %q", resp.PipelineName)
	}
	if resp.Status != "running" {
		t.Errorf("expected status 'running', got %q", resp.Status)
	}
}

func TestHandleRetryRun_NotRetryable(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "running", "step1", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/retry", nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleRetryRun(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for running run, got %d", rec.Code)
	}
}

func TestHandleResumeRun(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create pipeline YAML
	tmpDir := t.TempDir()
	pipelineYAML := `kind: Pipeline
metadata:
  name: test-pipeline
steps:
  - id: step1
    persona: navigator
    exec:
      prompt: "test"
  - id: step2
    persona: navigator
    depends_on: [step1]
    exec:
      prompt: "test2"
`
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pipelineDir, "test-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "failed", "", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	body := strings.NewReader(`{"from_step":"step2"}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/resume", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleResumeRun(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ResumeRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.OriginalRunID != runID {
		t.Errorf("expected original run ID %q, got %q", runID, resp.OriginalRunID)
	}
	if resp.FromStep != "step2" {
		t.Errorf("expected from_step 'step2', got %q", resp.FromStep)
	}
	if resp.Status != "running" {
		t.Errorf("expected status 'running', got %q", resp.Status)
	}
}

func TestHandleResumeRun_InvalidStep(t *testing.T) {
	srv, rwStore := testServer(t)

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
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pipelineDir, "test-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "failed", "", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	body := strings.NewReader(`{"from_step":"nonexistent"}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/resume", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleResumeRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid step, got %d", rec.Code)
	}
}

func TestHandleResumeRun_WrongState(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "running", "step1", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	body := strings.NewReader(`{"from_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/resume", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleResumeRun(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for running run, got %d", rec.Code)
	}
}

// flusherRecorder wraps httptest.ResponseRecorder to implement http.Flusher.
type flusherRecorder struct {
	*httptest.ResponseRecorder
}

func (f *flusherRecorder) Flush() {}

func TestHandleSSE_BackfillOnReconnect(t *testing.T) {
	srv, rwStore := testServer(t)
	go srv.broker.Start()
	defer srv.broker.Stop()

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "running", "step1", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	// Log some events to the DB
	if err := rwStore.LogEvent(runID, "step1", "started", "navigator", "Starting step1", 0, 0, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}
	if err := rwStore.LogEvent(runID, "step1", "running", "navigator", "Processing", 100, 1000, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	// Get the events to find their IDs
	events, err := srv.store.GetEvents(runID, state.EventQueryOptions{})
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	// Add a third event after the first two
	if err := rwStore.LogEvent(runID, "step1", "completed", "navigator", "Done", 200, 2000, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	// Simulate reconnection with Last-Event-ID set to the first event
	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/events", nil)
	req.SetPathValue("id", runID)
	req.Header.Set("Last-Event-ID", fmt.Sprintf("%d", events[0].ID))

	rec := &flusherRecorder{httptest.NewRecorder()}

	// Run in goroutine since SSE handler blocks; cancel via context
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		srv.handleSSE(rec, req)
		close(done)
	}()

	// Give handler time to write backfill, then cancel
	cancel()
	<-done

	body := rec.Body.String()
	if !strings.Contains(body, "retry: 3000") {
		t.Errorf("expected retry directive in SSE output, got: %s", body)
	}
	// Should contain backfilled events (events with ID > first event)
	if !strings.Contains(body, "Processing") {
		t.Errorf("expected backfilled event with 'Processing' message, got: %s", body)
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
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// No manifest means no personas - should return empty list, not error
	if len(resp.Personas) > 0 {
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

func TestHandleAPIContracts_NoDir(t *testing.T) {
	srv, _ := testServer(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/contracts", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIContracts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestHandleAPIContracts_WithFiles(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, ".wave", "contracts")
	if err := os.MkdirAll(contractDir, 0o755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}
	schema := `{"$schema":"http://json-schema.org/draft-07/schema#","title":"Test Contract","description":"A test schema","type":"object"}`
	if err := os.WriteFile(filepath.Join(contractDir, "test-contract.schema.json"), []byte(schema), 0o644); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/contracts", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIContracts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	contracts, ok := resp["contracts"].([]interface{})
	if !ok {
		t.Fatalf("expected contracts array, got %T", resp["contracts"])
	}
	if len(contracts) != 1 {
		t.Errorf("expected 1 contract, got %d", len(contracts))
	}
}

func TestHandleAPIContractDetail(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, ".wave", "contracts")
	if err := os.MkdirAll(contractDir, 0o755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}
	schema := `{"title":"My Contract","description":"desc","type":"object"}`
	if err := os.WriteFile(filepath.Join(contractDir, "my-contract.schema.json"), []byte(schema), 0o644); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/contracts/my-contract", nil)
	req.SetPathValue("name", "my-contract")
	rec := httptest.NewRecorder()
	srv.handleAPIContractDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ContractDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Name != "my-contract" {
		t.Errorf("expected name 'my-contract', got %q", resp.Name)
	}
	if resp.Title != "My Contract" {
		t.Errorf("expected title 'My Contract', got %q", resp.Title)
	}
	if resp.Schema != schema {
		t.Errorf("expected schema content, got %q", resp.Schema)
	}
}

func TestHandleAPIContractDetail_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/contracts/nonexistent", nil)
	req.SetPathValue("name", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleAPIContractDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAPIContractDetail_PathTraversal(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/contracts/../../etc/passwd", nil)
	req.SetPathValue("name", "../../etc/passwd")
	rec := httptest.NewRecorder()
	srv.handleAPIContractDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for path traversal, got %d", rec.Code)
	}
}

func TestHandleContractsPage(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/contracts", nil)
	rec := httptest.NewRecorder()
	srv.handleContractsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestHandleAPISkills_NoDir(t *testing.T) {
	srv, _ := testServer(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/skills", nil)
	rec := httptest.NewRecorder()
	srv.handleAPISkills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp SkillListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// No pipelines dir means no skills
	if len(resp.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(resp.Skills))
	}
}

func TestHandleAPISkills_WithPipelines(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	pipelineYAML := `kind: Pipeline
metadata:
  name: test-pipeline
requires:
  skills:
    golang:
      check: "go version"
    speckit:
      install: "wave skill install speckit"
      commands_glob: ".claude/commands/speckit.*.md"
steps:
  - id: step1
    persona: navigator
    exec:
      prompt: "test"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "test-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/skills", nil)
	rec := httptest.NewRecorder()
	srv.handleAPISkills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp SkillListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(resp.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(resp.Skills))
	}

	// Skills should be sorted alphabetically
	if resp.Skills[0].Name != "golang" {
		t.Errorf("expected first skill 'golang', got %q", resp.Skills[0].Name)
	}
	if resp.Skills[1].Name != "speckit" {
		t.Errorf("expected second skill 'speckit', got %q", resp.Skills[1].Name)
	}

	// Verify golang details
	if resp.Skills[0].CheckCmd != "go version" {
		t.Errorf("expected golang check cmd 'go version', got %q", resp.Skills[0].CheckCmd)
	}

	// Verify speckit details
	if resp.Skills[1].InstallCmd != "wave skill install speckit" {
		t.Errorf("expected speckit install cmd, got %q", resp.Skills[1].InstallCmd)
	}
	if resp.Skills[1].CommandsGlob != ".claude/commands/speckit.*.md" {
		t.Errorf("expected speckit commands glob, got %q", resp.Skills[1].CommandsGlob)
	}

	// Both should reference test-pipeline
	if len(resp.Skills[0].PipelineUsage) != 1 || resp.Skills[0].PipelineUsage[0] != "test-pipeline" {
		t.Errorf("expected golang to be used by test-pipeline, got %v", resp.Skills[0].PipelineUsage)
	}
}

func TestHandleSkillsPage(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/skills", nil)
	rec := httptest.NewRecorder()
	srv.handleSkillsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestHandleAPICompose_NoComposition(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	// Regular pipeline with no composition primitives
	pipelineYAML := `kind: Pipeline
metadata:
  name: simple-pipeline
  description: A simple pipeline
steps:
  - id: step1
    persona: navigator
    exec:
      prompt: "test"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "simple-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/compose", nil)
	rec := httptest.NewRecorder()
	srv.handleAPICompose(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp CompositionListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(resp.Pipelines) != 0 {
		t.Errorf("expected 0 composition pipelines, got %d", len(resp.Pipelines))
	}
}

func TestHandleAPICompose_WithComposition(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	// Pipeline with composition primitives
	pipelineYAML := `kind: Pipeline
metadata:
  name: batch-impl
  description: Batch implementation with iteration
  category: impl
steps:
  - id: plan
    persona: navigator
    exec:
      prompt: "plan"
  - id: iterate-tasks
    pipeline: impl-issue
    iterate:
      over: "{{ plan.output }}"
      mode: parallel
      max_concurrent: 3
  - id: aggregate
    aggregate:
      from: "{{ iterate-tasks.output }}"
      into: .wave/artifacts/results.json
      strategy: merge_arrays
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "batch-impl.yaml"), []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/compose", nil)
	rec := httptest.NewRecorder()
	srv.handleAPICompose(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp CompositionListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(resp.Pipelines) != 1 {
		t.Fatalf("expected 1 composition pipeline, got %d", len(resp.Pipelines))
	}

	p := resp.Pipelines[0]
	if p.Name != "batch-impl" {
		t.Errorf("expected name 'batch-impl', got %q", p.Name)
	}
	if p.Description != "Batch implementation with iteration" {
		t.Errorf("expected description, got %q", p.Description)
	}
	if p.StepCount != 3 {
		t.Errorf("expected 3 steps, got %d", p.StepCount)
	}

	// Check step types
	if len(p.Steps) != 3 {
		t.Fatalf("expected 3 step details, got %d", len(p.Steps))
	}
	if p.Steps[0].Type != "persona" {
		t.Errorf("expected step 0 type 'persona', got %q", p.Steps[0].Type)
	}
	if p.Steps[1].Type != "iterate" {
		t.Errorf("expected step 1 type 'iterate', got %q", p.Steps[1].Type)
	}
	if p.Steps[1].SubPipeline != "impl-issue" {
		t.Errorf("expected step 1 sub-pipeline 'impl-issue', got %q", p.Steps[1].SubPipeline)
	}
	if p.Steps[1].Details["mode"] != "parallel" {
		t.Errorf("expected step 1 mode 'parallel', got %q", p.Steps[1].Details["mode"])
	}
	if p.Steps[2].Type != "aggregate" {
		t.Errorf("expected step 2 type 'aggregate', got %q", p.Steps[2].Type)
	}
	if p.Steps[2].Details["strategy"] != "merge_arrays" {
		t.Errorf("expected step 2 strategy 'merge_arrays', got %q", p.Steps[2].Details["strategy"])
	}
}

func TestHandleComposePage(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/compose", nil)
	rec := httptest.NewRecorder()
	srv.handleComposePage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestHandleAPIPipelineInfo(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	pipelineYAML := `kind: Pipeline
metadata:
  name: test-pipeline
  description: A test pipeline
  category: test
steps:
  - id: step1
    persona: navigator
    exec:
      prompt: "test"
  - id: step2
    persona: craftsman
    exec:
      prompt: "build"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "test-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/pipelines/info", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPipelineInfo(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string][]PipelineStartInfo
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	pipelines := resp["pipelines"]
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}

	if pipelines[0].Name != "test-pipeline" {
		t.Errorf("expected name 'test-pipeline', got %q", pipelines[0].Name)
	}
	if pipelines[0].Description != "A test pipeline" {
		t.Errorf("expected description 'A test pipeline', got %q", pipelines[0].Description)
	}
	if pipelines[0].StepCount != 2 {
		t.Errorf("expected 2 steps, got %d", pipelines[0].StepCount)
	}
	if pipelines[0].Category != "test" {
		t.Errorf("expected category 'test', got %q", pipelines[0].Category)
	}
}

func TestClassifyStep_AllTypes(t *testing.T) {
	tests := []struct {
		name         string
		yamlContent  string
		expectedType string
		expectedSub  string
	}{
		{
			name: "branch step",
			yamlContent: `kind: Pipeline
metadata:
  name: branch-test
steps:
  - id: branch-step
    branch:
      on: "{{ input }}"
      cases:
        small: impl-issue
        large: impl-speckit
        default: skip
`,
			expectedType: "branch",
		},
		{
			name: "gate step",
			yamlContent: `kind: Pipeline
metadata:
  name: gate-test
steps:
  - id: gate-step
    gate:
      type: approval
      timeout: 30m
      message: "Approve?"
`,
			expectedType: "gate",
		},
		{
			name: "loop step",
			yamlContent: `kind: Pipeline
metadata:
  name: loop-test
steps:
  - id: loop-step
    pipeline: impl-issue
    loop:
      max_iterations: 5
      until: "{{ loop-step.output }}"
`,
			expectedType: "loop",
			expectedSub:  "impl-issue",
		},
		{
			name: "sub_pipeline step",
			yamlContent: `kind: Pipeline
metadata:
  name: sub-test
steps:
  - id: sub-step
    pipeline: impl-issue
`,
			expectedType: "sub_pipeline",
			expectedSub:  "impl-issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
			if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
				t.Fatalf("failed to create pipeline dir: %v", err)
			}

			pName := strings.ReplaceAll(tt.name, " ", "-")
			if err := os.WriteFile(filepath.Join(pipelineDir, pName+".yaml"), []byte(tt.yamlContent), 0o644); err != nil {
				t.Fatalf("failed to write pipeline yaml: %v", err)
			}

			origDir, _ := os.Getwd()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to chdir: %v", err)
			}
			defer func() {
				if err := os.Chdir(origDir); err != nil {
					t.Logf("warning: failed to restore dir: %v", err)
				}
			}()

			pipelines := getCompositionPipelines()
			if len(pipelines) != 1 {
				t.Fatalf("expected 1 composition pipeline, got %d", len(pipelines))
			}

			step := pipelines[0].Steps[0]
			if step.Type != tt.expectedType {
				t.Errorf("expected type %q, got %q", tt.expectedType, step.Type)
			}
			if tt.expectedSub != "" && step.SubPipeline != tt.expectedSub {
				t.Errorf("expected sub-pipeline %q, got %q", tt.expectedSub, step.SubPipeline)
			}
		})
	}
}

func TestHandleAPIPipelines_WithCategory(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	pipelineYAML := `kind: Pipeline
metadata:
  name: impl-issue
  description: Implement an issue
  category: impl
steps:
  - id: fetch
    persona: navigator
    exec:
      prompt: "fetch"
  - id: implement
    persona: craftsman
    exec:
      prompt: "implement"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "impl-issue.yaml"), []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/pipelines", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPipelines(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	pipelines, ok := resp["pipelines"].([]interface{})
	if !ok || len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %v", resp["pipelines"])
	}

	pl := pipelines[0].(map[string]interface{})
	if pl["category"] != "impl" {
		t.Errorf("expected category 'impl', got %v", pl["category"])
	}
	if pl["name"] != "impl-issue" {
		t.Errorf("expected name 'impl-issue', got %v", pl["name"])
	}
}

func TestHandlePersonasPage_NilManifest(t *testing.T) {
	srv, _ := testServer(t)
	// srv.manifest is nil by default from testServer

	req := httptest.NewRequest("GET", "/personas", nil)
	rec := httptest.NewRecorder()
	srv.handlePersonasPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestHandlePersonasPage_WithManifest(t *testing.T) {
	srv, _ := testServer(t)

	srv.manifest = &manifest.Manifest{
		Personas: map[string]manifest.Persona{
			"navigator": {
				Description: "Guides the process",
				Adapter:     "claude-code",
				Model:       "opus",
			},
			"craftsman": {
				Description: "Writes code",
				Adapter:     "claude-code",
				Model:       "sonnet",
			},
		},
	}

	req := httptest.NewRequest("GET", "/personas", nil)
	rec := httptest.NewRecorder()
	srv.handlePersonasPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "craftsman") {
		t.Errorf("expected body to contain 'craftsman', got: %s", body)
	}
	if !strings.Contains(body, "navigator") {
		t.Errorf("expected body to contain 'navigator', got: %s", body)
	}
}

func TestHandlePipelinesPage_NoPipelines(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/pipelines", nil)
	rec := httptest.NewRecorder()
	srv.handlePipelinesPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestHandlePipelinesPage_WithPipelines(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}
	pipelineYAML := `kind: Pipeline
metadata:
  name: my-pipeline
  description: My test pipeline
steps:
  - id: step1
    persona: navigator
    exec:
      prompt: "test"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "my-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/pipelines", nil)
	rec := httptest.NewRecorder()
	srv.handlePipelinesPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "my-pipeline") {
		t.Errorf("expected body to contain 'my-pipeline', got: %s", body)
	}
}

func TestHandleAPIPersonas_WithManifest(t *testing.T) {
	srv, _ := testServer(t)

	srv.manifest = &manifest.Manifest{
		Personas: map[string]manifest.Persona{
			"reviewer": {
				Description: "Reviews code",
				Adapter:     "claude-code",
				Model:       "opus",
				Temperature: 0.3,
				Skills:      []string{"golang"},
			},
		},
	}

	req := httptest.NewRequest("GET", "/api/personas", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPersonas(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp PersonaListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Personas) != 1 {
		t.Fatalf("expected 1 persona, got %d", len(resp.Personas))
	}
	if resp.Personas[0].Name != "reviewer" {
		t.Errorf("expected name 'reviewer', got %q", resp.Personas[0].Name)
	}
	if resp.Personas[0].Model != "opus" {
		t.Errorf("expected model 'opus', got %q", resp.Personas[0].Model)
	}
}

func TestHandleNotFound(t *testing.T) {
	srv, _ := testServer(t)

	mux := http.NewServeMux()
	srv.registerRoutes(mux)

	req := httptest.NewRequest("GET", "/nonexistent-path", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Page not found") {
		t.Errorf("expected body to contain 'Page not found', got: %s", body)
	}
}

func TestHandleSubmitRun_Success(t *testing.T) {
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
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pipelineDir, "test-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}

	// Change to temp dir so loadPipelineYAML finds the file
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	body := strings.NewReader(`{"pipeline":"test-pipeline","input":"test"}`)
	req := httptest.NewRequest("POST", "/api/runs", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleSubmitRun(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp SubmitRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.RunID == "" {
		t.Error("expected non-empty RunID")
	}
	if resp.PipelineName != "test-pipeline" {
		t.Errorf("expected pipeline name 'test-pipeline', got %q", resp.PipelineName)
	}
}

func TestHandleSubmitRun_MissingPipeline(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`{"input":"test"}`)
	req := httptest.NewRequest("POST", "/api/runs", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleSubmitRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSubmitRun_NonexistentPipeline(t *testing.T) {
	srv, _ := testServer(t)

	// Change to a temp dir with no pipelines
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	body := strings.NewReader(`{"pipeline":"nonexistent","input":"test"}`)
	req := httptest.NewRequest("POST", "/api/runs", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleSubmitRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRunLogs(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")

	// Log some events
	if err := rwStore.LogEvent(runID, "step1", "started", "navigator", "Starting step1", 0, 0, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}
	if err := rwStore.LogEvent(runID, "step1", "running", "navigator", "Processing", 100, 1000, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/logs", nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleRunLogs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp RunLogsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.RunID != runID {
		t.Errorf("expected run ID %q, got %q", runID, resp.RunID)
	}
	if len(resp.Logs) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(resp.Logs))
	}
	if resp.Logs[0].Message != "Starting step1" {
		t.Errorf("expected first log message 'Starting step1', got %q", resp.Logs[0].Message)
	}
	if resp.Logs[1].Message != "Processing" {
		t.Errorf("expected second log message 'Processing', got %q", resp.Logs[1].Message)
	}
}

func TestHandleRunLogs_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs/nonexistent-run-id/logs", nil)
	req.SetPathValue("id", "nonexistent-run-id")
	rec := httptest.NewRecorder()
	srv.handleRunLogs(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAuthMiddleware_JWTMode(t *testing.T) {
	secret := "test-jwt-secret-key"
	srv := &Server{
		authMode:  AuthModeJWT,
		jwtSecret: secret,
		bind:      "0.0.0.0",
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := srv.jwtAuthMiddleware(inner)

	// Generate a valid JWT
	claims := JWTClaims{
		Subject:   "test-user",
		IssuedAt:  1000000000,
		ExpiresAt: 9999999999,
	}
	token, err := GenerateJWT(secret, claims)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	// Test with valid JWT — should pass through
	req := httptest.NewRequest("GET", "/api/runs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with valid JWT, got %d", rec.Code)
	}

	// Test without token — should get 401
	req2 := httptest.NewRequest("GET", "/api/runs", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without JWT, got %d", rec2.Code)
	}

	// Test with invalid token — should get 401
	req3 := httptest.NewRequest("GET", "/api/runs", nil)
	req3.Header.Set("Authorization", "Bearer invalid.token.here")
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with invalid JWT, got %d", rec3.Code)
	}
}

func TestAuthMiddleware_NoneMode(t *testing.T) {
	srv := &Server{
		authMode: AuthModeNone,
		bind:     "0.0.0.0",
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := srv.applyMiddleware(inner)

	// Request without any auth should pass through
	req := httptest.NewRequest("GET", "/api/runs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with AuthModeNone (no auth required), got %d", rec.Code)
	}
}
