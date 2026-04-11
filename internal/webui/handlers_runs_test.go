package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
)

// TestEventToSummary verifies that all fields of state.LogRecord are mapped
// correctly to EventSummary.
func TestEventToSummary(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	record := state.LogRecord{
		ID:         42,
		RunID:      "run-abc",
		Timestamp:  ts,
		StepID:     "step-1",
		State:      "running",
		Persona:    "craftsman",
		Message:    "working on it",
		TokensUsed: 512,
		DurationMs: 3200,
	}

	got := eventToSummary(record)

	if got.ID != 42 {
		t.Errorf("ID: expected 42, got %d", got.ID)
	}
	if !got.Timestamp.Equal(ts) {
		t.Errorf("Timestamp: expected %v, got %v", ts, got.Timestamp)
	}
	if got.StepID != "step-1" {
		t.Errorf("StepID: expected %q, got %q", "step-1", got.StepID)
	}
	if got.State != "running" {
		t.Errorf("State: expected %q, got %q", "running", got.State)
	}
	if got.Persona != "craftsman" {
		t.Errorf("Persona: expected %q, got %q", "craftsman", got.Persona)
	}
	if got.Message != "working on it" {
		t.Errorf("Message: expected %q, got %q", "working on it", got.Message)
	}
	if got.TokensUsed != 512 {
		t.Errorf("TokensUsed: expected 512, got %d", got.TokensUsed)
	}
	if got.DurationMs != 3200 {
		t.Errorf("DurationMs: expected 3200, got %d", got.DurationMs)
	}
}

// TestEventToSummary_ZeroValues ensures zero-value fields are passed through without error.
func TestEventToSummary_ZeroValues(t *testing.T) {
	got := eventToSummary(state.LogRecord{})

	if got.ID != 0 {
		t.Errorf("ID: expected 0, got %d", got.ID)
	}
	if got.StepID != "" {
		t.Errorf("StepID: expected empty, got %q", got.StepID)
	}
	if got.TokensUsed != 0 {
		t.Errorf("TokensUsed: expected 0, got %d", got.TokensUsed)
	}
	if got.DurationMs != 0 {
		t.Errorf("DurationMs: expected 0, got %d", got.DurationMs)
	}
}

// TestArtifactToSummary verifies that ID, Name, Path, Type, and SizeBytes are
// mapped correctly from state.ArtifactRecord to ArtifactSummary.
func TestArtifactToSummary(t *testing.T) {
	record := state.ArtifactRecord{
		ID:        99,
		RunID:     "run-xyz",
		StepID:    "step-2",
		Name:      "impl_plan",
		Path:      ".wave/artifacts/impl_plan",
		Type:      "markdown",
		SizeBytes: 4096,
	}

	got := artifactToSummary(record)

	if got.ID != 99 {
		t.Errorf("ID: expected 99, got %d", got.ID)
	}
	if got.Name != "impl_plan" {
		t.Errorf("Name: expected %q, got %q", "impl_plan", got.Name)
	}
	if got.Path != ".wave/artifacts/impl_plan" {
		t.Errorf("Path: expected %q, got %q", ".wave/artifacts/impl_plan", got.Path)
	}
	if got.Type != "markdown" {
		t.Errorf("Type: expected %q, got %q", "markdown", got.Type)
	}
	if got.SizeBytes != 4096 {
		t.Errorf("SizeBytes: expected 4096, got %d", got.SizeBytes)
	}
}

// TestArtifactToSummary_ZeroValues ensures zero-value ArtifactRecord maps cleanly.
func TestArtifactToSummary_ZeroValues(t *testing.T) {
	got := artifactToSummary(state.ArtifactRecord{})

	if got.ID != 0 {
		t.Errorf("ID: expected 0, got %d", got.ID)
	}
	if got.Name != "" {
		t.Errorf("Name: expected empty, got %q", got.Name)
	}
	if got.SizeBytes != 0 {
		t.Errorf("SizeBytes: expected 0, got %d", got.SizeBytes)
	}
}

