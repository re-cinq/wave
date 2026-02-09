package defaults

import (
	"strings"
	"testing"
)

func TestGitHubIssueEnhancerPipeline_NoHardcodedRepo(t *testing.T) {
	pipelines, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	content, ok := pipelines["github-issue-enhancer.yaml"]
	if !ok {
		t.Fatal("github-issue-enhancer.yaml not found in embedded pipelines")
	}

	// Every gh command should use {{ input }} for the repo, not a hardcoded value
	for i, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		// Skip comment-only lines that don't contain gh commands
		if !strings.Contains(trimmed, "gh ") {
			continue
		}
		// Lines with --repo must use {{ input }}, not a hardcoded owner/repo
		if strings.Contains(trimmed, "--repo") {
			if !strings.Contains(trimmed, "--repo {{ input }}") {
				t.Errorf("line %d has hardcoded --repo value: %s", i+1, trimmed)
			}
		}
	}
}

func TestGitHubIssueEnhancerPipeline_UsesInputTemplate(t *testing.T) {
	pipelines, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	content, ok := pipelines["github-issue-enhancer.yaml"]
	if !ok {
		t.Fatal("github-issue-enhancer.yaml not found in embedded pipelines")
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

func TestGitHubIssueEnhancerPipeline_InputSchemaIsString(t *testing.T) {
	pipelines, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	content, ok := pipelines["github-issue-enhancer.yaml"]
	if !ok {
		t.Fatal("github-issue-enhancer.yaml not found in embedded pipelines")
	}

	// Input schema should be a simple string type, not a structured object
	if strings.Contains(content, "type: object") {
		t.Error("input schema should be type: string, not type: object")
	}
	if !strings.Contains(content, "type: string") {
		t.Error("input schema should contain type: string")
	}
}
