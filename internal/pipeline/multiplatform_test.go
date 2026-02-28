package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGitLabPipelinesParseable verifies all gl-* pipeline YAML files
// can be parsed by the YAMLPipelineLoader and have valid DAG structure.
func TestGitLabPipelinesParseable(t *testing.T) {
	loader := &YAMLPipelineLoader{}
	validator := &DAGValidator{}

	pipelines := []struct {
		name     string
		file     string
		minSteps int
	}{
		{"gl-implement", "gl-implement.yaml", 4},
		{"gl-research", "gl-research.yaml", 5},
		{"gl-refresh", "gl-refresh.yaml", 3},
		{"gl-rewrite", "gl-rewrite.yaml", 2},
	}

	repoRoot := filepath.Join("..", "..")

	for _, tc := range pipelines {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(repoRoot, ".wave", "pipelines", tc.file)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tc.file, err)
			}

			p, err := loader.Unmarshal(data)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tc.file, err)
			}

			if p.Metadata.Name != tc.name {
				t.Errorf("expected metadata.name %q, got %q", tc.name, p.Metadata.Name)
			}

			if len(p.Steps) < tc.minSteps {
				t.Errorf("expected at least %d steps, got %d", tc.minSteps, len(p.Steps))
			}

			if err := validator.ValidateDAG(p); err != nil {
				t.Errorf("DAG validation failed for %s: %v", tc.name, err)
			}

			sorted, err := validator.TopologicalSort(p)
			if err != nil {
				t.Errorf("topological sort failed for %s: %v", tc.name, err)
			}
			if len(sorted) != len(p.Steps) {
				t.Errorf("topological sort returned %d steps, expected %d", len(sorted), len(p.Steps))
			}
		})
	}
}

// TestGiteaPipelinesParseable verifies all gt-* pipeline YAML files
// can be parsed by the YAMLPipelineLoader and have valid DAG structure.
func TestGiteaPipelinesParseable(t *testing.T) {
	loader := &YAMLPipelineLoader{}
	validator := &DAGValidator{}

	pipelines := []struct {
		name     string
		file     string
		minSteps int
	}{
		{"gt-implement", "gt-implement.yaml", 4},
		{"gt-research", "gt-research.yaml", 5},
		{"gt-refresh", "gt-refresh.yaml", 3},
		{"gt-rewrite", "gt-rewrite.yaml", 2},
	}

	repoRoot := filepath.Join("..", "..")

	for _, tc := range pipelines {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(repoRoot, ".wave", "pipelines", tc.file)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tc.file, err)
			}

			p, err := loader.Unmarshal(data)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tc.file, err)
			}

			if p.Metadata.Name != tc.name {
				t.Errorf("expected metadata.name %q, got %q", tc.name, p.Metadata.Name)
			}

			if len(p.Steps) < tc.minSteps {
				t.Errorf("expected at least %d steps, got %d", tc.minSteps, len(p.Steps))
			}

			if err := validator.ValidateDAG(p); err != nil {
				t.Errorf("DAG validation failed for %s: %v", tc.name, err)
			}

			sorted, err := validator.TopologicalSort(p)
			if err != nil {
				t.Errorf("topological sort failed for %s: %v", tc.name, err)
			}
			if len(sorted) != len(p.Steps) {
				t.Errorf("topological sort returned %d steps, expected %d", len(sorted), len(p.Steps))
			}
		})
	}
}

