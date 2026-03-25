package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAPIDiffSummary_RunNotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs/nonexistent/diff", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleAPIDiffSummary(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}

func TestHandleAPIDiffSummary_EmptyBranchName(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatal(err)
	}
	// BranchName is empty by default

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/diff", nil)
	req.SetPathValue("id", runID)
	rec := httptest.NewRecorder()
	srv.handleAPIDiffSummary(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var summary DiffSummary
	json.NewDecoder(rec.Body).Decode(&summary)
	if summary.Available {
		t.Error("expected Available=false for empty BranchName")
	}
	if summary.Message == "" {
		t.Error("expected message for empty BranchName")
	}
}

func TestHandleAPIDiffSummary_MissingRunID(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs//diff", nil)
	req.SetPathValue("id", "")
	rec := httptest.NewRecorder()
	srv.handleAPIDiffSummary(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAPIDiffFile_RunNotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/runs/nonexistent/diff/foo.go", nil)
	req.SetPathValue("id", "nonexistent")
	req.SetPathValue("path", "foo.go")
	rec := httptest.NewRecorder()
	srv.handleAPIDiffFile(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAPIDiffFile_PathTraversal(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatal(err)
	}
	rwStore.UpdateRunBranch(runID, "some-branch")

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/diff/../../../etc/passwd", nil)
	req.SetPathValue("id", runID)
	req.SetPathValue("path", "../../../etc/passwd")
	rec := httptest.NewRecorder()
	srv.handleAPIDiffFile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAPIDiffFile_EmptyBranchName(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatal(err)
	}
	// BranchName is empty by default

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/diff/foo.go", nil)
	req.SetPathValue("id", runID)
	req.SetPathValue("path", "foo.go")
	rec := httptest.NewRecorder()
	srv.handleAPIDiffFile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAPIDiffFile_MissingPath(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatal(err)
	}
	rwStore.UpdateRunBranch(runID, "some-branch")

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/diff/", nil)
	req.SetPathValue("id", runID)
	req.SetPathValue("path", "")
	rec := httptest.NewRecorder()
	srv.handleAPIDiffFile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