// TestFormatDurationValue covers the two branches: under one minute and over.
func TestFormatDurationValue(t *testing.T) {
	cases := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "zero",
			duration: 0,
			want:     "0s",
		},
		{
			name:     "one second",
			duration: time.Second,
			want:     "1s",
		},
		{
			name:     "thirty seconds",
			duration: 30 * time.Second,
			want:     "30s",
		},
		{
			name:     "fifty-nine seconds",
			duration: 59 * time.Second,
			want:     "59s",
		},
		{
			name:     "exactly one minute",
			duration: time.Minute,
			want:     "1m",
		},
		{
			name:     "one minute thirty seconds",
			duration: 90 * time.Second,
			want:     "1m 30s",
		},
		{
			name:     "two minutes",
			duration: 2 * time.Minute,
			want:     "2m",
		},
		{
			name:     "ten minutes five seconds",
			duration: 10*time.Minute + 5*time.Second,
			want:     "10m 5s",
		},
		{
			name:     "exactly one hour",
			duration: time.Hour,
			want:     "1h",
		},
		{
			name:     "one hour thirty minutes",
			duration: time.Hour + 30*time.Minute,
			want:     "1h 30m",
		},
		{
			name:     "two hours",
			duration: 2 * time.Hour,
			want:     "2h",
		},
		{
			name:     "three hours fifteen minutes",
			duration: 3*time.Hour + 15*time.Minute,
			want:     "3h 15m",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatDurationValue(tc.duration)
			if got != tc.want {
				t.Errorf("formatDurationValue(%v) = %q, want %q", tc.duration, got, tc.want)
			}
		})
	}
}

// TestHandleRunDetailPage_MissingID verifies that a request without a run ID
// path value returns HTTP 400.
func TestHandleRunDetailPage_MissingID(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/runs/", nil)
	// Deliberately do NOT call req.SetPathValue("id", ...) to simulate missing ID.
	rec := httptest.NewRecorder()
	srv.handleRunDetailPage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing run ID, got %d", rec.Code)
	}
}

// TestHandleRunDetailPage_NotFound verifies that requesting an unknown run ID
// returns HTTP 404.
func TestHandleRunDetailPage_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/runs/does-not-exist", nil)
	req.SetPathValue("id", "does-not-exist")
	rec := httptest.NewRecorder()
	srv.handleRunDetailPage(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown run, got %d", rec.Code)
	}
}

// TestHandleRunDetailPage_ValidRun verifies that a request for a known run
// returns HTTP 200 with HTML content that includes the run ID.
func TestHandleRunDetailPage_ValidRun(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, err := rwStore.CreateRun("test-pipeline", "test input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	req := httptest.NewRequest("GET", "/runs/"+runID, nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleRunDetailPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid run, got %d: %s", rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type: expected %q, got %q", "text/html; charset=utf-8", contentType)
	}

	body := rec.Body.String()
	if body == "" {
		t.Error("expected non-empty HTML body")
	}
	// The stub template renders the run ID inside a div.
	if !strings.Contains(body, runID) {
		t.Errorf("expected body to contain run ID %q, got: %s", runID, body)
	}
}

