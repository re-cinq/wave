package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/event"
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
