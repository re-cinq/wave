package webui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/recinq/wave/internal/runner"
)

func TestGateRegistry_RegisterAndResolve(t *testing.T) {
	g := NewGateRegistry()

	gate := &runner.WebUIGate{
		Type: "approval",
		Choices: []runner.WebUIGateChoice{
			{Key: "a", Label: "Approve", Target: "next"},
		},
	}

	ch := g.Register("run1", "step1", gate)

	// Channel should not have a decision yet
	select {
	case <-ch:
		t.Fatal("channel should not have a decision before resolve")
	default:
	}

	// Resolve with a decision
	decision := &runner.WebUIGateDecision{Choice: "a", Label: "Approve", Target: "next"}
	if err := g.Resolve("run1", decision); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Channel should now have the decision
	select {
	case d := <-ch:
		if d.Choice != "a" {
			t.Errorf("expected choice 'a', got %q", d.Choice)
		}
	default:
		t.Error("channel should have a decision after resolve")
	}
}

func TestGateRegistry_ResolveNonExistent(t *testing.T) {
	g := NewGateRegistry()

	decision := &runner.WebUIGateDecision{Choice: "a"}
	err := g.Resolve("run1", decision)
	if err == nil {
		t.Error("expected error for non-existent gate")
	}
}

func TestGateRegistry_DoubleResolve(t *testing.T) {
	g := NewGateRegistry()
	gate := &runner.WebUIGate{
		Type:    "approval",
		Choices: []runner.WebUIGateChoice{{Key: "a", Label: "Approve"}},
	}
	g.Register("run1", "step1", gate)

	decision := &runner.WebUIGateDecision{Choice: "a", Label: "Approve"}
	if err := g.Resolve("run1", decision); err != nil {
		t.Fatalf("first resolve should succeed, got %v", err)
	}

	err := g.Resolve("run1", decision)
	if err == nil {
		t.Error("second resolve should return error (already resolved)")
	}
}

func TestGateRegistry_Remove(t *testing.T) {
	g := NewGateRegistry()
	gate := &runner.WebUIGate{
		Type:    "approval",
		Choices: []runner.WebUIGateChoice{{Key: "a", Label: "Approve"}},
	}
	g.Register("run1", "step1", gate)

	g.Remove("run1")

	decision := &runner.WebUIGateDecision{Choice: "a"}
	err := g.Resolve("run1", decision)
	if err == nil {
		t.Error("resolve after remove should return error")
	}
}

func TestGateRegistry_GetPending(t *testing.T) {
	g := NewGateRegistry()

	// No pending gate
	if got := g.GetPending("run1"); got != nil {
		t.Error("expected nil for no pending gate")
	}

	gate := &runner.WebUIGate{
		Type:    "approval",
		Choices: []runner.WebUIGateChoice{{Key: "a", Label: "Approve"}},
	}
	g.Register("run1", "step1", gate)

	got := g.GetPending("run1")
	if got == nil {
		t.Fatal("expected pending gate, got nil")
	}
	if got.Type != "approval" {
		t.Errorf("expected type 'approval', got %q", got.Type)
	}
}

func TestGateRegistry_GetPendingStepID(t *testing.T) {
	g := NewGateRegistry()

	if got := g.GetPendingStepID("run1"); got != "" {
		t.Errorf("expected empty string for no pending gate, got %q", got)
	}

	gate := &runner.WebUIGate{Type: "approval"}
	g.Register("run1", "review", gate)

	if got := g.GetPendingStepID("run1"); got != "review" {
		t.Errorf("expected 'review', got %q", got)
	}
}

