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

// TestPipelinesPageRendersEmptyHTML verifies that the pipelines page renders HTML
// successfully when no pipeline files exist.
func TestPipelinesPageRendersEmptyHTML(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/pipelines", nil)
	rec := httptest.NewRecorder()
	srv.handlePipelinesPage(rec, req)

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

// TestPipelinesPageRendersNames verifies that the pipelines page renders
// pipeline names when pipeline files exist.
func TestPipelinesPageRendersNames(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".agents", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	pipelineYAML := `kind: Pipeline
metadata:
  name: page-render-pipeline
  description: A test pipeline
  category: test
steps:
  - id: step1
    persona: navigator
    exec:
      type: prompt
      source: "plan"
  - id: step2
    persona: craftsman
    dependencies: [step1]
    exec:
      type: prompt
      source: "implement"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "page-render-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
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

	req := httptest.NewRequest("GET", "/pipelines", nil)
	rec := httptest.NewRecorder()
	srv.handlePipelinesPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "page-render-pipeline") {
		t.Errorf("expected body to contain pipeline name 'page-render-pipeline', got: %s", body)
	}
}

// TestPipelinesAPIReturnsEmptyList verifies the API returns an empty pipeline list
// when no pipeline files exist.
func TestPipelinesAPIReturnsEmptyList(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/api/pipelines", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPipelines(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	pipelines, ok := resp["pipelines"]
	if !ok {
		t.Fatal("expected 'pipelines' key in response")
	}
	if pipelines != nil {
		arr, ok := pipelines.([]interface{})
		if ok && len(arr) != 0 {
			t.Errorf("expected empty pipelines list, got %d items", len(arr))
		}
	}
}

// TestPipelinesAPIReturnsSummaryFields verifies the API returns pipeline summaries
// with correct fields including step IDs and category.
func TestPipelinesAPIReturnsSummaryFields(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".agents", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	pipelineYAML := `kind: Pipeline
metadata:
  name: summary-fields-pipeline
  description: Summary fields test
  category: testing
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "analyze"
  - id: build
    persona: craftsman
    dependencies: [analyze]
    exec:
      type: prompt
      source: "build"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "summary-fields-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
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

	req := httptest.NewRequest("GET", "/api/pipelines", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPipelines(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type: expected %q, got %q", "application/json", contentType)
	}

	var resp map[string][]PipelineSummary
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	pipelines := resp["pipelines"]
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}

	p := pipelines[0]
	if p.Name != "summary-fields-pipeline" {
		t.Errorf("expected name 'summary-fields-pipeline', got %q", p.Name)
	}
	if p.Description != "Summary fields test" {
		t.Errorf("expected description 'Summary fields test', got %q", p.Description)
	}
	if p.Category != "testing" {
		t.Errorf("expected category 'testing', got %q", p.Category)
	}
	if p.StepCount != 2 {
		t.Errorf("expected 2 steps, got %d", p.StepCount)
	}
	if len(p.Steps) != 2 {
		t.Fatalf("expected 2 step IDs, got %d", len(p.Steps))
	}
	if p.Steps[0] != "analyze" {
		t.Errorf("expected first step 'analyze', got %q", p.Steps[0])
	}
	if p.Steps[1] != "build" {
		t.Errorf("expected second step 'build', got %q", p.Steps[1])
	}
}

// TestPipelineInfoAPIReturnsMetadata verifies the pipeline info endpoint returns
// lightweight metadata for the start form.
func TestPipelineInfoAPIReturnsMetadata(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".agents", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	pipelineYAML := `kind: Pipeline
metadata:
  name: info-metadata-pipeline
  description: Info metadata test
  category: ops
steps:
  - id: step1
    persona: navigator
    exec:
      type: prompt
      source: "work"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "info-metadata-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
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

	req := httptest.NewRequest("GET", "/api/pipelines/info", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPipelineInfo(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string][]PipelineStartInfo
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	infos := resp["pipelines"]
	if len(infos) != 1 {
		t.Fatalf("expected 1 pipeline info, got %d", len(infos))
	}

	info := infos[0]
	if info.Name != "info-metadata-pipeline" {
		t.Errorf("expected name 'info-metadata-pipeline', got %q", info.Name)
	}
	if info.Description != "Info metadata test" {
		t.Errorf("expected description 'Info metadata test', got %q", info.Description)
	}
	if info.Category != "ops" {
		t.Errorf("expected category 'ops', got %q", info.Category)
	}
	if info.StepCount != 1 {
		t.Errorf("expected 1 step, got %d", info.StepCount)
	}
}

// TestPipelineInfoAPIReturnsEmptyList verifies the pipeline info endpoint returns
// an empty list when no pipelines exist.
func TestPipelineInfoAPIReturnsEmptyList(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/api/pipelines/info", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPipelineInfo(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	pipelines, ok := resp["pipelines"]
	if !ok {
		t.Fatal("expected 'pipelines' key in response")
	}
	if pipelines != nil {
		arr, ok := pipelines.([]interface{})
		if ok && len(arr) != 0 {
			t.Errorf("expected empty pipeline info list, got %d items", len(arr))
		}
	}
}
