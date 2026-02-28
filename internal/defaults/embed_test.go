package defaults

import (
	"strings"
	"testing"

	"github.com/recinq/wave/internal/pipeline"
	"gopkg.in/yaml.v3"
)

func TestGhRewritePipeline_NoHardcodedRepo(t *testing.T) {
	pipelines, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	content, ok := pipelines["gh-rewrite.yaml"]
	if !ok {
		t.Fatal("gh-rewrite.yaml not found in embedded pipelines")
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

func TestGhRewritePipeline_UsesInputTemplate(t *testing.T) {
	pipelines, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	content, ok := pipelines["gh-rewrite.yaml"]
	if !ok {
		t.Fatal("gh-rewrite.yaml not found in embedded pipelines")
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

func TestGhRewritePipeline_InputSchemaIsString(t *testing.T) {
	pipelines, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	content, ok := pipelines["gh-rewrite.yaml"]
	if !ok {
		t.Fatal("gh-rewrite.yaml not found in embedded pipelines")
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
		"gh-pr-review.yaml",
		"dead-code.yaml",
		"debug.yaml",
		"doc-fix.yaml",
		"explain.yaml",
		"gh-research.yaml",
		"gh-rewrite.yaml",
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

func TestGetPersonaConfigs_ReturnsAllPersonas(t *testing.T) {
	configs, err := GetPersonaConfigs()
	if err != nil {
		t.Fatalf("GetPersonaConfigs() error: %v", err)
	}

	// Should have exactly 26 persona configs (all .md files minus base-protocol)
	if len(configs) != 26 {
		t.Errorf("expected 26 persona configs, got %d", len(configs))
	}

	// Verify a few known personas exist
	expected := []string{"navigator", "craftsman", "summarizer", "implementer", "github-analyst", "gitea-analyst", "gitlab-analyst", "bitbucket-analyst"}
	for _, name := range expected {
		if _, ok := configs[name]; !ok {
			t.Errorf("expected persona config %q not found", name)
		}
	}
}

func TestGetPersonaConfigs_HasRequiredFields(t *testing.T) {
	configs, err := GetPersonaConfigs()
	if err != nil {
		t.Fatalf("GetPersonaConfigs() error: %v", err)
	}

	for name, cfg := range configs {
		if cfg.Description == "" {
			t.Errorf("persona %q has empty Description", name)
		}
		if cfg.Temperature < 0 || cfg.Temperature > 1.0 {
			t.Errorf("persona %q has invalid Temperature %f (must be 0.0-1.0)", name, cfg.Temperature)
		}
		if len(cfg.Permissions.AllowedTools) == 0 {
			t.Errorf("persona %q has no allowed_tools", name)
		}
		// Adapter and SystemPromptFile should NOT be set — they're injected at init time
		if cfg.Adapter != "" {
			t.Errorf("persona %q should not have adapter set in config (got %q)", name, cfg.Adapter)
		}
		if cfg.SystemPromptFile != "" {
			t.Errorf("persona %q should not have system_prompt_file set in config (got %q)", name, cfg.SystemPromptFile)
		}
	}
}

func TestGetPersonaConfigs_ModelOverrides(t *testing.T) {
	configs, err := GetPersonaConfigs()
	if err != nil {
		t.Fatalf("GetPersonaConfigs() error: %v", err)
	}

	// Only these personas should have model overrides
	expectedModels := map[string]string{
		"provocateur": "opus",
		"validator":   "sonnet",
		"synthesizer": "sonnet",
	}

	for name, cfg := range configs {
		expected, hasExpected := expectedModels[name]
		if hasExpected {
			if cfg.Model != expected {
				t.Errorf("persona %q should have model %q, got %q", name, expected, cfg.Model)
			}
		} else {
			if cfg.Model != "" {
				t.Errorf("persona %q should not have a model override, got %q", name, cfg.Model)
			}
		}
	}
}

func TestGetPersonaConfigs_MatchesPersonaFiles(t *testing.T) {
	configs, err := GetPersonaConfigs()
	if err != nil {
		t.Fatalf("GetPersonaConfigs() error: %v", err)
	}

	personas, err := GetPersonas()
	if err != nil {
		t.Fatalf("GetPersonas() error: %v", err)
	}

	// Every persona config should have a corresponding .md file
	for name := range configs {
		mdFile := name + ".md"
		if _, ok := personas[mdFile]; !ok {
			t.Errorf("persona config %q has no corresponding .md file", name)
		}
	}

	// Every .md file (except base-protocol) should have a corresponding config
	for mdFile := range personas {
		if mdFile == "base-protocol.md" {
			continue
		}
		name := strings.TrimSuffix(mdFile, ".md")
		if _, ok := configs[name]; !ok {
			t.Errorf("persona .md file %q has no corresponding .yaml config", mdFile)
		}
	}
}
