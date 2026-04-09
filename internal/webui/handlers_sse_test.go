package webui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

// TestHandleSSE_InvalidLastEventID verifies that an invalid (non-numeric)
// Last-Event-ID header is handled gracefully without crashing or returning
// an error. The handler should simply skip the backfill.
func TestHandleSSE_InvalidLastEventID(t *testing.T) {
	srv, _ := testServer(t)
	go srv.broker.Start()
	defer srv.broker.Stop()

	req := httptest.NewRequest("GET", "/api/runs/run-invalid-id/events", nil)
	req.SetPathValue("id", "run-invalid-id")
	req.Header.Set("Last-Event-ID", "not-a-number")

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

	// Handler should still have written the retry directive successfully.
	if !strings.Contains(rec.Body.String(), "retry: 3000") {
		t.Errorf("expected retry directive despite invalid Last-Event-ID, got: %q", rec.Body.String())
	}
	// Should not contain any backfill data since the header was invalid.
	if strings.Contains(rec.Body.String(), "event:") {
		t.Errorf("expected no backfill events for invalid Last-Event-ID, got: %q", rec.Body.String())
	}
}

// TestHandleSSE_NonExistentRunID verifies that subscribing to a run ID that
// does not exist in the database still works. The subscriber connects and
// receives the retry directive but no backfill events.
func TestHandleSSE_NonExistentRunID(t *testing.T) {
	srv, _ := testServer(t)
	go srv.broker.Start()
	defer srv.broker.Stop()

	req := httptest.NewRequest("GET", "/api/runs/nonexistent-run-xyz/events", nil)
	req.SetPathValue("id", "nonexistent-run-xyz")
	// Set a valid Last-Event-ID to trigger backfill code path (which should
	// return zero events for a non-existent run).
	req.Header.Set("Last-Event-ID", "1")

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

	body := rec.Body.String()
	if !strings.Contains(body, "retry: 3000") {
		t.Errorf("expected retry directive for non-existent run, got: %q", body)
	}
	// No event data should be present since the run does not exist.
	if strings.Contains(body, "data:") {
		t.Errorf("expected no event data for non-existent run, got: %q", body)
	}
}

// TestHandleSSE_BrokerCleanupAfterDisconnect verifies that after a client
// disconnects, the broker properly removes the subscriber channel so it
// does not leak.
func TestHandleSSE_BrokerCleanupAfterDisconnect(t *testing.T) {
	srv, _ := testServer(t)
	go srv.broker.Start()
	defer srv.broker.Stop()

	// Record initial subscriber count.
	srv.broker.mu.RLock()
	initialCount := len(srv.broker.clients)
	srv.broker.mu.RUnlock()

	req := httptest.NewRequest("GET", "/api/runs/run-cleanup/events", nil)
	req.SetPathValue("id", "run-cleanup")

	rec := &flusherRecorder{httptest.NewRecorder()}

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		srv.handleSSE(rec, req)
		close(done)
	}()

	// Give the handler a moment to subscribe to the broker.
	time.Sleep(50 * time.Millisecond)

	// Verify the subscriber was added.
	srv.broker.mu.RLock()
	duringCount := len(srv.broker.clients)
	srv.broker.mu.RUnlock()
	if duringCount != initialCount+1 {
		t.Errorf("expected %d subscribers during connection, got %d", initialCount+1, duringCount)
	}

	// Disconnect the client.
	cancel()
	<-done

	// Give the broker a moment to process the unsubscribe.
	time.Sleep(50 * time.Millisecond)

	// Verify the subscriber was removed.
	srv.broker.mu.RLock()
	afterCount := len(srv.broker.clients)
	srv.broker.mu.RUnlock()
	if afterCount != initialCount {
		t.Errorf("expected %d subscribers after disconnect (cleanup), got %d", initialCount, afterCount)
	}
}

// TestHandleSSE_NegativeLastEventID verifies that a negative Last-Event-ID
// value is handled gracefully. Since ParseInt succeeds for negative values,
// but the value is not > 0, no backfill should occur.
func TestHandleSSE_NegativeLastEventID(t *testing.T) {
	srv, _ := testServer(t)
	go srv.broker.Start()
	defer srv.broker.Stop()

	req := httptest.NewRequest("GET", "/api/runs/run-neg-id/events", nil)
	req.SetPathValue("id", "run-neg-id")
	req.Header.Set("Last-Event-ID", "-5")

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

	body := rec.Body.String()
	if !strings.Contains(body, "retry: 3000") {
		t.Errorf("expected retry directive, got: %q", body)
	}
	// No backfill should occur for negative IDs.
	if strings.Contains(body, "data:") {
		t.Errorf("expected no backfill for negative Last-Event-ID, got: %q", body)
	}
}

