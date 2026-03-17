package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
