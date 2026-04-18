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

// TestComposePageRendersEmptyHTML verifies that the compose page renders HTML
// successfully when no composition pipelines exist.
func TestComposePageRendersEmptyHTML(t *testing.T) {
	srv, _ := testServer(t)

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

	req := httptest.NewRequest("GET", "/compose", nil)
	rec := httptest.NewRecorder()
	srv.handleComposePage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type: expected %q, got %q", "text/html; charset=utf-8", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<html>") {
		t.Error("expected HTML content in response")
	}
}

// TestComposePageRendersPipelineNames verifies that the compose page renders
// pipeline names when composition pipelines exist.
func TestComposePageRendersPipelineNames(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".agents", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	// Write a pipeline with an iterate step (composition primitive).
	pipelineYAML := `kind: Pipeline
metadata:
  name: compose-render-test
  description: A composition pipeline
steps:
  - id: fan-out
    iterate:
      over: items
      mode: parallel
    pipeline: sub-task
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "compose-render-test.yaml"), []byte(pipelineYAML), 0o644); err != nil {
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

	req := httptest.NewRequest("GET", "/compose", nil)
	rec := httptest.NewRecorder()
	srv.handleComposePage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "compose-render-test") {
		t.Errorf("expected body to contain pipeline name 'compose-render-test', got: %s", body)
	}
}

// TestComposeAPIReturnsEmptyList verifies the API returns an empty pipeline list
// when no composition pipelines exist.
func TestComposeAPIReturnsEmptyList(t *testing.T) {
	srv, _ := testServer(t)

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

	req := httptest.NewRequest("GET", "/api/compose", nil)
	rec := httptest.NewRecorder()
	srv.handleAPICompose(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp CompositionListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Pipelines) != 0 {
		t.Errorf("expected 0 pipelines, got %d", len(resp.Pipelines))
	}
}

// TestComposeAPIReturnsStepDetails verifies the API returns composition
// pipelines with correct structure including step types.
func TestComposeAPIReturnsStepDetails(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".agents", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	pipelineYAML := `kind: Pipeline
metadata:
  name: compose-detail-test
  description: Test composition
  category: test
steps:
  - id: fan-out
    iterate:
      over: items
      mode: parallel
      max_concurrent: 3
    pipeline: worker
  - id: merge
    persona: navigator
    dependencies: [fan-out]
    exec:
      type: prompt
      source: "merge results"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "compose-detail-test.yaml"), []byte(pipelineYAML), 0o644); err != nil {
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

	req := httptest.NewRequest("GET", "/api/compose", nil)
	rec := httptest.NewRecorder()
	srv.handleAPICompose(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type: expected %q, got %q", "application/json", contentType)
	}

	var resp CompositionListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(resp.Pipelines))
	}

	p := resp.Pipelines[0]
	if p.Name != "compose-detail-test" {
		t.Errorf("expected pipeline name 'compose-detail-test', got %q", p.Name)
	}
	if p.Description != "Test composition" {
		t.Errorf("expected description 'Test composition', got %q", p.Description)
	}
	if p.StepCount != 2 {
		t.Errorf("expected 2 steps, got %d", p.StepCount)
	}
	if len(p.Steps) != 2 {
		t.Fatalf("expected 2 step details, got %d", len(p.Steps))
	}
	if p.Steps[0].Type != "iterate" {
		t.Errorf("expected step type 'iterate', got %q", p.Steps[0].Type)
	}
	if p.Steps[1].Type != "persona" {
		t.Errorf("expected step type 'persona', got %q", p.Steps[1].Type)
	}
}

// TestComposeAPIExcludesNonCompositionPipelines verifies that pipelines
// without composition primitives are not included in the response.
func TestComposeAPIExcludesNonCompositionPipelines(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".agents", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	// A pipeline with no composition primitives.
	plainYAML := `kind: Pipeline
metadata:
  name: plain-only-pipeline
steps:
  - id: step1
    persona: navigator
    exec:
      type: prompt
      source: "do something"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "plain-only-pipeline.yaml"), []byte(plainYAML), 0o644); err != nil {
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

	req := httptest.NewRequest("GET", "/api/compose", nil)
	rec := httptest.NewRecorder()
	srv.handleAPICompose(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp CompositionListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Pipelines) != 0 {
		t.Errorf("expected 0 composition pipelines, got %d", len(resp.Pipelines))
	}
}