// TestHandleSSE_EmptyLastEventID verifies that an empty Last-Event-ID header
// (as opposed to absent) is treated the same as no header.
func TestHandleSSE_EmptyLastEventID(t *testing.T) {
	srv, _ := testServer(t)
	go srv.broker.Start()
	defer srv.broker.Stop()

	req := httptest.NewRequest("GET", "/api/runs/run-empty-id/events", nil)
	req.SetPathValue("id", "run-empty-id")
	req.Header.Set("Last-Event-ID", "")

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

	body := rec.Body.String()
	if !strings.Contains(body, "retry: 3000") {
		t.Errorf("expected retry directive, got: %q", body)
	}
}

// TestHandleSSE_OverflowLastEventID verifies that a Last-Event-ID value that
// overflows int64 is handled gracefully (ParseInt fails, so no backfill).
func TestHandleSSE_OverflowLastEventID(t *testing.T) {
	srv, _ := testServer(t)
	go srv.broker.Start()
	defer srv.broker.Stop()

	req := httptest.NewRequest("GET", "/api/runs/run-overflow/events", nil)
	req.SetPathValue("id", "run-overflow")
	// Value that overflows int64
	req.Header.Set("Last-Event-ID", "99999999999999999999999999")

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

	body := rec.Body.String()
	if !strings.Contains(body, "retry: 3000") {
		t.Errorf("expected retry directive despite overflow Last-Event-ID, got: %q", body)
	}
}

// TestMatchesRunID_EmptyRunID verifies that when both the run ID filter and
// the pipeline_id in the payload are empty, matchesRunID returns true because
// strings.Contains(data, "") is always true and the pipeline_id equals the
// filter. This documents the actual behavior.
func TestMatchesRunID_EmptyRunID(t *testing.T) {
	data := `{"pipeline_id":"","step":"build","state":"running"}`
	if !matchesRunID(data, "") {
		t.Errorf("expected matchesRunID to return true when both run ID and pipeline_id are empty")
	}
}

// TestMatchesRunID_EmptyRunIDNoMatch verifies that when the run ID filter is
// empty but the pipeline_id is non-empty, matchesRunID returns false.
func TestMatchesRunID_EmptyRunIDNoMatch(t *testing.T) {
	data := `{"pipeline_id":"abc-123","step":"build","state":"running"}`
	if matchesRunID(data, "") {
		t.Errorf("expected matchesRunID to return false when run ID is empty but pipeline_id is not")
	}
}

// TestMatchesRunID_SpecialCharsInRunID verifies that run IDs containing JSON
// special characters (like double quotes) cause the quick strings.Contains
// check to fail because JSON escapes the quotes. The function returns false
// even though the pipeline_id would match after JSON unmarshalling. This
// documents the limitation of the quick-check optimization.
func TestMatchesRunID_SpecialCharsInRunID(t *testing.T) {
	runID := `run-with-"quotes`
	payload := map[string]string{
		"pipeline_id": runID,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	// The quick strings.Contains check fails because JSON escapes the quotes,
	// so the raw string does not contain the literal run ID.
	if matchesRunID(string(raw), runID) {
		t.Errorf("expected matchesRunID to return false for run ID with JSON-escaped special chars (known limitation)")
	}
}

// TestMatchesRunID_RunIDWithDashes verifies that typical run IDs containing
// dashes and alphanumeric characters are matched correctly.
func TestMatchesRunID_RunIDWithDashes(t *testing.T) {
	runID := "pipeline-run-abc-def-123-456"
	data := `{"pipeline_id":"pipeline-run-abc-def-123-456","state":"running"}`
	if !matchesRunID(data, runID) {
		t.Errorf("expected matchesRunID to return true for run ID with dashes")
	}
}

// TestMatchesRunID_NullJSON verifies that a JSON null payload returns false.
func TestMatchesRunID_NullJSON(t *testing.T) {
	if matchesRunID("null", "some-run") {
		t.Errorf("expected matchesRunID to return false for null JSON")
	}
}
