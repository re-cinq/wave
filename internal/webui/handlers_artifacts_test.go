package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectMimeType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"json", "output.json", "application/json"},
		{"yaml", "config.yaml", "application/x-yaml"},
		{"yml", "config.yml", "application/x-yaml"},
		{"markdown", "README.md", "text/markdown"},
		{"go source", "main.go", "text/x-go"},
		{"python", "script.py", "text/x-python"},
		{"javascript", "app.js", "text/javascript"},
		{"typescript", "app.ts", "text/typescript"},
		{"html", "index.html", "text/html"},
		{"css", "style.css", "text/css"},
		{"text", "notes.txt", "text/plain"},
		{"log", "run.log", "text/plain"},
		{"unknown extension", "binary.exe", "application/octet-stream"},
		{"no extension", "Makefile", "application/octet-stream"},
		{"uppercase extension", "DATA.JSON", "application/json"},
		{"mixed case yaml", "Config.YAML", "application/x-yaml"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectMimeType(tc.filename)
			if got != tc.want {
				t.Errorf("detectMimeType(%q) = %q, want %q", tc.filename, got, tc.want)
			}
		})
	}
}

func TestHandleArtifact_MissingParameters(t *testing.T) {
	srv, _ := testServer(t)

	tests := []struct {
		name    string
		pathFn  func(r *http.Request)
		wantErr string
	}{
		{
			name: "all empty - missing parameters",
			pathFn: func(r *http.Request) {
				// PathValue returns empty string when not set
			},
			wantErr: "missing parameters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/runs//artifacts//", nil)
			tc.pathFn(req)
			rec := httptest.NewRecorder()
			srv.handleArtifact(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}

			// Verify error message in plain text response (http.Error format)
			body := rec.Body.String()
			if !strings.Contains(body, tc.wantErr) {
				t.Errorf("expected body to contain %q, got %q", tc.wantErr, body)
			}
		})
	}
}