// TestHandleRunDetailPage_WithPipelineAndEvents exercises the full path through
// handleRunDetailPage including buildStepDetails and DAG layout computation.
func TestHandleRunDetailPage_WithPipelineAndEvents(t *testing.T) {
	srv, rwStore := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}
	pipelineYAML := `kind: Pipeline
metadata:
  name: test-pipeline
steps:
  - id: step1
    persona: navigator
    exec:
      prompt: "plan"
  - id: step2
    persona: craftsman
    depends_on: [step1]
    exec:
      prompt: "implement"
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

	runID, err := rwStore.CreateRun("test-pipeline", "test input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}
	if err := rwStore.UpdateRunStatus(runID, "running", "step1", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	// Log events to exercise buildStepDetails state machine
	if err := rwStore.LogEvent(runID, "step1", "running", "navigator", "Starting", 0, 0, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}
	if err := rwStore.LogEvent(runID, "step1", "completed", "navigator", "Done", 500, 5000, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}
	if err := rwStore.LogEvent(runID, "step2", "running", "craftsman", "Building", 100, 1000, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}
	if err := rwStore.LogEvent(runID, "step2", "failed", "craftsman", "Error occurred", 200, 2000, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	req := httptest.NewRequest("GET", "/runs/"+runID, nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleRunDetailPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestHandleAPIRunDetail_WithEvents tests the API endpoint with events.
func TestHandleAPIRunDetail_WithEvents(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, err := rwStore.CreateRun("test-pipeline", "test input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	if err := rwStore.LogEvent(runID, "step1", "running", "navigator", "Working", 100, 500, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}
	if err := rwStore.LogEvent(runID, "step1", "completed", "navigator", "Done", 200, 1000, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs/"+runID, nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleAPIRunDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp RunDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(resp.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(resp.Events))
	}
}

// TestRunToSummary_CompletedRun tests duration calculation for completed runs.
func TestRunToSummary_CompletedRun(t *testing.T) {
	start := time.Now().Add(-5 * time.Minute)
	end := time.Now()
	run := state.RunRecord{
		RunID:        "run-123",
		PipelineName: "my-pipeline",
		Status:       "completed",
		TotalTokens:  1000,
		StartedAt:    start,
		CompletedAt:  &end,
	}

	summary := runToSummary(run)

	if summary.RunID != "run-123" {
		t.Errorf("expected run ID 'run-123', got %q", summary.RunID)
	}
	if summary.Duration == "" {
		t.Error("expected non-empty duration for completed run")
	}
	if summary.TotalTokens != 1000 {
		t.Errorf("expected 1000 tokens, got %d", summary.TotalTokens)
	}
}

// TestRunToSummary_RunningRun tests duration calculation for running runs.
func TestRunToSummary_RunningRun(t *testing.T) {
	start := time.Now().Add(-30 * time.Second)
	run := state.RunRecord{
		RunID:        "run-456",
		PipelineName: "my-pipeline",
		Status:       "running",
		StartedAt:    start,
	}

	summary := runToSummary(run)

	if summary.Duration == "" {
		t.Error("expected non-empty duration for running run")
	}
}

// TestHandleRunsPage_WithData tests the HTML runs page with pagination data.
func TestHandleRunsPage_WithData(t *testing.T) {
	srv, rwStore := testServer(t)

	for i := 0; i < 3; i++ {
		if _, err := rwStore.CreateRun("test-pipeline", "input"); err != nil {
			t.Fatalf("failed to create run: %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/runs", nil)
	rec := httptest.NewRecorder()
	srv.handleRunsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<html>") {
		t.Error("expected HTML content in response")
	}
}

// TestHandleRunsPage_StatusFilter tests the HTML runs page with status filter.
func TestHandleRunsPage_StatusFilter(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "completed", "", 100); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	req := httptest.NewRequest("GET", "/runs?status=completed", nil)
	rec := httptest.NewRecorder()
	srv.handleRunsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestBuildStepDetails_WithPipeline exercises buildStepDetails directly.
func TestBuildStepDetails_WithPipeline(t *testing.T) {
	srv, rwStore := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}
	pipelineYAML := `kind: Pipeline
metadata:
  name: test-pipeline
steps:
  - id: step1
    persona: navigator
    exec:
      prompt: "plan"
  - id: step2
    persona: craftsman
    depends_on: [step1]
    exec:
      prompt: "implement"
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

	runID, _ := rwStore.CreateRun("test-pipeline", "input")

	// Log events covering all state transitions
	if err := rwStore.LogEvent(runID, "step1", "running", "navigator", "Starting", 0, 0, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}
	if err := rwStore.LogEvent(runID, "step1", "completed", "navigator", "Done", 500, 5000, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}
	if err := rwStore.LogEvent(runID, "step2", "running", "craftsman", "Building", 100, 1000, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	details := srv.buildStepDetails(runID, "test-pipeline")

	if len(details) != 2 {
		t.Fatalf("expected 2 step details, got %d", len(details))
	}

	// step1 should be completed
	if details[0].StepID != "step1" {
		t.Errorf("expected step1, got %q", details[0].StepID)
	}
	if details[0].State != "completed" {
		t.Errorf("expected step1 state 'completed', got %q", details[0].State)
	}
	if details[0].Persona != "navigator" {
		t.Errorf("expected step1 persona 'navigator', got %q", details[0].Persona)
	}
	if details[0].TokensUsed != 500 {
		t.Errorf("expected step1 tokens 500, got %d", details[0].TokensUsed)
	}
	if details[0].Progress != 100 {
		t.Errorf("expected step1 progress 100, got %d", details[0].Progress)
	}

	// step2 should be running
	if details[1].StepID != "step2" {
		t.Errorf("expected step2, got %q", details[1].StepID)
	}
	if details[1].State != "running" {
		t.Errorf("expected step2 state 'running', got %q", details[1].State)
	}
	if details[1].Progress != 50 {
		t.Errorf("expected step2 progress 50, got %d", details[1].Progress)
	}
}

