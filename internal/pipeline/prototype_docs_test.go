//go:build integration
// +build integration

// Integration tests for prototype docs phase.
// Run with: go test -tags=integration ./internal/pipeline/...

package pipeline

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
)

func TestPrototypeDocsPhaseInitialization(t *testing.T) {
	// Test that docs phase can be initialized and executed
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "docs phase with completed spec phase",
			input:       "Build a web application for team collaboration (docs phase test)",
			expectError: false,
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

			// Trim to docs step only
			docsStep := findStepByID(pipeline, "docs")
			if docsStep == nil {
				t.Fatal("Docs step not found in pipeline")
			}
			docsOnly := *docsStep
			docsOnly.Dependencies = nil
			pipeline.Steps = []Step{docsOnly}

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

			err = executor.Execute(ctx, pipeline, testManifest, tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for input %q, but got none", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for input %q: %v", tt.input, err)
			}
		})
	}
}

func TestPrototypeDocsPhaseConfiguration(t *testing.T) {
	// Verify docs phase has correct dependencies, persona, and artifact injection
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	docsStep := findStepByID(pipeline, "docs")
	if docsStep == nil {
		t.Fatal("Docs step not found in pipeline")
	}

	// Verify dependency on spec phase
	expectedDependencies := []string{"spec"}
	if len(docsStep.Dependencies) != len(expectedDependencies) {
		t.Errorf("Expected %d dependencies, got %d", len(expectedDependencies), len(docsStep.Dependencies))
	}

	// Verify artifact injection is configured
	if len(docsStep.Memory.InjectArtifacts) == 0 {
		t.Error("Docs step has no artifact injection configured")
	}

	// Verify persona is correct
	if docsStep.Persona != "philosopher" {
		t.Errorf("Expected philosopher persona, got %s", docsStep.Persona)
	}
}

func TestPrototypeDocsPhaseArtifactInjection(t *testing.T) {
	// Test that docs phase properly injects artifacts from spec phase
	_ = createPrototypeTestManifest()
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	// Find docs step
	docsStep := findStepByID(pipeline, "docs")
	if docsStep == nil {
		t.Fatal("Docs step not found in pipeline")
	}

	// Verify artifact injection configuration
	if len(docsStep.Memory.InjectArtifacts) == 0 {
		t.Fatal("No artifact injection configured for docs phase")
	}

	// Verify spec artifact injection
	found := false
	for _, injection := range docsStep.Memory.InjectArtifacts {
		if injection.Step == "spec" && injection.Artifact == "spec" && injection.As == "input-spec.md" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Spec artifact injection not properly configured")
	}

	// Verify output artifacts configuration
	expectedArtifacts := map[string]string{
		"feature-docs":        "feature-docs.md",
		"stakeholder-summary": "stakeholder-summary.md",
		"contract_data":       "artifact.json",
	}

	if len(docsStep.OutputArtifacts) != len(expectedArtifacts) {
		t.Errorf("Expected %d output artifacts, got %d", len(expectedArtifacts), len(docsStep.OutputArtifacts))
	}

	for _, artifact := range docsStep.OutputArtifacts {
		expectedPath, exists := expectedArtifacts[artifact.Name]
		if !exists {
			t.Errorf("Unexpected artifact: %s", artifact.Name)
			continue
		}

		if artifact.Path != expectedPath {
			t.Errorf("Expected path %s for artifact %s, got %s", expectedPath, artifact.Name, artifact.Path)
		}
	}
}

func TestPrototypeDocsPhaseContractValidation(t *testing.T) {
	// Test that docs phase has proper contract validation configured
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	docsStep := findStepByID(pipeline, "docs")
	if docsStep == nil {
		t.Fatal("Docs step not found in pipeline")
	}

	// Verify contract configuration
	if docsStep.Handover.Contract.Type != "json_schema" {
		t.Errorf("Expected json_schema contract, got %s", docsStep.Handover.Contract.Type)
	}

	expectedSchemaPath := ".wave/contracts/docs-phase.schema.json"
	if docsStep.Handover.Contract.SchemaPath != expectedSchemaPath {
		t.Errorf("Expected schema path %s, got %s", expectedSchemaPath, docsStep.Handover.Contract.SchemaPath)
	}

	if !docsStep.Handover.Contract.MustPass {
		t.Error("Contract validation should be required (must_pass: true)")
	}

	if docsStep.Handover.Contract.MaxRetries <= 0 {
		t.Error("Contract should allow retries for robustness")
	}
}

func TestPrototypeDocsPhaseWorkspaceConfiguration(t *testing.T) {
	// Test that docs phase has proper workspace configuration
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	docsStep := findStepByID(pipeline, "docs")
	if docsStep == nil {
		t.Fatal("Docs step not found in pipeline")
	}

	// Verify workspace mount configuration
	if len(docsStep.Workspace.Mount) == 0 {
		t.Error("Docs step has no workspace mounts configured")
	}

	// Verify readwrite access for documentation generation
	hasReadWriteMount := false
	for _, mount := range docsStep.Workspace.Mount {
		if mount.Mode == "readwrite" {
			hasReadWriteMount = true
			break
		}
	}

	if !hasReadWriteMount {
		t.Error("Docs step requires readwrite workspace mount for file creation")
	}
}