// TestGitLabPipelinesUseGitLabPersonas verifies gl-* pipelines reference
// GitLab-specific personas (not GitHub ones) for platform-specific steps.
func TestGitLabPipelinesUseGitLabPersonas(t *testing.T) {
	loader := &YAMLPipelineLoader{}
	repoRoot := filepath.Join("..", "..")

	tests := []struct {
		file             string
		forbiddenPersona string
	}{
		{"gl-research.yaml", "github-analyst"},
		{"gl-research.yaml", "github-commenter"},
		{"gl-refresh.yaml", "github-analyst"},
		{"gl-refresh.yaml", "github-enhancer"},
		{"gl-rewrite.yaml", "github-analyst"},
		{"gl-rewrite.yaml", "github-enhancer"},
	}

	for _, tc := range tests {
		t.Run(tc.file+"_no_"+tc.forbiddenPersona, func(t *testing.T) {
			path := filepath.Join(repoRoot, ".wave", "pipelines", tc.file)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tc.file, err)
			}

			p, err := loader.Unmarshal(data)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tc.file, err)
			}

			for _, step := range p.Steps {
				if step.Persona == tc.forbiddenPersona {
					t.Errorf("step %q in %s uses forbidden persona %q", step.ID, tc.file, tc.forbiddenPersona)
				}
			}
		})
	}
}

// TestGiteaPipelinesUseGiteaPersonas verifies gt-* pipelines reference
// Gitea-specific personas (not GitHub ones) for platform-specific steps.
func TestGiteaPipelinesUseGiteaPersonas(t *testing.T) {
	loader := &YAMLPipelineLoader{}
	repoRoot := filepath.Join("..", "..")

	tests := []struct {
		file             string
		forbiddenPersona string
	}{
		{"gt-research.yaml", "github-analyst"},
		{"gt-research.yaml", "github-commenter"},
		{"gt-refresh.yaml", "github-analyst"},
		{"gt-refresh.yaml", "github-enhancer"},
		{"gt-rewrite.yaml", "github-analyst"},
		{"gt-rewrite.yaml", "github-enhancer"},
	}

	for _, tc := range tests {
		t.Run(tc.file+"_no_"+tc.forbiddenPersona, func(t *testing.T) {
			path := filepath.Join(repoRoot, ".wave", "pipelines", tc.file)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tc.file, err)
			}

			p, err := loader.Unmarshal(data)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tc.file, err)
			}

			for _, step := range p.Steps {
				if step.Persona == tc.forbiddenPersona {
					t.Errorf("step %q in %s uses forbidden persona %q", step.ID, tc.file, tc.forbiddenPersona)
				}
			}
		})
	}
}

// TestMultiplatformPipelinesShareContracts verifies that gl-* and gt-* pipelines
// reuse the same contract schema paths as their gh-* counterparts.
func TestMultiplatformPipelinesShareContracts(t *testing.T) {
	loader := &YAMLPipelineLoader{}
	repoRoot := filepath.Join("..", "..")

	// Load gh-implement as reference
	ghData, err := os.ReadFile(filepath.Join(repoRoot, ".wave", "pipelines", "gh-implement.yaml"))
	if err != nil {
		t.Fatalf("failed to read gh-implement.yaml: %v", err)
	}
	ghPipeline, err := loader.Unmarshal(ghData)
	if err != nil {
		t.Fatalf("failed to parse gh-implement.yaml: %v", err)
	}

	// Build map of step ID -> schema paths from gh-implement
	ghSchemas := make(map[string]string)
	for _, step := range ghPipeline.Steps {
		if step.Handover.Contract.SchemaPath != "" {
			ghSchemas[step.ID] = step.Handover.Contract.SchemaPath
		}
	}

	// Verify gl-implement uses same schemas
	variants := []string{"gl-implement.yaml", "gt-implement.yaml"}
	for _, variant := range variants {
		t.Run(variant, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(repoRoot, ".wave", "pipelines", variant))
			if err != nil {
				t.Fatalf("failed to read %s: %v", variant, err)
			}
			p, err := loader.Unmarshal(data)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", variant, err)
			}

			for _, step := range p.Steps {
				if step.Handover.Contract.SchemaPath == "" {
					continue
				}
				// Map create-mr back to create-pr for comparison
				stepID := step.ID
				if stepID == "create-mr" {
					stepID = "create-pr"
				}
				expected, ok := ghSchemas[stepID]
				if !ok {
					continue
				}
				if step.Handover.Contract.SchemaPath != expected {
					t.Errorf("step %q in %s uses schema %q, expected %q (same as gh-implement)", step.ID, variant, step.Handover.Contract.SchemaPath, expected)
				}
			}
		})
	}
}
