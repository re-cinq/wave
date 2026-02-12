package defaults

import (
	"strings"
	"testing"

	"github.com/recinq/wave/internal/pipeline"
	"gopkg.in/yaml.v3"
)

func TestGhIssueRewritePipeline_NoHardcodedRepo(t *testing.T) {
	pipelines, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	content, ok := pipelines["gh-issue-rewrite.yaml"]
	if !ok {
		t.Fatal("gh-issue-rewrite.yaml not found in embedded pipelines")
	}

	// Every gh command should use {{ input }} for the repo, not a hardcoded value
	for i, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		// Skip comment-only lines that don't contain gh commands
		if !strings.Contains(trimmed, "gh ") {
			continue
		}
		// Lines with --repo must use {{ input }} or <REPO> placeholder, not a hardcoded owner/repo
		if strings.Contains(trimmed, "--repo") {
			if !strings.Contains(trimmed, "--repo {{ input }}") && !strings.Contains(trimmed, "--repo <REPO>") {
				t.Errorf("line %d has hardcoded --repo value: %s", i+1, trimmed)
			}
		}
	}
}

func TestGhIssueRewritePipeline_UsesInputTemplate(t *testing.T) {
	pipelines, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	content, ok := pipelines["gh-issue-rewrite.yaml"]
	if !ok {
		t.Fatal("gh-issue-rewrite.yaml not found in embedded pipelines")
	}

	// The pipeline must contain {{ input }} template variables for interpolation
	if !strings.Contains(content, "{{ input }}") {
		t.Error("pipeline should contain {{ input }} template variables")
	}

	// Count occurrences — scan-issues has 2 (Input line + gh issue list),
	// plan-enhancements has 1 (gh issue view), apply-enhancements has 3,
	// verify-enhancements has 1 = 7 total minimum
	count := strings.Count(content, "{{ input }}")
	if count < 7 {
		t.Errorf("expected at least 7 {{ input }} occurrences, got %d", count)
	}
}

func TestGhIssueRewritePipeline_InputSchemaIsString(t *testing.T) {
	pipelines, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	content, ok := pipelines["gh-issue-rewrite.yaml"]
	if !ok {
		t.Fatal("gh-issue-rewrite.yaml not found in embedded pipelines")
	}

	// Input schema should be a simple string type, not a structured object
	if strings.Contains(content, "type: object") {
		t.Error("input schema should be type: string, not type: object")
	}
	if !strings.Contains(content, "type: string") {
		t.Error("input schema should contain type: string")
	}
}

func TestGetReleasePipelines_ReturnsSubset(t *testing.T) {
	all, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	release, err := GetReleasePipelines()
	if err != nil {
		t.Fatalf("GetReleasePipelines() error: %v", err)
	}

	if len(release) >= len(all) {
		t.Errorf("release set (%d) should be a strict subset of all pipelines (%d)", len(release), len(all))
	}

	for name := range release {
		if _, ok := all[name]; !ok {
			t.Errorf("release pipeline %q not found in all pipelines", name)
		}
	}
}

func TestReleasePipelineNames_ReturnsSubset(t *testing.T) {
	allNames := PipelineNames()
	releaseNames := ReleasePipelineNames()

	if len(releaseNames) >= len(allNames) {
		t.Errorf("release names (%d) should be a strict subset of all pipeline names (%d)", len(releaseNames), len(allNames))
	}

	allSet := make(map[string]bool, len(allNames))
	for _, name := range allNames {
		allSet[name] = true
	}

	for _, name := range releaseNames {
		if !allSet[name] {
			t.Errorf("release pipeline name %q not found in all pipeline names", name)
		}
	}
}

func TestGetReleasePipelines_OnlyReleaseTrue(t *testing.T) {
	release, err := GetReleasePipelines()
	if err != nil {
		t.Fatalf("GetReleasePipelines() error: %v", err)
	}

	if len(release) == 0 {
		t.Fatal("expected at least one release pipeline, got 0")
	}

	for name, content := range release {
		var p pipeline.Pipeline
		if err := yaml.Unmarshal([]byte(content), &p); err != nil {
			t.Errorf("failed to unmarshal release pipeline %q: %v", name, err)
			continue
		}
		if !p.Metadata.Release {
			t.Errorf("pipeline %q is in release set but metadata.release is false", name)
		}
	}
}

func TestGetReleasePipelines_ExcludesNonRelease(t *testing.T) {
	all, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	release, err := GetReleasePipelines()
	if err != nil {
		t.Fatalf("GetReleasePipelines() error: %v", err)
	}

	for name, content := range all {
		var p pipeline.Pipeline
		if err := yaml.Unmarshal([]byte(content), &p); err != nil {
			continue // skip pipelines that fail to unmarshal
		}
		if !p.Metadata.Release {
			if _, ok := release[name]; ok {
				t.Errorf("pipeline %q does not have release: true but is in the release set", name)
			}
		}
	}
}

func TestGetReleasePipelines_KnownReleasePipelines(t *testing.T) {
	release, err := GetReleasePipelines()
	if err != nil {
		t.Fatalf("GetReleasePipelines() error: %v", err)
	}

	expected := []string{
		"adr.yaml",
		"changelog.yaml",
		"code-review.yaml",
		"dead-code.yaml",
		"debug.yaml",
		"doc-sync.yaml",
		"explain.yaml",
		"feature.yaml",
		"gh-issue-research.yaml",
		"gh-issue-rewrite.yaml",
		"improve.yaml",
		"onboard.yaml",
		"plan.yaml",
		"refactor.yaml",
		"security-scan.yaml",
		"test-gen.yaml",
	}

	for _, name := range expected {
		if _, ok := release[name]; !ok {
			t.Errorf("expected release pipeline %q not found in GetReleasePipelines() result", name)
		}
	}
}

func TestGetReleasePipelines_DisabledAndReleaseIncluded(t *testing.T) {
	all, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	release, err := GetReleasePipelines()
	if err != nil {
		t.Fatalf("GetReleasePipelines() error: %v", err)
	}

	for name, content := range all {
		var p pipeline.Pipeline
		if err := yaml.Unmarshal([]byte(content), &p); err != nil {
			continue
		}
		if p.Metadata.Release && p.Metadata.Disabled {
			if _, ok := release[name]; !ok {
				t.Errorf("pipeline %q has both release: true and disabled: true but is not in the release set", name)
			}
		}
	}
}
