package webui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGateRegistry_RegisterAndResolve(t *testing.T) {
	g := NewGateRegistry()

	ch := g.Register("run1", "step1")

	// Channel should not be closed yet
	select {
	case <-ch:
		t.Fatal("channel should not be closed before resolve")
	default:
	}

	// Resolve
	ok := g.Resolve("run1", "step1")
	if !ok {
		t.Error("expected resolve to return true")
	}

	// Channel should now be closed
	select {
	case <-ch:
		// good
	default:
		t.Error("channel should be closed after resolve")
	}
}

func TestGateRegistry_ResolveNonExistent(t *testing.T) {
	g := NewGateRegistry()

	ok := g.Resolve("run1", "step1")
	if ok {
		t.Error("expected resolve to return false for non-existent gate")
	}
}

func TestGateRegistry_DoubleResolve(t *testing.T) {
	g := NewGateRegistry()
	g.Register("run1", "step1")

	ok1 := g.Resolve("run1", "step1")
	if !ok1 {
		t.Error("first resolve should return true")
	}

	ok2 := g.Resolve("run1", "step1")
	if ok2 {
		t.Error("second resolve should return false (already resolved)")
	}
}

func TestGateRegistry_Cleanup(t *testing.T) {
	g := NewGateRegistry()
	g.Register("run1", "step1")

	g.Cleanup("run1", "step1")

	ok := g.Resolve("run1", "step1")
	if ok {
		t.Error("resolve after cleanup should return false")
	}
}

func TestHandleResolveGate_Success(t *testing.T) {
	srv, _ := testServer(t)
	srv.gates = NewGateRegistry()
	srv.gates.Register("run-123", "approve")

	req := httptest.NewRequest("POST", "/api/runs/run-123/gate/approve", nil)
	req.SetPathValue("id", "run-123")
	req.SetPathValue("step", "approve")
	rec := httptest.NewRecorder()

	srv.handleResolveGate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleResolveGate_NotFound(t *testing.T) {
	srv, _ := testServer(t)
	srv.gates = NewGateRegistry()

	req := httptest.NewRequest("POST", "/api/runs/run-123/gate/approve", nil)
	req.SetPathValue("id", "run-123")
	req.SetPathValue("step", "approve")
	rec := httptest.NewRecorder()

	srv.handleResolveGate(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleResolveGate_NoGateRegistry(t *testing.T) {
	srv, _ := testServer(t)
	// srv.gates is nil

	req := httptest.NewRequest("POST", "/api/runs/run-123/gate/approve", nil)
	req.SetPathValue("id", "run-123")
	req.SetPathValue("step", "approve")
	rec := httptest.NewRecorder()

	srv.handleResolveGate(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