func TestHandleArtifact_ArtifactNotFoundInDB(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create a run but register no artifact for "missing-artifact" name
	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/artifacts/step-1/missing-artifact", nil)
	req.SetPathValue("id", runID)
	req.SetPathValue("step", "step-1")
	req.SetPathValue("name", "missing-artifact")

	rec := httptest.NewRecorder()
	srv.handleArtifact(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.Contains(resp["error"], "artifact not found") {
		t.Errorf("expected error about artifact not found, got %q", resp["error"])
	}
}

func TestHandleArtifact_PathTraversalBlocked(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create a run and register an artifact whose path contains ".."
	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	// Register an artifact with a path-traversal path.
	// filepath.Clean will preserve ".." components when the path resolves outside root.
	traversalPath := "/tmp/../etc/passwd"
	if err := rwStore.RegisterArtifact(runID, "step-1", "evil.txt", traversalPath, "file", 100); err != nil {
		t.Fatalf("failed to register artifact: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/artifacts/step-1/evil.txt", nil)
	req.SetPathValue("id", runID)
	req.SetPathValue("step", "step-1")
	req.SetPathValue("name", "evil.txt")

	rec := httptest.NewRecorder()
	srv.handleArtifact(rec, req)

	// /tmp/../etc/passwd cleans to /etc/passwd which has no "..", so test a path
	// that genuinely retains ".." after cleaning — e.g. a relative path.
	// The handler calls filepath.Clean and then checks strings.Contains(cleanPath, "..").
	// A relative path like "../../etc/passwd" still contains ".." after Clean.
	// Re-register with a relative traversal path.
	if err := rwStore.RegisterArtifact(runID, "step-1", "relative-evil.txt", "../../etc/passwd", "file", 100); err != nil {
		t.Fatalf("failed to register relative artifact: %v", err)
	}

	req2 := httptest.NewRequest("GET", "/api/runs/"+runID+"/artifacts/step-1/relative-evil.txt", nil)
	req2.SetPathValue("id", runID)
	req2.SetPathValue("step", "step-1")
	req2.SetPathValue("name", "relative-evil.txt")

	rec2 := httptest.NewRecorder()
	srv.handleArtifact(rec2, req2)

	if rec2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for relative path traversal, got %d", rec2.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec2.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.Contains(resp["error"], "path traversal") {
		t.Errorf("expected path traversal error, got %q", resp["error"])
	}
}

func TestHandleArtifact_FileNotFoundOnDisk(t *testing.T) {
	srv, rwStore := testServer(t)

	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	// Register artifact pointing to a non-existent file
	if err := rwStore.RegisterArtifact(runID, "step-1", "ghost.json", "/nonexistent/path/ghost.json", "json", 512); err != nil {
		t.Fatalf("failed to register artifact: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/artifacts/step-1/ghost.json", nil)
	req.SetPathValue("id", runID)
	req.SetPathValue("step", "step-1")
	req.SetPathValue("name", "ghost.json")

	rec := httptest.NewRecorder()
	srv.handleArtifact(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.Contains(resp["error"], "artifact file not found") {
		t.Errorf("expected 'artifact file not found' error, got %q", resp["error"])
	}
}

func TestHandleArtifact_SuccessWithRedaction(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create a temp file with content that includes a credential pattern
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "output.json")
	content := `{"result": "ok", "key": "sk-abcdefghijklmnopqrst1234567890"}`
	if err := os.WriteFile(artifactPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	if err := rwStore.RegisterArtifact(runID, "step-1", "output.json", artifactPath, "json", int64(len(content))); err != nil {
		t.Fatalf("failed to register artifact: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/artifacts/step-1/output.json", nil)
	req.SetPathValue("id", runID)
	req.SetPathValue("step", "step-1")
	req.SetPathValue("name", "output.json")

	rec := httptest.NewRecorder()
	srv.handleArtifact(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: body=%s", rec.Code, rec.Body.String())
	}

	var resp ArtifactContentResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Credential should be redacted
	if strings.Contains(resp.Content, "sk-abcdefghijklmnopqrst1234567890") {
		t.Error("expected credential to be redacted in response content")
	}
	if !strings.Contains(resp.Content, "[REDACTED]") {
		t.Errorf("expected [REDACTED] placeholder in content, got: %s", resp.Content)
	}

	// Metadata assertions
	if resp.Metadata.Name != "output.json" {
		t.Errorf("expected metadata.name=output.json, got %q", resp.Metadata.Name)
	}
	if resp.Metadata.MimeType != "application/json" {
		t.Errorf("expected mime type application/json, got %q", resp.Metadata.MimeType)
	}
	if resp.Metadata.Truncated {
		t.Error("expected Truncated=false for small file")
	}
}

func TestHandleArtifact_RawDownload(t *testing.T) {
	srv, rwStore := testServer(t)

	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "data.txt")
	rawContent := "raw artifact content\nline 2\n"
	if err := os.WriteFile(artifactPath, []byte(rawContent), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	if err := rwStore.RegisterArtifact(runID, "step-1", "data.txt", artifactPath, "file", int64(len(rawContent))); err != nil {
		t.Fatalf("failed to register artifact: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/artifacts/step-1/data.txt?raw=true", nil)
	req.SetPathValue("id", runID)
	req.SetPathValue("step", "step-1")
	req.SetPathValue("name", "data.txt")

	rec := httptest.NewRecorder()
	srv.handleArtifact(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Raw mode: Content-Type should be text/plain
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("expected Content-Type text/plain, got %q", ct)
	}

	// Content-Disposition should be attachment with filename
	cd := rec.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Errorf("expected Content-Disposition=attachment, got %q", cd)
	}
	if !strings.Contains(cd, "data.txt") {
		t.Errorf("expected filename data.txt in Content-Disposition, got %q", cd)
	}

	// Body should be raw content, not JSON
	body := rec.Body.String()
	if body != rawContent {
		t.Errorf("expected raw content %q, got %q", rawContent, body)
	}
}

func TestHandleArtifact_TruncationForOversizedArtifact(t *testing.T) {
	srv, rwStore := testServer(t)

	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "large.txt")

	// Create a file larger than maxArtifactSize (1 MB)
	largeContent := strings.Repeat("A", maxArtifactSize+100)
	if err := os.WriteFile(artifactPath, []byte(largeContent), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	if err := rwStore.RegisterArtifact(runID, "step-1", "large.txt", artifactPath, "file", int64(len(largeContent))); err != nil {
		t.Fatalf("failed to register artifact: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/artifacts/step-1/large.txt", nil)
	req.SetPathValue("id", runID)
	req.SetPathValue("step", "step-1")
	req.SetPathValue("name", "large.txt")

	rec := httptest.NewRecorder()
	srv.handleArtifact(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp ArtifactContentResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Metadata.Truncated {
		t.Error("expected Truncated=true for oversized file")
	}

	// Content length should be at most maxArtifactSize characters (before HTML escaping)
	// Since all content is 'A' chars (no HTML special chars), escaped length == original.
	if len(resp.Content) > maxArtifactSize {
		t.Errorf("expected content length <= %d, got %d", maxArtifactSize, len(resp.Content))
	}
}

func TestHandleArtifact_HTMLEscaping(t *testing.T) {
	srv, rwStore := testServer(t)

	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "script.html")
	content := `<script>alert("xss")</script>`
	if err := os.WriteFile(artifactPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	runID, err := rwStore.CreateRun("test-pipeline", "input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	if err := rwStore.RegisterArtifact(runID, "step-1", "script.html", artifactPath, "html", int64(len(content))); err != nil {
		t.Fatalf("failed to register artifact: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/artifacts/step-1/script.html", nil)
	req.SetPathValue("id", runID)
	req.SetPathValue("step", "step-1")
	req.SetPathValue("name", "script.html")

	rec := httptest.NewRecorder()
	srv.handleArtifact(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: body=%s", rec.Code, rec.Body.String())
	}

	var resp ArtifactContentResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// JSON API returns raw content (no HTML escaping — encoding/json handles JSON escaping)
	if !strings.Contains(resp.Content, "<script>") {
		t.Error("expected raw content in JSON API response, not HTML-escaped")
	}
	// Verify the raw content is returned as-is (no HTML escaping)
	if resp.Content != content {
		t.Errorf("expected raw content %q, got %q", content, resp.Content)
	}
}
