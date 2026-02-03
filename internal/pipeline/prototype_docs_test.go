//go:build integration
// +build integration

// Integration tests for prototype docs phase.
// Run with: go test -tags=integration ./internal/pipeline/...

package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
)

func TestPrototypeDocsPhaseInitialization(t *testing.T) {
	// Test that docs phase can be initialized and executed with spec dependency
	tests := []struct {
		name         string
		hasSpecPhase bool
		expectError  bool
	}{
		{
			name:         "docs phase with completed spec phase",
			hasSpecPhase: true,
			expectError:  false,
		},
		{
			name:         "docs phase without spec phase",
			hasSpecPhase: false,
			expectError:  true, // Should fail due to missing dependency
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

			// Change to project root for schema file access during execution
			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir("../.."); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(originalWd)

			// Setup test environment
			tempDir := t.TempDir()

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

			if tt.hasSpecPhase {
				// Create mock spec phase artifacts in workspace
				specWorkspace := filepath.Join(tempDir, "spec")
				err = os.MkdirAll(specWorkspace, 0755)
				if err != nil {
					t.Fatalf("Failed to create spec workspace: %v", err)
				}

				// Create mock spec.md file
				specContent := `# Test Feature Specification

## Overview
This is a test feature for validating the docs phase pipeline.

## User Stories
- As a user, I want to test the docs phase functionality
- As a developer, I want to ensure the pipeline works correctly

## Functional Requirements
1. The docs phase must process the spec phase output
2. The docs phase must generate feature documentation
3. The docs phase must create stakeholder summaries

## Success Criteria
- Documentation is generated successfully
- Stakeholder summary is comprehensible
- All spec requirements are covered in documentation
`
				err = os.WriteFile(filepath.Join(specWorkspace, "spec.md"), []byte(specContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create spec.md: %v", err)
				}

				// Create mock artifact.json for spec phase
				specArtifact := `{
  "phase": "spec",
  "artifacts": {
    "spec": {"path": "spec.md", "exists": true, "content_type": "markdown"}
  },
  "validation": {
    "specification_quality": "good"
  },
  "metadata": {
    "timestamp": "2026-02-03T10:30:00Z",
    "input_description": "Test feature for docs phase validation"
  }
}`
				err = os.WriteFile(filepath.Join(specWorkspace, "artifact.json"), []byte(specArtifact), 0644)
				if err != nil {
					t.Fatalf("Failed to create spec artifact.json: %v", err)
				}
			}

			// Execute docs phase only (second step) - would need spec phase first in real execution
			input := "Build a web application for team collaboration (docs phase test)"
			err = executor.Execute(ctx, pipeline, testManifest, input)

			if tt.expectError && err == nil {
				t.Error("Expected error for docs phase without spec phase, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for docs phase: %v", err)
			}

			// If no error expected and spec phase exists, verify docs phase configuration
			if !tt.expectError && tt.hasSpecPhase {
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
		})
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