// TestHandleRunsPage_ActivePage verifies that the runs page includes
// nav-link-active on the Runs nav link.
func TestHandleRunsPage_ActivePage(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/runs", nil)
	rec := httptest.NewRecorder()
	srv.handleRunsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "nav-link-active") {
		t.Error("expected nav-link-active class in runs page HTML")
	}
}

// TestHandleRunDetailPage_ActivePage verifies that the run detail page includes
// nav-link-active on the Runs nav link.
func TestHandleRunDetailPage_ActivePage(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, err := rwStore.CreateRun("test-pipeline", "test input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	req := httptest.NewRequest("GET", "/runs/"+runID, nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleRunDetailPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "nav-link-active") {
		t.Error("expected nav-link-active class in run detail page HTML")
	}
}

// TestBuildStepDetails_NoPipeline tests that buildStepDetails returns nil when
// the pipeline YAML doesn't exist.
func TestBuildStepDetails_NoPipeline(t *testing.T) {
	srv, rwStore := testServer(t)

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

	runID, _ := rwStore.CreateRun("nonexistent-pipeline", "input")
	details := srv.buildStepDetails(runID, "nonexistent-pipeline")

	if details != nil {
		t.Errorf("expected nil details for missing pipeline, got %d", len(details))
	}
}

// TestParseLinkedURL tests GitHub issue/PR URL extraction from input strings.
func TestParseLinkedURL(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "github issue URL",
			input: "https://github.com/re-cinq/wave/issues/562",
			want:  "https://github.com/re-cinq/wave/issues/562",
		},
		{
			name:  "github PR URL",
			input: "https://github.com/re-cinq/wave/pull/123",
			want:  "https://github.com/re-cinq/wave/pull/123",
		},
		{
			name:  "URL embedded in text",
			input: "Please review https://github.com/re-cinq/wave/issues/42 and fix it",
			want:  "https://github.com/re-cinq/wave/issues/42",
		},
		{
			name:  "multiple URLs returns first",
			input: "https://github.com/re-cinq/wave/issues/1 and https://github.com/re-cinq/wave/pull/2",
			want:  "https://github.com/re-cinq/wave/issues/1",
		},
		{
			name:  "non-github URL",
			input: "https://gitlab.com/org/repo/issues/5",
			want:  "",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "no URL in text",
			input: "just some plain text without URLs",
			want:  "",
		},
		{
			name:  "github URL without issue or PR path",
			input: "https://github.com/re-cinq/wave",
			want:  "",
		},
		{
			name:  "repo with dots and hyphens",
			input: "https://github.com/my-org/my.project/issues/99",
			want:  "https://github.com/my-org/my.project/issues/99",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseLinkedURL(tc.input)
			if got != tc.want {
				t.Errorf("parseLinkedURL(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestRunToSummary_NewFields verifies that Input, LinkedURL, FormattedStartedAt,
// and FormattedCompletedAt are populated correctly by runToSummary.
func TestRunToSummary_NewFields(t *testing.T) {
	start := time.Date(2026, 3, 25, 14, 30, 0, 0, time.UTC)
	end := time.Date(2026, 3, 25, 14, 35, 0, 0, time.UTC)

	t.Run("with github URL input and completion time", func(t *testing.T) {
		run := state.RunRecord{
			RunID:        "run-new-1",
			PipelineName: "impl-issue",
			Status:       "completed",
			Input:        "https://github.com/re-cinq/wave/issues/562",
			StartedAt:    start,
			CompletedAt:  &end,
			BranchName:   "562-stats-card",
		}
		summary := runToSummary(run)

		if summary.Input != run.Input {
			t.Errorf("Input: expected %q, got %q", run.Input, summary.Input)
		}
		if summary.LinkedURL != "https://github.com/re-cinq/wave/issues/562" {
			t.Errorf("LinkedURL: expected GitHub URL, got %q", summary.LinkedURL)
		}
		if summary.FormattedStartedAt == "" {
			t.Error("FormattedStartedAt: expected non-empty")
		}
		if summary.FormattedCompletedAt == "" {
			t.Error("FormattedCompletedAt: expected non-empty for completed run")
		}
		if summary.BranchName != "562-stats-card" {
			t.Errorf("BranchName: expected %q, got %q", "562-stats-card", summary.BranchName)
		}
	})

	t.Run("without completion time", func(t *testing.T) {
		run := state.RunRecord{
			RunID:        "run-new-2",
			PipelineName: "impl-issue",
			Status:       "running",
			Input:        "some plain text input",
			StartedAt:    start,
		}
		summary := runToSummary(run)

		if summary.Input != "some plain text input" {
			t.Errorf("Input: expected %q, got %q", "some plain text input", summary.Input)
		}
		if summary.LinkedURL != "" {
			t.Errorf("LinkedURL: expected empty for non-URL input, got %q", summary.LinkedURL)
		}
		if summary.FormattedStartedAt == "" {
			t.Error("FormattedStartedAt: expected non-empty")
		}
		if summary.FormattedCompletedAt != "" {
			t.Errorf("FormattedCompletedAt: expected empty for running run, got %q", summary.FormattedCompletedAt)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		run := state.RunRecord{
			RunID:        "run-new-3",
			PipelineName: "impl-issue",
			Status:       "pending",
			StartedAt:    start,
		}
		summary := runToSummary(run)

		if summary.Input != "" {
			t.Errorf("Input: expected empty, got %q", summary.Input)
		}
		if summary.LinkedURL != "" {
			t.Errorf("LinkedURL: expected empty, got %q", summary.LinkedURL)
		}
		if summary.InputPreview != "" {
			t.Errorf("InputPreview: expected empty, got %q", summary.InputPreview)
		}
	})

	t.Run("long input gets truncated preview", func(t *testing.T) {
		longInput := strings.Repeat("a", 100)
		run := state.RunRecord{
			RunID:        "run-new-4",
			PipelineName: "impl-issue",
			Status:       "completed",
			Input:        longInput,
			StartedAt:    start,
			CompletedAt:  &end,
		}
		summary := runToSummary(run)

		if summary.Input != longInput {
			t.Error("Input: expected full input text")
		}
		if len(summary.InputPreview) > 84 { // 80 chars + "..."
			t.Errorf("InputPreview: expected truncated, got length %d", len(summary.InputPreview))
		}
		if !strings.HasSuffix(summary.InputPreview, "...") {
			t.Error("InputPreview: expected to end with '...'")
		}
	})
}

// TestBuildStepDetails_GateChoicesData verifies that GateChoicesData and
// GateFreeform are populated from pipeline gate configuration.
func TestBuildStepDetails_GateChoicesData(t *testing.T) {
	srv, rwStore := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}
	pipelineYAML := `kind: Pipeline
metadata:
  name: gate-pipeline
steps:
  - id: review-gate
    type: gate
    gate:
      type: approval
      prompt: "Approve this change?"
      freeform: true
      choices:
        - key: approve
          label: Approve
          target: next-step
        - key: reject
          label: Reject
          target: _fail
  - id: next-step
    persona: craftsman
    depends_on: [review-gate]
    exec:
      prompt: "implement"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "gate-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
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

	runID, _ := rwStore.CreateRun("gate-pipeline", "input")

	// Log the gate step as running so the interactive panel would render
	if err := rwStore.LogEvent(runID, "review-gate", "running", "", "Waiting for approval", 0, 0, "", "", ""); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	details := srv.buildStepDetails(runID, "gate-pipeline")

	if len(details) != 2 {
		t.Fatalf("expected 2 step details, got %d", len(details))
	}

	gate := details[0]
	if gate.StepID != "review-gate" {
		t.Errorf("expected step ID 'review-gate', got %q", gate.StepID)
	}
	if gate.StepType != "gate" {
		t.Errorf("expected step type 'gate', got %q", gate.StepType)
	}
	if gate.GatePrompt != "Approve this change?" {
		t.Errorf("expected gate prompt 'Approve this change?', got %q", gate.GatePrompt)
	}
	if !gate.GateFreeform {
		t.Error("expected GateFreeform to be true")
	}
	if len(gate.GateChoicesData) != 2 {
		t.Fatalf("expected 2 gate choices, got %d", len(gate.GateChoicesData))
	}
	if gate.GateChoicesData[0].Key != "approve" {
		t.Errorf("expected first choice key 'approve', got %q", gate.GateChoicesData[0].Key)
	}
	if gate.GateChoicesData[0].Label != "Approve" {
		t.Errorf("expected first choice label 'Approve', got %q", gate.GateChoicesData[0].Label)
	}
	if gate.GateChoicesData[0].Target != "next-step" {
		t.Errorf("expected first choice target 'next-step', got %q", gate.GateChoicesData[0].Target)
	}
	if gate.GateChoicesData[1].Key != "reject" {
		t.Errorf("expected second choice key 'reject', got %q", gate.GateChoicesData[1].Key)
	}
	if gate.GateChoicesData[1].Target != "_fail" {
		t.Errorf("expected second choice target '_fail', got %q", gate.GateChoicesData[1].Target)
	}
	if gate.GateChoices != "Approve, Reject" {
		t.Errorf("expected GateChoices 'Approve, Reject', got %q", gate.GateChoices)
	}

	// Non-gate step should have nil GateChoicesData
	impl := details[1]
	if impl.GateChoicesData != nil {
		t.Errorf("expected nil GateChoicesData for non-gate step, got %v", impl.GateChoicesData)
	}
	if impl.GateFreeform {
		t.Error("expected GateFreeform to be false for non-gate step")
	}
}

// TestHandleRunsPage_RunningSection_Populated verifies that a running run
// appears in the rp-section with the correct count badge and a link to the run.
func TestHandleRunsPage_RunningSection_Populated(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}
	if err := rwStore.UpdateRunStatus(runID, "running", "", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	req := httptest.NewRequest("GET", "/runs", nil)
	rec := httptest.NewRecorder()
	srv.handleRunsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "rp-section") {
		t.Error("expected rp-section in response body")
	}
	if !strings.Contains(body, "rp-badge") {
		t.Error("expected rp-badge in response body")
	}
	if !strings.Contains(body, `href="/runs/`+runID+`"`) {
		t.Errorf("expected link to run %q in response body", runID)
	}
}

// TestHandleRunsPage_RunningSection_Empty verifies that the empty-state CTA
// is shown when no pipelines are running.
func TestHandleRunsPage_RunningSection_Empty(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create a completed run — should not appear in running section
	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "completed", "", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	req := httptest.NewRequest("GET", "/runs", nil)
	rec := httptest.NewRecorder()
	srv.handleRunsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "rp-empty") {
		t.Error("expected rp-empty class when no running pipelines")
	}
	if !strings.Contains(body, `href="/pipelines"`) {
		t.Error("expected CTA link to /pipelines in empty state")
	}
}

// TestHandleRunsPage_RunningSection_FilterRespected verifies that the pipeline
// filter query parameter is applied to the running section (FR-008).
func TestHandleRunsPage_RunningSection_FilterRespected(t *testing.T) {
	srv, rwStore := testServer(t)

	run1ID, err := rwStore.CreateRun("pipeline-alpha", "input")
	if err != nil {
		t.Fatalf("failed to create run1: %v", err)
	}
	if err := rwStore.UpdateRunStatus(run1ID, "running", "", 0); err != nil {
		t.Fatalf("failed to update run1 status: %v", err)
	}

	run2ID, err := rwStore.CreateRun("pipeline-beta", "input")
	if err != nil {
		t.Fatalf("failed to create run2: %v", err)
	}
	if err := rwStore.UpdateRunStatus(run2ID, "running", "", 0); err != nil {
		t.Fatalf("failed to update run2 status: %v", err)
	}

	req := httptest.NewRequest("GET", "/runs?pipeline=pipeline-alpha", nil)
	rec := httptest.NewRecorder()
	srv.handleRunsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, `href="/runs/`+run1ID+`"`) {
		t.Errorf("expected link to run1 %q for pipeline-alpha filter", run1ID)
	}
	if strings.Contains(body, `href="/runs/`+run2ID+`"`) {
		t.Errorf("did not expect link to run2 %q when filtered to pipeline-alpha", run2ID)
	}
}

func TestFormatSmartTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"today", time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location()), "12:00"},
		{"same year", time.Date(now.Year(), 1, 15, 10, 30, 0, 0, time.Local), "Jan 15 10:30"},
		{"different year", time.Date(2024, 6, 15, 10, 30, 0, 0, time.Local), "Jun 15, 2024"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSmartTime(tt.t)
			if got != tt.want {
				t.Errorf("formatSmartTime() = %q, want %q", got, tt.want)
			}
		})
	}
}
