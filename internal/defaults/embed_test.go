package defaults

import (
	"strings"
	"testing"

	"github.com/recinq/wave/internal/manifest"
)

func TestGetReleasePipelines_ReturnsSubset(t *testing.T) {
	all, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}

	release, err := GetReleasePipelines()
	if err != nil {
		t.Fatalf("GetReleasePipelines() error: %v", err)
	}

	if len(release) > len(all) {
		t.Errorf("release set (%d) should be a subset of all pipelines (%d)", len(release), len(all))
	}

	for name := range release {
		if _, ok := all[name]; !ok {
			t.Errorf("release pipeline %q not found in all pipelines", name)
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
		header, err := manifest.LoadPipelineHeader([]byte(content))
		if err != nil {
			t.Errorf("failed to unmarshal release pipeline %q: %v", name, err)
			continue
		}
		if !header.Metadata.Release {
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
		header, err := manifest.LoadPipelineHeader([]byte(content))
		if err != nil {
			continue // skip pipelines that fail to unmarshal
		}
		if !header.Metadata.Release {
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

	// Post-consolidation fleet (see docs/adr/010-pipeline-io-protocol.md
	// and .agents/output/consolidation-map.md).
	expected := []string{
		"audit-security.yaml",
		"impl-issue.yaml",
		"ops-pr-review.yaml",
		"plan-research.yaml",
		"plan-scope.yaml",
		"plan-task.yaml",
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
		header, err := manifest.LoadPipelineHeader([]byte(content))
		if err != nil {
			continue
		}
		if header.Metadata.Release && header.Metadata.Disabled {
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

	// Should have exactly 12 persona configs (all .md files minus base-protocol)
	if len(configs) != 12 {
		t.Errorf("expected 12 persona configs, got %d", len(configs))
	}

	// Verify a few known personas exist
	expected := []string{"navigator", "craftsman", "summarizer", "implementer", "github-analyst", "github-scoper", "github-commenter"}
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

	for name, cfg := range configs {
		if cfg.Model != "" {
			t.Errorf("persona %q should not have a hardcoded model (adapter-agnostic), got %q", name, cfg.Model)
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
