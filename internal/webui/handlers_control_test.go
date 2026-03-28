package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/state"
)

func TestHandleStartPipeline_MissingName(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`{"input":"test"}`)
	req := httptest.NewRequest("POST", "/api/pipelines//start", body)
	// No path value set for "name"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleStartPipeline(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing pipeline name, got %d", rec.Code)
	}
}

func TestHandleStartPipeline_InvalidBody(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/api/pipelines/test/start", body)
	req.SetPathValue("name", "test")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleStartPipeline(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCancelRun_MissingID(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("POST", "/api/runs//cancel", nil)
	rec := httptest.NewRecorder()
	srv.handleCancelRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing run ID, got %d", rec.Code)
	}
}

func TestHandleCancelRun_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("POST", "/api/runs/nonexistent/cancel", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleCancelRun(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent run, got %d", rec.Code)
	}
}

func TestHandleRetryRun_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("POST", "/api/runs/nonexistent/retry", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleRetryRun(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent run, got %d", rec.Code)
	}
}

func TestHandleRetryRun_MissingID(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("POST", "/api/runs//retry", nil)
	rec := httptest.NewRecorder()
	srv.handleRetryRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing run ID, got %d", rec.Code)
	}
}

func TestHandleResumeRun_MissingID(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`{"from_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs//resume", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleResumeRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing run ID, got %d", rec.Code)
	}
}

func TestHandleResumeRun_MissingFromStep(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "failed", "", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	body := strings.NewReader(`{"from_step":""}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/resume", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleResumeRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing from_step, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleResumeRun_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`{"from_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs/nonexistent/resume", body)
	req.SetPathValue("id", "nonexistent")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleResumeRun(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent run, got %d", rec.Code)
	}
}

func TestHandleResumeRun_InvalidBody(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/api/runs/some-id/resume", body)
	req.SetPathValue("id", "some-id")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleResumeRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCancelRun_WrongState(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "completed", "", 100); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/cancel", strings.NewReader(`{}`))
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleCancelRun(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for completed run, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRetryRun_WrongState(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "running", "", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/retry", nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleRetryRun(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for running run, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleResumeRun_CompletedState(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "completed", "", 100); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	body := strings.NewReader(`{"from_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/resume", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleResumeRun(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for completed run, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCancelRun_WithBody(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "running", "", 0); err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	body := strings.NewReader(`{"force":true}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/cancel", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleCancelRun(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp CancelRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.RunID != runID {
		t.Errorf("expected run ID %q, got %q", runID, resp.RunID)
	}
	if resp.Status != "cancelling" {
		t.Errorf("expected status 'cancelling', got %q", resp.Status)
	}
}

// --- loadPipelineYAML path traversal tests ---

func TestLoadPipelineYAML_PathTraversal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "directory traversal", input: "../../etc/passwd", wantErr: "invalid pipeline name"},
		{name: "absolute path", input: "/etc/passwd", wantErr: "invalid pipeline name"},
		{name: "dot-dot slash", input: "../secret", wantErr: "invalid pipeline name"},
		{name: "empty name", input: "", wantErr: "invalid pipeline name"},
		{name: "spaces", input: "my pipeline", wantErr: "invalid pipeline name"},
		{name: "null byte", input: "test\x00evil", wantErr: "invalid pipeline name"},
		{name: "starts with dot", input: ".hidden", wantErr: "invalid pipeline name"},
		{name: "starts with hyphen", input: "-flag", wantErr: "invalid pipeline name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadPipelineYAML(tt.input)
			if err == nil {
				t.Fatal("expected error for malicious pipeline name, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestLoadPipelineYAML_ValidName(t *testing.T) {
	// Set up a temp dir with a pipeline YAML
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	if err := os.MkdirAll(".wave/pipelines", 0o755); err != nil {
		t.Fatal(err)
	}
	pipelineYAML := `kind: pipeline
metadata:
  name: test-pipeline
steps:
  - id: step1
    persona: navigator
    prompt: "do something"
`
	if err := os.WriteFile(".wave/pipelines/test-pipeline.yaml", []byte(pipelineYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := loadPipelineYAML("test-pipeline")
	if err != nil {
		t.Fatalf("expected no error for valid pipeline name, got: %v", err)
	}
	if len(p.Steps) != 1 || p.Steps[0].ID != "step1" {
		t.Errorf("unexpected pipeline structure: %+v", p)
	}
}

// --- Fork handler tests ---

func TestHandleForkRun_MissingID(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`{"from_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs//fork", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleForkRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing run ID, got %d", rec.Code)
	}
}

func TestHandleForkRun_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`{"from_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs/nonexistent/fork", body)
	req.SetPathValue("id", "nonexistent")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleForkRun(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent run, got %d", rec.Code)
	}
}

func TestHandleForkRun_MissingFromStep(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "failed", "", 0); err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"from_step":""}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/fork", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleForkRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing from_step, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleForkRun_RunningRun(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "running", "", 0); err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"from_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/fork", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleForkRun(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for running run, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleForkRun_InvalidBody(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/api/runs/some-id/fork", body)
	req.SetPathValue("id", "some-id")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleForkRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- Rewind handler tests ---

func TestHandleRewindRun_MissingID(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`{"to_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs//rewind", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleRewindRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing run ID, got %d", rec.Code)
	}
}

func TestHandleRewindRun_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`{"to_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs/nonexistent/rewind", body)
	req.SetPathValue("id", "nonexistent")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleRewindRun(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent run, got %d", rec.Code)
	}
}

func TestHandleRewindRun_MissingToStep(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "failed", "", 0); err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"to_step":""}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/rewind", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleRewindRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing to_step, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRewindRun_RunningRun(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "running", "", 0); err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"to_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/rewind", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleRewindRun(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for running run, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRewindRun_InvalidBody(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/api/runs/some-id/rewind", body)
	req.SetPathValue("id", "some-id")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleRewindRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d: %s", rec.Code, rec.Body.String())
	}
}

