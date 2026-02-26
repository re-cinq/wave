package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// findProjectRoot walks up from the current working directory until it finds
// wave.yaml, which marks the project root.
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "wave.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (wave.yaml)")
		}
		dir = parent
	}
}

// pipelinePath returns the absolute path to a pipeline YAML file under
// .wave/pipelines/ in the project root.
func pipelinePath(t *testing.T, root, name string) string {
	t.Helper()
	return filepath.Join(root, ".wave", "pipelines", name)
}

// pipelineTestCase describes the expected properties of a single dead-code
// pipeline YAML file.
type pipelineTestCase struct {
	file     string
	name     string
	steps    int
	stepIDs  []string
	personas map[string]string // step ID -> expected persona
}

// deadCodeTestCases returns the table of expected properties for all four
// dead-code pipeline files.
func deadCodeTestCases() []pipelineTestCase {
	return []pipelineTestCase{
		{
			file:    "dead-code.yaml",
			name:    "dead-code",
			steps:   2,
			stepIDs: []string{"scan", "format"},
			personas: map[string]string{
				"scan":   "navigator",
				"format": "summarizer",
			},
		},
		{
			file:    "dead-code-pr.yaml",
			name:    "dead-code-pr",
			steps:   3,
			stepIDs: []string{"scan", "format", "publish-pr-comment"},
			personas: map[string]string{
				"scan":               "navigator",
				"format":             "summarizer",
				"publish-pr-comment": "github-commenter",
			},
		},
		{
			file:    "dead-code-issue.yaml",
			name:    "dead-code-issue",
			steps:   3,
			stepIDs: []string{"scan", "format", "create-github-issue"},
			personas: map[string]string{
				"scan":                "navigator",
				"format":              "summarizer",
				"create-github-issue": "github-commenter",
			},
		},
		{
			file:    "dead-code-heal.yaml",
			name:    "dead-code-heal",
			steps:   4,
			stepIDs: []string{"scan", "clean", "verify", "create-pr"},
			personas: map[string]string{
				"scan":      "navigator",
				"clean":     "craftsman",
				"verify":    "auditor",
				"create-pr": "craftsman",
			},
		},
	}
}

func TestDeadCodePipelines_Load(t *testing.T) {
	root := findProjectRoot(t)
	loader := &YAMLPipelineLoader{}

	for _, tc := range deadCodeTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			path := pipelinePath(t, root, tc.file)
			p, err := loader.Load(path)
			if err != nil {
				t.Fatalf("failed to load %s: %v", tc.file, err)
			}

			if p.Kind != "WavePipeline" {
				t.Errorf("Kind = %q, want %q", p.Kind, "WavePipeline")
			}

			if p.Metadata.Name != tc.name {
				t.Errorf("Metadata.Name = %q, want %q", p.Metadata.Name, tc.name)
			}

			if !p.Metadata.Release {
				t.Errorf("Metadata.Release = false, want true")
			}

			if len(p.Steps) != tc.steps {
				t.Fatalf("len(Steps) = %d, want %d", len(p.Steps), tc.steps)
			}

			for i, wantID := range tc.stepIDs {
				if p.Steps[i].ID != wantID {
					t.Errorf("Steps[%d].ID = %q, want %q", i, p.Steps[i].ID, wantID)
				}
			}
		})
	}
}

func TestDeadCodePipelines_DAGValid(t *testing.T) {
	root := findProjectRoot(t)
	loader := &YAMLPipelineLoader{}
	validator := &DAGValidator{}

	for _, tc := range deadCodeTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			path := pipelinePath(t, root, tc.file)
			p, err := loader.Load(path)
			if err != nil {
				t.Fatalf("failed to load %s: %v", tc.file, err)
			}

			if err := validator.ValidateDAG(p); err != nil {
				t.Errorf("DAG validation failed for %s: %v", tc.file, err)
			}
		})
	}
}

func TestDeadCodePipelines_StepPersonas(t *testing.T) {
	root := findProjectRoot(t)
	loader := &YAMLPipelineLoader{}

	for _, tc := range deadCodeTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			path := pipelinePath(t, root, tc.file)
			p, err := loader.Load(path)
			if err != nil {
				t.Fatalf("failed to load %s: %v", tc.file, err)
			}

			for _, step := range p.Steps {
				if step.Persona == "" {
					t.Errorf("step %q has empty persona", step.ID)
				}

				wantPersona, ok := tc.personas[step.ID]
				if !ok {
					t.Errorf("step %q not listed in expected personas", step.ID)
					continue
				}
				if step.Persona != wantPersona {
					t.Errorf("step %q persona = %q, want %q", step.ID, step.Persona, wantPersona)
				}
			}
		})
	}
}

func TestDeadCodePipelines_RequiresTools(t *testing.T) {
	root := findProjectRoot(t)
	loader := &YAMLPipelineLoader{}

	requiresGH := map[string]bool{
		"dead-code-pr.yaml":    true,
		"dead-code-issue.yaml": true,
	}

	noGH := map[string]bool{
		"dead-code.yaml":      true,
		"dead-code-heal.yaml": true,
	}

	// Pipelines that should require "gh".
	for file := range requiresGH {
		t.Run(file+"_requires_gh", func(t *testing.T) {
			path := pipelinePath(t, root, file)
			p, err := loader.Load(path)
			if err != nil {
				t.Fatalf("failed to load %s: %v", file, err)
			}

			if p.Requires == nil {
				t.Fatalf("Requires is nil, expected tools containing gh")
			}

			found := false
			for _, tool := range p.Requires.Tools {
				if tool == "gh" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Requires.Tools = %v, expected to contain %q", p.Requires.Tools, "gh")
			}
		})
	}

	// Pipelines that should NOT require "gh".
	for file := range noGH {
		t.Run(file+"_no_gh", func(t *testing.T) {
			path := pipelinePath(t, root, file)
			p, err := loader.Load(path)
			if err != nil {
				t.Fatalf("failed to load %s: %v", file, err)
			}

			if p.Requires != nil {
				for _, tool := range p.Requires.Tools {
					if tool == "gh" {
						t.Errorf("Requires.Tools contains %q but should not", "gh")
					}
				}
			}
		})
	}
}

func TestDeadCodePipelines_ScanStepContract(t *testing.T) {
	root := findProjectRoot(t)
	loader := &YAMLPipelineLoader{}

	for _, tc := range deadCodeTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			path := pipelinePath(t, root, tc.file)
			p, err := loader.Load(path)
			if err != nil {
				t.Fatalf("failed to load %s: %v", tc.file, err)
			}

			// Find the scan step.
			var scanStep *Step
			for i := range p.Steps {
				if p.Steps[i].ID == "scan" {
					scanStep = &p.Steps[i]
					break
				}
			}
			if scanStep == nil {
				t.Fatal("no step with ID \"scan\" found")
			}

			contract := scanStep.Handover.Contract
			if contract.Type != "json_schema" {
				t.Errorf("scan contract type = %q, want %q", contract.Type, "json_schema")
			}

			wantSchemaPath := ".wave/contracts/dead-code-scan.schema.json"
			if contract.SchemaPath != wantSchemaPath {
				t.Errorf("scan contract schema_path = %q, want %q", contract.SchemaPath, wantSchemaPath)
			}
		})
	}
}
