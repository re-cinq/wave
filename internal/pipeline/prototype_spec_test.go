//go:build integration
// +build integration

// Integration tests for prototype spec phase.
// Run with: go test -tags=integration ./internal/pipeline/...
// These tests require proper mock adapter setup with schema-compliant JSON output.

package pipeline

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"gopkg.in/yaml.v3"
)

func TestPrototypeSpecPhaseInitialization(t *testing.T) {
	// Test that spec phase can be initialized and executed
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "valid project description",
			input:       "Build a task management CLI tool with user authentication",
			expectError: false,
		},
		{
			name:        "minimal project description",
			input:       "Simple file organizer",
			expectError: false,
		},
		{
			name:        "empty input",
			input:       "",
			expectError: false, // executor does not reject empty input; pipelines may not require it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test manifest and load pipeline BEFORE changing directory
			testManifest := createPrototypeTestManifest()
			pipeline, err := loadTestPrototypePipeline()
			if err != nil {
				t.Fatalf("Failed to load prototype pipeline: %v", err)
			}

			// Trim to spec step only
			specStep := findStepByID(pipeline, "spec")
			if specStep == nil {
				t.Fatal("Spec step not found in pipeline")
			}
			specOnly := *specStep
			specOnly.Dependencies = nil
			pipeline.Steps = []Step{specOnly}

			// Change to project root for schema file access during execution
			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir("../.."); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(originalWd)

			// Create mock adapter for testing
			mockAdapter := adapter.NewMockAdapter()

			// Create test executor
			emitter := event.NewNDJSONEmitter()
			executor := NewDefaultPipelineExecutor(mockAdapter,
				WithEmitter(emitter),
				WithDebug(true),
			)

			// Create test context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Execute spec phase only (first step)
			err = executor.Execute(ctx, pipeline, testManifest, tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for input %q, but got none", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for input %q: %v", tt.input, err)
			}

			// Execute succeeded without error — spec phase completed
		})
	}
}

func TestPrototypeSpecPhaseArtifacts(t *testing.T) {
	// Test that spec phase produces required artifacts

	// Create test environment and load pipeline BEFORE changing directory
	testManifest := createPrototypeTestManifest()
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	// Trim to spec step only — this test validates spec phase artifacts, not the full pipeline
	specStep := findStepByID(pipeline, "spec")
	if specStep == nil {
		t.Fatal("Spec step not found in pipeline")
	}
	specOnly := *specStep
	specOnly.Dependencies = nil
	pipeline.Steps = []Step{specOnly}

	// Change to project root for schema file access during execution
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWd)

	mockAdapter := adapter.NewMockAdapter()
	emitter := event.NewNDJSONEmitter()
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(emitter),
		WithDebug(true),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute spec phase
	input := "Build a web application for team collaboration"
	err = executor.Execute(ctx, pipeline, testManifest, input)
	if err != nil {
		t.Fatalf("Spec phase execution failed: %v", err)
	}

	// Verify output artifacts are properly configured
	if len(specStep.OutputArtifacts) == 0 {
		t.Error("Spec step has no output artifacts configured")
	}

	// Verify contract configuration
	if specStep.Handover.Contract.Type != "json_schema" {
		t.Errorf("Expected json_schema contract, got %s", specStep.Handover.Contract.Type)
	}

	expectedSchemaPath := ".wave/contracts/spec-phase.schema.json"
	if specStep.Handover.Contract.SchemaPath != expectedSchemaPath {
		t.Errorf("Expected schema path %s, got %s", expectedSchemaPath, specStep.Handover.Contract.SchemaPath)
	}
}

func TestPrototypeSpecPhaseWorkspaceIsolation(t *testing.T) {
	// Test that spec phase runs in isolated workspace
	_ = createPrototypeTestManifest()
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	// Verify workspace configuration
	specStep := findStepByID(pipeline, "spec")
	if specStep == nil {
		t.Fatal("Spec step not found in pipeline")
	}

	// Check workspace mount configuration
	if len(specStep.Workspace.Mount) == 0 {
		t.Error("Spec step has no workspace mounts configured")
	}

	// Verify readwrite access for spec generation
	hasReadWriteMount := false
	for _, mount := range specStep.Workspace.Mount {
		if mount.Mode == "readwrite" {
			hasReadWriteMount = true
			break
		}
	}

	if !hasReadWriteMount {
		t.Error("Spec step requires readwrite workspace mount")
	}
}

// Helper functions

func createPrototypeTestManifest() *manifest.Manifest {
	return &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata: manifest.Metadata{
			Name:        "test-prototype",
			Description: "Test manifest for prototype pipeline",
		},
		Adapters: map[string]manifest.Adapter{
			"claude": {
				Binary:      "claude",
				Mode:        "headless",
				OutputFormat: "json",
				DefaultPermissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Write", "Edit"},
					Deny:         []string{},
				},
			},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:          "claude",
				Description:      "Codebase navigation and analysis",
				SystemPromptFile: ".wave/personas/navigator.md",
				Temperature:      0.1,
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Glob", "Grep"},
					Deny:         []string{"Write(*)", "Edit(*)"},
				},
			},
			"craftsman": {
				Adapter:          "claude",
				Description:      "Implementation specialist",
				SystemPromptFile: ".wave/personas/craftsman.md",
				Temperature:      0.7,
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
					Deny:         []string{},
				},
			},
			"philosopher": {
				Adapter:          "claude",
				Description:      "Documentation and design specialist",
				SystemPromptFile: ".wave/personas/philosopher.md",
				Temperature:      0.5,
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Write", "Edit", "Glob", "Grep"},
					Deny:         []string{"Bash(*)"},
				},
			},
		},
		Runtime: manifest.Runtime{
			DefaultTimeoutMin: 30,
			WorkspaceRoot:     ".wave/workspaces",
		},
	}
}

func loadTestPrototypePipeline() (*Pipeline, error) {
	// Load the actual prototype pipeline definition
	// Navigate up to project root from internal/pipeline directory
	pipelineData, err := os.ReadFile("../../.wave/pipelines/prototype.yaml")
	if err != nil {
		return nil, err
	}

	var pipeline Pipeline
	if err := yaml.Unmarshal(pipelineData, &pipeline); err != nil {
		return nil, err
	}

	return &pipeline, nil
}

func findStepByID(pipeline *Pipeline, id string) *Step {
	for i := range pipeline.Steps {
		if pipeline.Steps[i].ID == id {
			return &pipeline.Steps[i]
		}
	}
	return nil
}