// setupPipelineDir creates a temporary .wave/pipelines/ directory with a test pipeline
// and changes the working directory there. Returns a cleanup function.
func setupPipelineDir(t *testing.T, pipelineName string, steps []string) {
	t.Helper()
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	if err := os.MkdirAll(".wave/pipelines", 0o755); err != nil {
		t.Fatal(err)
	}

	var stepYAML string
	for _, s := range steps {
		stepYAML += "  - id: " + s + "\n    persona: navigator\n    prompt: \"do\"\n"
	}

	yaml := "kind: pipeline\nmetadata:\n  name: " + pipelineName + "\nsteps:\n" + stepYAML
	if err := os.WriteFile(filepath.Join(".wave/pipelines", pipelineName+".yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHandleRewindRun_Success(t *testing.T) {
	srv, rwStore := testServer(t)
	setupPipelineDir(t, "test-pipeline", []string{"step1", "step2", "step3"})

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "failed", "step3 error", 100); err != nil {
		t.Fatal(err)
	}

	// Save checkpoints so there is something to delete
	for i, stepID := range []string{"step1", "step2", "step3"} {
		if err := rwStore.SaveCheckpoint(&state.CheckpointRecord{
			RunID:     runID,
			StepID:    stepID,
			StepIndex: i,
		}); err != nil {
			t.Fatal(err)
		}
	}

	body := strings.NewReader(`{"to_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/rewind", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleRewindRun(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp RewindRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.RunID != runID {
		t.Errorf("expected run ID %q, got %q", runID, resp.RunID)
	}
	if resp.ToStep != "step1" {
		t.Errorf("expected to_step 'step1', got %q", resp.ToStep)
	}
	if resp.Status != "failed" {
		t.Errorf("expected status 'failed' (aligned with DB), got %q", resp.Status)
	}
	if len(resp.StepsDeleted) != 2 {
		t.Errorf("expected 2 steps deleted (step2, step3), got %d: %v", len(resp.StepsDeleted), resp.StepsDeleted)
	}
}

func TestHandleRewindRun_NothingToRewind(t *testing.T) {
	srv, rwStore := testServer(t)
	setupPipelineDir(t, "test-pipeline", []string{"step1"})

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "failed", "error", 0); err != nil {
		t.Fatal(err)
	}

	// Rewind to the last step - nothing after it
	body := strings.NewReader(`{"to_step":"step1"}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/rewind", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleRewindRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for nothing to rewind, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRewindRun_StepNotFound(t *testing.T) {
	srv, rwStore := testServer(t)
	setupPipelineDir(t, "test-pipeline", []string{"step1", "step2"})

	runID, _ := rwStore.CreateRun("test-pipeline", "input")
	if err := rwStore.UpdateRunStatus(runID, "failed", "error", 0); err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"to_step":"nonexistent-step"}`)
	req := httptest.NewRequest("POST", "/api/runs/"+runID+"/rewind", body)
	req.SetPathValue("id", runID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleRewindRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for step not found, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- Fork Points handler tests ---

func TestHandleForkPoints_MissingID(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs//fork-points", nil)
	rec := httptest.NewRecorder()
	srv.handleForkPoints(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing run ID, got %d", rec.Code)
	}
}

func TestHandleForkPoints_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs/nonexistent/fork-points", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleForkPoints(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent run, got %d", rec.Code)
	}
}

func TestHandleForkPoints_Empty(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/fork-points", nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleForkPoints(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ForkPointsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.RunID != runID {
		t.Errorf("expected run ID %q, got %q", runID, resp.RunID)
	}
	if len(resp.ForkPoints) != 0 {
		t.Errorf("expected 0 fork points, got %d", len(resp.ForkPoints))
	}
}

func TestHandleForkPoints_WithCheckpoints(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, _ := rwStore.CreateRun("test-pipeline", "input")

	// Save checkpoints
	for i, stepID := range []string{"step1", "step2"} {
		if err := rwStore.SaveCheckpoint(&state.CheckpointRecord{
			RunID:     runID,
			StepID:    stepID,
			StepIndex: i,
		}); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/fork-points", nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleForkPoints(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ForkPointsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.ForkPoints) != 2 {
		t.Errorf("expected 2 fork points, got %d", len(resp.ForkPoints))
	}
}

func TestIsHeartbeat(t *testing.T) {
	tests := []struct {
		name     string
		ev       event.Event
		expected bool
	}{
		{
			name:     "empty progress event is heartbeat",
			ev:       event.Event{State: "step_progress", Message: "", TokensUsed: 0, DurationMs: 0},
			expected: true,
		},
		{
			name:     "empty stream_activity is heartbeat",
			ev:       event.Event{State: "stream_activity", Message: "", TokensUsed: 0, DurationMs: 0},
			expected: true,
		},
		{
			name:     "event with message is not heartbeat",
			ev:       event.Event{State: "step_progress", Message: "processing", TokensUsed: 0, DurationMs: 0},
			expected: false,
		},
		{
			name:     "event with tokens is not heartbeat",
			ev:       event.Event{State: "stream_activity", Message: "", TokensUsed: 100, DurationMs: 0},
			expected: false,
		},
		{
			name:     "non-progress event is not heartbeat",
			ev:       event.Event{State: "step_completed", Message: "", TokensUsed: 0, DurationMs: 0},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHeartbeat(tt.ev)
			if got != tt.expected {
				t.Errorf("isHeartbeat() = %v, want %v", got, tt.expected)
			}
		})
	}
}
