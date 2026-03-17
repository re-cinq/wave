package webui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMatchesRunID_Match verifies that a valid JSON payload with a matching
// pipeline_id returns true.
func TestMatchesRunID_Match(t *testing.T) {
	runID := "abc-123"
	data := `{"pipeline_id":"abc-123","step":"build","state":"running"}`
	if !matchesRunID(data, runID) {
		t.Errorf("expected matchesRunID to return true for matching run ID")
	}
}

// TestMatchesRunID_NoMatch verifies that a valid JSON payload whose pipeline_id
// differs from the requested run ID returns false.
func TestMatchesRunID_NoMatch(t *testing.T) {
	runID := "abc-123"
	data := `{"pipeline_id":"xyz-999","step":"build","state":"running"}`
	if matchesRunID(data, runID) {
		t.Errorf("expected matchesRunID to return false for non-matching run ID")
	}
}

// TestMatchesRunID_InvalidJSON verifies that invalid JSON input returns false
// even when the raw string contains the run ID as a substring.
func TestMatchesRunID_InvalidJSON(t *testing.T) {
	runID := "abc-123"
	// Contains the run ID but is not valid JSON.
	data := `not-json abc-123 content`
	if matchesRunID(data, runID) {
		t.Errorf("expected matchesRunID to return false for invalid JSON")
	}
}

// TestMatchesRunID_EmptyData verifies that an empty data string returns false.
func TestMatchesRunID_EmptyData(t *testing.T) {
	runID := "abc-123"
	if matchesRunID("", runID) {
		t.Errorf("expected matchesRunID to return false for empty data")
	}
}

// TestMatchesRunID_SubstringNotPipelineID verifies that when the run ID appears
// as a substring inside a different JSON field value, the function still returns
// false because pipeline_id does not equal the run ID.
func TestMatchesRunID_SubstringNotPipelineID(t *testing.T) {
	runID := "abc-123"
	// run ID appears in "message" but pipeline_id is different.
	payload := map[string]string{
		"pipeline_id": "other-pipeline",
		"message":     "event for run abc-123 completed",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	if matchesRunID(string(raw), runID) {
		t.Errorf("expected matchesRunID to return false when runID is a substring of another field, not pipeline_id")
	}
}

// TestHandleSSE_Headers verifies that the SSE handler sets the correct
// response headers: Content-Type, Cache-Control, and Connection.
func TestHandleSSE_Headers(t *testing.T) {
	srv, _ := testServer(t)
	go srv.broker.Start()
	defer srv.broker.Stop()

	req := httptest.NewRequest("GET", "/api/runs/run-001/events", nil)
	req.SetPathValue("id", "run-001")

	rec := &flusherRecorder{httptest.NewRecorder()}

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		srv.handleSSE(rec, req)
		close(done)
	}()

	cancel()
	<-done

	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %q", cc)
	}
	if conn := rec.Header().Get("Connection"); conn != "keep-alive" {
		t.Errorf("expected Connection keep-alive, got %q", conn)
	}
}

// TestHandleSSE_MissingRunID verifies that a request without a run ID path
// value results in a 400 Bad Request response.
func TestHandleSSE_MissingRunID(t *testing.T) {
	srv, _ := testServer(t)

	// Do not set the "id" path value so it defaults to the empty string.
	req := httptest.NewRequest("GET", "/api/runs//events", nil)
	rec := httptest.NewRecorder()

	srv.handleSSE(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for missing run ID, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing run ID") {
		t.Errorf("expected body to contain 'missing run ID', got %q", rec.Body.String())
	}
}

// TestHandleSSE_ClientDisconnect verifies that the SSE handler exits cleanly
// when the client disconnects via context cancellation and does not block.
func TestHandleSSE_ClientDisconnect(t *testing.T) {
	srv, _ := testServer(t)
	go srv.broker.Start()
	defer srv.broker.Stop()

	req := httptest.NewRequest("GET", "/api/runs/run-disconnect/events", nil)
	req.SetPathValue("id", "run-disconnect")

	rec := &flusherRecorder{httptest.NewRecorder()}

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		srv.handleSSE(rec, req)
		close(done)
	}()

	// Cancel the context to simulate client disconnect.
	cancel()

	select {
	case <-done:
		// Handler exited cleanly — expected behaviour.
	default:
		// Block until done; the test will time out if the handler hangs.
		<-done
	}

	// The retry directive must still have been written before the handler blocked.
	if !strings.Contains(rec.Body.String(), "retry: 3000") {
		t.Errorf("expected retry directive before disconnect, got: %q", rec.Body.String())
	}
}
