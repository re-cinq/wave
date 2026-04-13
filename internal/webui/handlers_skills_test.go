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

// TestSkillsPageRendersEmptyHTML verifies that the skills page renders HTML
// successfully when no pipeline files with skills exist.
func TestSkillsPageRendersEmptyHTML(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/skills", nil)
	rec := httptest.NewRecorder()
	srv.handleSkillsPage(rec, req)

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

// TestSkillsPageRendersSkillNames verifies that the skills page renders
// skill names extracted from pipeline files.
func TestSkillsPageRendersSkillNames(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	pipelineYAML := `kind: Pipeline
metadata:
  name: skill-page-test-pipeline
requires:
  skills:
    git-skill:
      check: "git --version"
      install: "apt install git"
    docker-skill:
      check: "docker --version"
steps:
  - id: step1
    persona: navigator
    exec:
      type: prompt
      source: "work"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "skill-page-test-pipeline.yaml"), []byte(pipelineYAML), 0o644); err != nil {
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

	req := httptest.NewRequest("GET", "/skills", nil)
	rec := httptest.NewRecorder()
	srv.handleSkillsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "git-skill") {
		t.Errorf("expected body to contain skill name 'git-skill', got: %s", body)
	}
	if !strings.Contains(body, "docker-skill") {
		t.Errorf("expected body to contain skill name 'docker-skill', got: %s", body)
	}
}

// TestSkillsAPIReturnsEmptyList verifies the API returns an empty skill list
// when no pipelines with skills exist.
func TestSkillsAPIReturnsEmptyList(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/api/skills", nil)
	rec := httptest.NewRecorder()
	srv.handleAPISkills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp SkillListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(resp.Skills))
	}
}

// TestSkillsAPIReturnsCheckAndInstallCmds verifies the API returns skill summaries
// with correct fields including check and install commands.
func TestSkillsAPIReturnsCheckAndInstallCmds(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	pipelineYAML := `kind: Pipeline
metadata:
  name: skill-cmds-test
requires:
  skills:
    gh-cli-skill:
      check: "gh --version"
      install: "brew install gh"
steps:
  - id: step1
    persona: navigator
    exec:
      type: prompt
      source: "work"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "skill-cmds-test.yaml"), []byte(pipelineYAML), 0o644); err != nil {
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

	req := httptest.NewRequest("GET", "/api/skills", nil)
	rec := httptest.NewRecorder()
	srv.handleAPISkills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type: expected %q, got %q", "application/json", contentType)
	}

	var resp SkillListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(resp.Skills))
	}

	s := resp.Skills[0]
	if s.Name != "gh-cli-skill" {
		t.Errorf("expected skill name 'gh-cli-skill', got %q", s.Name)
	}
	if s.CheckCmd != "gh --version" {
		t.Errorf("expected check command 'gh --version', got %q", s.CheckCmd)
	}
	if s.InstallCmd != "brew install gh" {
		t.Errorf("expected install command 'brew install gh', got %q", s.InstallCmd)
	}
	if len(s.PipelineUsage) != 1 || s.PipelineUsage[0] != "skill-cmds-test" {
		t.Errorf("expected pipeline usage ['skill-cmds-test'], got %v", s.PipelineUsage)
	}
}

// TestSkillsAPIDeduplicatesAcrossPipelines verifies that skills used by
// multiple pipelines are deduplicated with combined pipeline usage.
func TestSkillsAPIDeduplicatesAcrossPipelines(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	pipeline1 := `kind: Pipeline
metadata:
  name: dedup-pipeline-a
requires:
  skills:
    shared-dedup-skill:
      check: "shared --version"
steps:
  - id: step1
    persona: navigator
    exec:
      type: prompt
      source: "work"
`
	pipeline2 := `kind: Pipeline
metadata:
  name: dedup-pipeline-b
requires:
  skills:
    shared-dedup-skill:
      check: "shared --version"
steps:
  - id: step1
    persona: craftsman
    exec:
      type: prompt
      source: "build"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "dedup-pipeline-a.yaml"), []byte(pipeline1), 0o644); err != nil {
		t.Fatalf("failed to write pipeline yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pipelineDir, "dedup-pipeline-b.yaml"), []byte(pipeline2), 0o644); err != nil {
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

	req := httptest.NewRequest("GET", "/api/skills", nil)
	rec := httptest.NewRecorder()
	srv.handleAPISkills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp SkillListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Skills) != 1 {
		t.Fatalf("expected 1 deduplicated skill, got %d", len(resp.Skills))
	}

	s := resp.Skills[0]
	if s.Name != "shared-dedup-skill" {
		t.Errorf("expected skill name 'shared-dedup-skill', got %q", s.Name)
	}
	if len(s.PipelineUsage) != 2 {
		t.Errorf("expected 2 pipeline usages, got %d: %v", len(s.PipelineUsage), s.PipelineUsage)
	}
}

// TestSkillsAPIExcludesPipelinesWithoutSkills verifies that pipelines
// without a requires.skills section do not contribute to the skill list.
func TestSkillsAPIExcludesPipelinesWithoutSkills(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	pipelineDir := filepath.Join(tmpDir, ".wave", "pipelines")
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		t.Fatalf("failed to create pipeline dir: %v", err)
	}

	noSkillsYAML := `kind: Pipeline
metadata:
  name: no-skills-exclude-pipeline
steps:
  - id: step1
    persona: navigator
    exec:
      type: prompt
      source: "work"
`
	if err := os.WriteFile(filepath.Join(pipelineDir, "no-skills-exclude-pipeline.yaml"), []byte(noSkillsYAML), 0o644); err != nil {
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

	req := httptest.NewRequest("GET", "/api/skills", nil)
	rec := httptest.NewRecorder()
	srv.handleAPISkills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp SkillListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(resp.Skills))
	}
}