func TestHandleGateApprove_MissingCSRFHeader(t *testing.T) {
	srv, _ := testServer(t)

	gate := &runner.WebUIGate{
		Type:    "approval",
		Choices: []runner.WebUIGateChoice{{Key: "a", Label: "Approve"}},
	}
	srv.realtime.gateRegistry.Register("run-123", "review", gate)

	body, _ := json.Marshal(GateApproveRequest{Choice: "a"})
	req := httptest.NewRequest("POST", "/api/runs/run-123/gates/review/approve", bytes.NewReader(body))
	req.SetPathValue("id", "run-123")
	req.SetPathValue("step", "review")
	// Deliberately omit X-Wave-Request header
	rec := httptest.NewRecorder()

	srv.handleGateApprove(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without CSRF header, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGateApprove_Success(t *testing.T) {
	srv, _ := testServer(t)

	gate := &runner.WebUIGate{
		Type: "approval",
		Choices: []runner.WebUIGateChoice{
			{Key: "a", Label: "Approve", Target: "implement"},
			{Key: "r", Label: "Reject", Target: "_fail"},
		},
	}
	ch := srv.realtime.gateRegistry.Register("run-123", "review", gate)

	body, _ := json.Marshal(GateApproveRequest{Choice: "a"})
	req := httptest.NewRequest("POST", "/api/runs/run-123/gates/review/approve", bytes.NewReader(body))
	req.Header.Set("X-Wave-Request", "1")
	req.SetPathValue("id", "run-123")
	req.SetPathValue("step", "review")
	rec := httptest.NewRecorder()

	srv.handleGateApprove(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp GateApproveResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Choice != "a" {
		t.Errorf("expected choice 'a', got %q", resp.Choice)
	}
	if resp.Label != "Approve" {
		t.Errorf("expected label 'Approve', got %q", resp.Label)
	}

	// The channel should have received the decision
	select {
	case d := <-ch:
		if d.Choice != "a" || d.Target != "implement" {
			t.Errorf("unexpected decision: %+v", d)
		}
	default:
		t.Error("expected decision on channel")
	}
}

func TestHandleGateApprove_NotFound(t *testing.T) {
	srv, _ := testServer(t)

	body, _ := json.Marshal(GateApproveRequest{Choice: "a"})
	req := httptest.NewRequest("POST", "/api/runs/run-123/gates/review/approve", bytes.NewReader(body))
	req.Header.Set("X-Wave-Request", "1")
	req.SetPathValue("id", "run-123")
	req.SetPathValue("step", "review")
	rec := httptest.NewRecorder()

	srv.handleGateApprove(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGateApprove_NoGateRegistry(t *testing.T) {
	srv, _ := testServer(t)
	srv.realtime.gateRegistry = nil // simulate uninitialized registry

	body, _ := json.Marshal(GateApproveRequest{Choice: "a"})
	req := httptest.NewRequest("POST", "/api/runs/run-123/gates/review/approve", bytes.NewReader(body))
	req.Header.Set("X-Wave-Request", "1")
	req.SetPathValue("id", "run-123")
	req.SetPathValue("step", "review")
	rec := httptest.NewRecorder()

	srv.handleGateApprove(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGateApprove_StepMismatch(t *testing.T) {
	srv, _ := testServer(t)

	gate := &runner.WebUIGate{
		Type:    "approval",
		Choices: []runner.WebUIGateChoice{{Key: "a", Label: "Approve"}},
	}
	srv.realtime.gateRegistry.Register("run-123", "review", gate)

	body, _ := json.Marshal(GateApproveRequest{Choice: "a"})
	req := httptest.NewRequest("POST", "/api/runs/run-123/gates/wrong-step/approve", bytes.NewReader(body))
	req.Header.Set("X-Wave-Request", "1")
	req.SetPathValue("id", "run-123")
	req.SetPathValue("step", "wrong-step")
	rec := httptest.NewRecorder()

	srv.handleGateApprove(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGateApprove_InvalidChoice(t *testing.T) {
	srv, _ := testServer(t)

	gate := &runner.WebUIGate{
		Type:    "approval",
		Choices: []runner.WebUIGateChoice{{Key: "a", Label: "Approve"}},
	}
	srv.realtime.gateRegistry.Register("run-123", "review", gate)

	body, _ := json.Marshal(GateApproveRequest{Choice: "z"})
	req := httptest.NewRequest("POST", "/api/runs/run-123/gates/review/approve", bytes.NewReader(body))
	req.Header.Set("X-Wave-Request", "1")
	req.SetPathValue("id", "run-123")
	req.SetPathValue("step", "review")
	rec := httptest.NewRecorder()

	srv.handleGateApprove(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGateApprove_MissingChoice(t *testing.T) {
	srv, _ := testServer(t)

	body, _ := json.Marshal(GateApproveRequest{})
	req := httptest.NewRequest("POST", "/api/runs/run-123/gates/review/approve", bytes.NewReader(body))
	req.Header.Set("X-Wave-Request", "1")
	req.SetPathValue("id", "run-123")
	req.SetPathValue("step", "review")
	rec := httptest.NewRecorder()

	srv.handleGateApprove(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
